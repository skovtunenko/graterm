package graterm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"sync"
	"time"
)

// TerminationOrder is an application components termination order.
//
// Lower order - higher priority.
type TerminationOrder int

type terminationFunc struct {
	terminator *Terminator

	order         TerminationOrder
	componentName string
	timeout       time.Duration
	hookFunc      func(ctx context.Context)
}

// Terminator is a component terminator that executes registered termination hooks in a specified order.
type Terminator struct {
	hooksMx *sync.Mutex
	hooks   map[TerminationOrder][]terminationFunc

	wg *sync.WaitGroup

	cancelFunc context.CancelFunc

	log Logger
}

// NewWithSignals creates a new instance of component Terminator.
//
// Example of useful signals might be: syscall.SIGINT, syscall.SIGTERM.
//
// Note: this method will start internal monitoring goroutine.
func NewWithSignals(appCtx context.Context, sig ...os.Signal) (*Terminator, context.Context) {
	chSignals := make(chan os.Signal, 1)
	ctx, cancel := withSignals(appCtx, chSignals, sig...)
	return &Terminator{
		hooksMx:    &sync.Mutex{},
		hooks:      make(map[TerminationOrder][]terminationFunc),
		wg:         &sync.WaitGroup{},
		cancelFunc: cancel,
		log:        noopLogger{},
	}, ctx
}

// withSignals return a copy of the parent context that will be canceled by signal(s).
// If no signals are provided, any incoming signal will cause cancel.
// Otherwise, just the provided signals will.
//
// Note: this method will start internal monitoring goroutine.
func withSignals(ctx context.Context, chSignals chan os.Signal, sig ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	signal.Notify(chSignals, sig...)

	// function invoke cancel once a signal arrived OR parent context is done:
	go func() {
		defer cancel()

		select {
		case <-chSignals:
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}

// SetLogger sets the logger implementation.
//
// If log is nil, then NOOP logger implementation will be used.
func (s *Terminator) SetLogger(log Logger) {
	if log == nil {
		log = noopLogger{}
	}

	s.hooksMx.Lock()
	defer s.hooksMx.Unlock()
	s.log = log
}

func (s *Terminator) WithOrder(order TerminationOrder) *terminationFunc {
	return &terminationFunc{
		terminator: s,
		order:      order,
	}
}

func (tf *terminationFunc) WithName(componentName string) *terminationFunc {
	tf.componentName = componentName
	return tf
}

func (tf *terminationFunc) Register(timeout time.Duration, hookFunc func(ctx context.Context)) {
	tf.timeout = timeout
	tf.hookFunc = hookFunc

	tf.terminator.hooksMx.Lock()
	defer tf.terminator.hooksMx.Unlock()
	tf.terminator.hooks[tf.order] = append(tf.terminator.hooks[tf.order], *tf)
}

// Register registers termination hook with priority and human-readable name.
// The lower the order the higher the execution priority, the earlier it will be executed.
// If there are multiple hooks with the same order they will be executed in parallel.
//
// Deprecated: use WithOrder instead.
func (s *Terminator) Register(order TerminationOrder, componentName string, timeout time.Duration, hookFunc func(ctx context.Context)) {
	comm := terminationFunc{
		componentName: componentName,
		timeout:       timeout,
		hookFunc:      hookFunc,
	}

	s.hooksMx.Lock()
	defer s.hooksMx.Unlock()
	s.hooks[order] = append(s.hooks[order], comm)
}

// Wait waits (with timeout) for Terminator to finish termination after the appCtx is done.
func (s *Terminator) Wait(appCtx context.Context, timeout time.Duration) error {
	{
		s.wg.Add(1)
		go s.waitShutdown(appCtx)
	}

	<-appCtx.Done()

	wgChan := make(chan struct{})
	go func() {
		defer close(wgChan)
		s.wg.Wait()
	}()

	select {
	case <-time.After(timeout):
		return fmt.Errorf("termination timed out after %v", timeout)
	case <-wgChan:
		return nil
	}
}

// waitShutdown waits for the context to be done and then sequentially notifies existing shutdown hooks.
func (s *Terminator) waitShutdown(appCtx context.Context) {
	defer s.wg.Done()

	<-appCtx.Done() // Block until application context is done (most likely, when the registered os.Signal will be received)

	s.hooksMx.Lock()
	defer s.hooksMx.Unlock()

	order := make([]int, 0, len(s.hooks))
	for k := range s.hooks {
		order = append(order, int(k))
	}
	sort.Ints(order)

	for _, o := range order {
		o := o

		runWg := sync.WaitGroup{}

		for _, c := range s.hooks[TerminationOrder(o)] {
			runWg.Add(1)

			go func(f terminationFunc) {
				defer runWg.Done()

				ctx, cancel := context.WithTimeout(context.Background(), f.timeout)
				defer cancel()

				go func() {
					defer func() {
						defer cancel()

						if err := recover(); err != nil {
							s.log.Printf("registered hook for component: %q (order: %d) panicked, recovered: %+v",
								f.componentName, o, err)
						}
					}()

					f.hookFunc(ctx)
				}()

				<-ctx.Done() // block until the hookFunc is over OR timeout has been expired

				switch err := ctx.Err(); {
				case errors.Is(err, context.DeadlineExceeded):
					s.log.Printf("registered hook for component: %q (order: %d) timed out after %v", f.componentName, o, f.timeout)
				case errors.Is(err, context.Canceled):
					s.log.Printf("registered hook for component: %q (order: %d) finished termination in time", f.componentName, o)
				}
			}(c)
		}

		runWg.Wait()
	}
}
