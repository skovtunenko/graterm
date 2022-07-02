package graterm

import (
	"context"
	"github.com/stretchr/testify/require"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestNewWithDefaultSignals(t *testing.T) {
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	got, ctx := NewWithDefaultSignals(rootCtx)
	require.NotNil(t, got)
	require.NotNil(t, ctx)

	require.NotNil(t, got.hooksMx)
	require.NotNil(t, got.hooks)
	require.NotNil(t, got.wg)
	require.NotNil(t, got.cancelFunc)
	require.NotNil(t, got.log)
}

func TestStopper_Register(t *testing.T) {
	t.Parallel()

	t.Run("add_only_one_hook", func(t *testing.T) {
		rootCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		s, ctx := NewWithDefaultSignals(rootCtx)
		require.NotNil(t, ctx)

		s.Register(TerminationOrder(1), "Hook", 1*time.Second, func(_ context.Context) {})

		require.Equal(t, 1, len(s.hooks))
		got, ok := s.hooks[TerminationOrder(1)]
		require.True(t, ok)
		require.Equal(t, 1, len(got))
	})

	t.Run("add_with_different_order", func(t *testing.T) {
		rootCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		s, ctx := NewWithDefaultSignals(rootCtx)
		require.NotNil(t, ctx)

		s.Register(TerminationOrder(1), "Hook1", time.Second, func(_ context.Context) {})
		s.Register(TerminationOrder(2), "Hook2", time.Second, func(_ context.Context) {})

		require.Equal(t, 2, len(s.hooks))
		got, ok := s.hooks[TerminationOrder(1)]
		require.True(t, ok)
		require.Equal(t, 1, len(got))

		got2, ok2 := s.hooks[TerminationOrder(2)]
		require.True(t, ok2)
		require.Equal(t, 1, len(got2))
	})

	t.Run("add_with_the_same_order", func(t *testing.T) {
		rootCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		s, ctx := NewWithDefaultSignals(rootCtx)
		require.NotNil(t, ctx)

		s.Register(TerminationOrder(1), "Hook1", time.Second, func(_ context.Context) {})
		s.Register(TerminationOrder(1), "Hook2", time.Second, func(_ context.Context) {})

		require.Equal(t, 1, len(s.hooks))
		got, ok := s.hooks[TerminationOrder(1)]
		require.True(t, ok)
		require.Equal(t, 2, len(got))
	})
}

func TestStopper_waitShutdown(t *testing.T) {
	t.Run("execution_waits_for_the_context_to_be_done_before_proceeding", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		s, ctx := NewWithDefaultSignals(ctx)
		require.NotNil(t, ctx)

		i := 0
		s.Register(TerminationOrder(1), "Hook", time.Second, func(_ context.Context) { i = 1 })

		s.wg.Add(1)
		go s.waitShutdown(ctx)

		runtime.Gosched()
		require.Equal(t, 0, i)

		cancel()
		s.wg.Wait()
		require.Equal(t, 1, i)
	})

	t.Run("hooks_are_executed_in_order", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		s, ctx := NewWithDefaultSignals(ctx)
		require.NotNil(t, ctx)

		res := make([]int, 0, 4)
		s.Register(TerminationOrder(2), "Hook1", time.Second, func(_ context.Context) { res = append(res, 2) })
		s.Register(TerminationOrder(4), "Hook2", time.Second, func(_ context.Context) { res = append(res, 4) })
		s.Register(TerminationOrder(1), "Hook3", time.Second, func(_ context.Context) { res = append(res, 1) })
		s.Register(TerminationOrder(3), "Hook4", time.Second, func(_ context.Context) { res = append(res, 3) })

		cancel()

		s.wg.Add(1)
		go s.waitShutdown(ctx)

		s.wg.Wait()
		require.Equal(t, []int{1, 2, 3, 4}, res)
	})

	t.Run("timeouted_hooks_are_ignored", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		s, ctx := NewWithDefaultSignals(ctx)
		require.NotNil(t, ctx)

		t1Mx := sync.Mutex{}
		t1 := 0

		t2 := 0

		goLeakWg := sync.WaitGroup{}
		goLeakWg.Add(1)
		s.Register(TerminationOrder(1), "Hook1", time.Nanosecond, func(_ context.Context) {
			defer goLeakWg.Done()
			time.Sleep(500 * time.Millisecond)
			t1Mx.Lock()
			defer t1Mx.Unlock()
			t1 = 1
		})
		s.Register(TerminationOrder(2), "Hook2", time.Second, func(_ context.Context) { t2 = 1 })

		cancel()

		s.wg.Add(1)
		go s.waitShutdown(ctx)

		s.wg.Wait()

		t1Mx.Lock()
		require.Equal(t, 0, t1)
		require.Equal(t, 1, t2) // checking before Unlock to be sure that 2nd hook won't be just executed after unlocking
		t1Mx.Unlock()

		goLeakWg.Wait()
	})
}

func TestStopper_Wait(t *testing.T) {
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

			s, ctx := NewWithDefaultSignals(rootCtx)
			require.NotNil(t, ctx)

			s.wg.Add(1)

			done := make(chan struct{})
			var waitErr error
			go func() {
				waitErr = s.Wait(ctx, tt.timeout)
				close(done)
			}()

			cancel()
			if tt.wouldDone {
				s.wg.Done()
			} else {
				defer s.wg.Done()
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
