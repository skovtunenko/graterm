package graterm

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestNewWithSignals(t *testing.T) {
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	got, ctx := NewWithSignals(rootCtx, syscall.SIGINT, syscall.SIGTERM)
	require.NotNil(t, got)
	require.NotNil(t, ctx)

	require.NotNil(t, got.hooksMx)
	require.NotNil(t, got.hooks)
	require.NotNil(t, got.wg)
	require.NotNil(t, got.cancelFunc)
	require.NotNil(t, got.log)
}

func TestTerminator_Register(t *testing.T) {
	t.Parallel()

	t.Run("add_only_one_hook_with_negative_timeout", func(t *testing.T) {
		rootCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		terminator, ctx := NewWithSignals(rootCtx, syscall.SIGINT, syscall.SIGTERM)
		require.NotNil(t, ctx)

		terminator.WithOrder(777).Register(-1, func(_ context.Context) {}) // negative timeout

		require.Equal(t, 1, len(terminator.hooks))
		got, ok := terminator.hooks[Order(777)]
		require.True(t, ok)
		require.Equal(t, 1, len(got))

		require.Equal(t, time.Minute, got[0].timeout)
	})

	t.Run("add_only_one_hook", func(t *testing.T) {
		rootCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		terminator, ctx := NewWithSignals(rootCtx, syscall.SIGINT, syscall.SIGTERM)
		require.NotNil(t, ctx)

		terminator.WithOrder(1).
			WithName("Hook").
			Register(1*time.Second, func(_ context.Context) {})

		require.Equal(t, 1, len(terminator.hooks))
		got, ok := terminator.hooks[Order(1)]
		require.True(t, ok)
		require.Equal(t, 1, len(got))
	})

	t.Run("add_with_different_order", func(t *testing.T) {
		rootCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		terminator, ctx := NewWithSignals(rootCtx, syscall.SIGINT, syscall.SIGTERM)
		require.NotNil(t, ctx)

		terminator.WithOrder(1).
			WithName("Hook1").
			Register(1*time.Second, func(_ context.Context) {})
		terminator.WithOrder(2).
			WithName("Hook2").
			Register(1*time.Second, func(_ context.Context) {})

		require.Equal(t, 2, len(terminator.hooks))
		got, ok := terminator.hooks[Order(1)]
		require.True(t, ok)
		require.Equal(t, 1, len(got))

		got2, ok2 := terminator.hooks[Order(2)]
		require.True(t, ok2)
		require.Equal(t, 1, len(got2))
	})

	t.Run("add_with_the_same_order", func(t *testing.T) {
		rootCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		terminator, ctx := NewWithSignals(rootCtx, syscall.SIGINT, syscall.SIGTERM)
		require.NotNil(t, ctx)

		terminator.WithOrder(1).
			WithName("Hook1").
			Register(1*time.Second, func(_ context.Context) {})
		terminator.WithOrder(1).
			WithName("Hook2").
			Register(1*time.Second, func(_ context.Context) {})

		require.Equal(t, 1, len(terminator.hooks))
		got, ok := terminator.hooks[Order(1)]
		require.True(t, ok)
		require.Equal(t, 2, len(got))
	})

	t.Run("panic_in_registered_hook", func(t *testing.T) {
		rootCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		terminator, ctx := NewWithSignals(rootCtx, syscall.SIGINT, syscall.SIGTERM)
		require.NotNil(t, ctx)

		terminator.WithOrder(1).
			WithName("Panicked Hook1").
			Register(1*time.Second, func(_ context.Context) { panic(errors.New("panic in Hook1")) })
		terminator.WithOrder(2).
			WithName("Panicked Hook2").
			Register(1*time.Second, func(_ context.Context) { panic(errors.New("panic in Hook2")) })
		require.Equal(t, 2, len(terminator.hooks))

		cancel()
		gotErr := terminator.Wait(ctx, time.Minute) // some long period of time
		require.NoError(t, gotErr)
	})
}

func TestTerminator_waitShutdown(t *testing.T) {
	t.Run("execution_waits_for_the_context_to_be_done_before_proceeding", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		terminator, ctx := NewWithSignals(ctx, syscall.SIGINT, syscall.SIGTERM)
		require.NotNil(t, ctx)

		i := 0
		terminator.WithOrder(1).WithName("Hook").Register(time.Second, func(_ context.Context) { i = 1 })

		terminator.wg.Add(1)
		go terminator.waitShutdown(ctx)

		runtime.Gosched()
		require.Equal(t, 0, i)

		cancel()
		terminator.wg.Wait()
		require.Equal(t, 1, i)
	})

	t.Run("hooks_are_executed_in_order", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		terminator, ctx := NewWithSignals(ctx, syscall.SIGINT, syscall.SIGTERM)
		require.NotNil(t, ctx)

		res := make([]int, 0, 4)
		terminator.WithOrder(2).WithName("Hook1").Register(time.Second, func(_ context.Context) { res = append(res, 2) })
		terminator.WithOrder(4).WithName("Hook2").Register(time.Second, func(_ context.Context) { res = append(res, 4) })
		terminator.WithOrder(1).WithName("Hook3").Register(time.Second, func(_ context.Context) { res = append(res, 1) })
		terminator.WithOrder(3).WithName("Hook4").Register(time.Second, func(_ context.Context) { res = append(res, 3) })

		cancel()

		terminator.wg.Add(1)
		go terminator.waitShutdown(ctx)

		terminator.wg.Wait()
		require.Equal(t, []int{1, 2, 3, 4}, res)
	})

	t.Run("timeouted_hooks_are_ignored", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		terminator, ctx := NewWithSignals(ctx, syscall.SIGINT, syscall.SIGTERM)
		require.NotNil(t, ctx)

		t1Mx := sync.Mutex{}
		t1 := 0

		t2 := 0

		goLeakWg := sync.WaitGroup{}
		goLeakWg.Add(1)
		terminator.WithOrder(1).WithName("Hook1").
			Register(time.Nanosecond, func(_ context.Context) {
				defer goLeakWg.Done()
				time.Sleep(500 * time.Millisecond)
				t1Mx.Lock()
				defer t1Mx.Unlock()
				t1 = 1
			})
		terminator.WithOrder(2).
			WithName("Hook2").
			Register(time.Second, func(_ context.Context) { t2 = 1 })

		cancel()

		terminator.wg.Add(1)
		go terminator.waitShutdown(ctx)

		terminator.wg.Wait()

		t1Mx.Lock()
		require.Equal(t, 0, t1)
		require.Equal(t, 1, t2) // checking before Unlock to be sure that 2nd hook won't be just executed after unlocking
		t1Mx.Unlock()

		goLeakWg.Wait()
	})
}

func TestTerminator_Wait(t *testing.T) {
	tests := []struct {
		name      string
		timeout   time.Duration
		wouldDone bool
		wantErr   bool
	}{
		{
			name:      "canceled",
			wouldDone: true,
			timeout:   time.Minute,
			wantErr:   false,
		},
		{
			name:      "timeout",
			wouldDone: false,
			timeout:   time.Millisecond,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			rootCtx, cancel := context.WithCancel(context.Background())
			defer cancel()

			terminator, ctx := NewWithSignals(rootCtx, syscall.SIGINT, syscall.SIGTERM)
			require.NotNil(t, ctx)

			terminator.wg.Add(1)

			done := make(chan struct{})
			var waitErr error
			go func() {
				waitErr = terminator.Wait(ctx, tt.timeout)
				close(done)
			}()

			cancel()
			if tt.wouldDone {
				terminator.wg.Done()
			} else {
				defer terminator.wg.Done()
			}

			select {
			case <-time.After(time.Second):
				t.Fatal("waiting group is done timed out")
			case <-done:
			}

			if tt.wantErr {
				require.Error(t, waitErr)
			} else {
				require.NoError(t, waitErr)
			}
		})
	}
}

func TestTerminator_restores_signals(t *testing.T) {
	if os.Getenv("HANG_ON_STOP") == "1" {
		terminator, ctx := NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		require.NotNil(t, ctx)

		terminator.WithOrder(0).Register(5*time.Second, func(ctx context.Context) {
			select {}
		})

		_ = terminator.Wait(ctx, time.Minute)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestTerminator_restores_signal")
	cmd.Env = append(cmd.Env, "HANG_ON_STOP=1")
	require.NoError(t, cmd.Start())

	time.Sleep(100 * time.Millisecond)
	require.NoError(t, cmd.Process.Signal(syscall.SIGINT))

	time.Sleep(100 * time.Millisecond)
	require.NoError(t, cmd.Process.Signal(syscall.SIGINT))

	var exitErr *exec.ExitError
	require.ErrorAs(t, cmd.Wait(), &exitErr)

	ws, ok := exitErr.Sys().(syscall.WaitStatus)
	require.Truef(t, ok, "exit error is not syscall.WaitStatus, but %T", exitErr.Sys())

	assert.True(t, ws.Signaled(), "process stopped not by signal")
	assert.Equal(t, syscall.SIGINT, ws.Signal(), "process stopped not by SIGINT")
}

func Test_withSignals(t *testing.T) {
	t.Run("termination_on_SIGHUP", func(t *testing.T) {
		rootCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		chSignals := make(chan os.Signal, 1)
		sig := syscall.SIGHUP
		defer signal.Stop(chSignals)

		gotCtx, gotCancelFunc := withSignals(rootCtx, chSignals, sig)
		err := syscall.Kill(syscall.Getpid(), sig)
		require.NoError(t, err)

		require.NotNil(t, gotCtx)
		require.NotNil(t, gotCancelFunc)

		require.NoError(t, gotCtx.Err())
	})
}

func TestTerminator_SetLogger(t *testing.T) {
	type args struct {
		log Logger
	}
	tests := []struct {
		name    string
		args    args
		wantLog Logger
	}{
		{
			name: "default_logger",
			args: args{
				log: log.Default(),
			},
			wantLog: log.Default(),
		},
		{
			name: "private_noop_logger",
			args: args{
				log: log.Default(),
			},
			wantLog: log.Default(),
		},
		{
			name: "nil_logger",
			args: args{
				log: nil,
			},
			wantLog: noopLogger{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			rootCtx, cancel := context.WithCancel(context.Background())
			defer cancel()

			got, ctx := NewWithSignals(rootCtx, os.Interrupt)
			require.NotNil(t, got)
			require.NotNil(t, ctx)

			got.SetLogger(tt.args.log)
			require.Equal(t, tt.wantLog, got.log)
		})
	}
}
