package graterm

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"
)

// defaultSignals is a default set of signals to handle.
var defaultSignals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}

// TerminationOrder is an application components termination order.
//
// Lower order - higher priority.
type TerminationOrder int

type terminationFunc struct {
	componentName string
	timeout       time.Duration
	hookFunc      func(ctx context.Context)
}

// Stopper is a component stopper that executes registered termination hooks in a specified order.
type Stopper struct {
	hooksMx *sync.Mutex
	hooks   map[TerminationOrder][]terminationFunc

	wg *sync.WaitGroup

	cancelFunc context.CancelFunc // todo check later on if this needed?

	log Logger
}

// NewWithDefaultSignals creates a new instance of component stopper.
// Invokes withSignals with syscall.SIGINT and syscall.SIGTERM as default signals.
//
// If the log parameter is nil, then noop logger will be used.
//
// Note: this method will start internal monitoring goroutine.
func NewWithDefaultSignals(appCtx context.Context, log Logger) (*Stopper, context.Context) {
	return NewWithSignals(appCtx, log, defaultSignals...)
}

// NewWithSignals creates a new instance of component stopper.
//
// If the log parameter is nil, then noop logger will be used.
//
// Note: this method will start internal monitoring goroutine.
func NewWithSignals(appCtx context.Context, log Logger, sig ...os.Signal) (*Stopper, context.Context) {
	if log == nil {
		log = noopLogger{}
	}
	chSignals := make(chan os.Signal, 1)
	ctx, cancel := withSignals(appCtx, chSignals, sig...)
	return &Stopper{
		hooksMx:    &sync.Mutex{},
		hooks:      make(map[TerminationOrder][]terminationFunc),
		wg:         &sync.WaitGroup{},
		cancelFunc: cancel,
		log:        log,
	}, ctx
}

// withSignals return a copy of the parent context that will be canceled by signal.
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

// Register registers termination hook with priority and human-readable name.
// The lower the order the higher the execution priority, the earlier it will be executed.
// If there are multiple hooks with the same order they will be executed in parallel.
func (s *Stopper) Register(order TerminationOrder, componentName string, timeout time.Duration, hookFunc func(ctx context.Context)) {
	comm := terminationFunc{
		componentName: componentName,
		timeout:       timeout,
		hookFunc:      hookFunc,
	}
	s.hooksMx.Lock()
	defer s.hooksMx.Unlock()

	s.hooks[order] = append(s.hooks[order], comm)
}

// Wait waits (with timeout) for Stopper to finish termination after the ctx is done.
func (s *Stopper) Wait(ctx context.Context, timeout time.Duration) error {
	{
		s.wg.Add(1)
		go s.waitShutdown(ctx)
	}

	// block till the end of the app:
	<-ctx.Done()

	wgChan := waitWG(s.wg)

	select {
	case <-time.After(timeout):
		return errors.New("graterm.WaitGroup is timed out") // todo change error text
	case <-wgChan:
		return nil
	}
}

// waitWG returns a chan that will be closed once wg is done.
func waitWG(wg *sync.WaitGroup) <-chan struct{} {
	c := make(chan struct{})

	go func() {
		defer close(c)
		wg.Wait()
	}()

	return c
}

// waitShutdown waits for the context to be done and then sequentially notifies existing shutdown hooks.
func (s *Stopper) waitShutdown(appCtx context.Context) {
	defer s.wg.Done()

	<-appCtx.Done() // Block until application context is done

	s.hooksMx.Lock()
	defer s.hooksMx.Unlock()

	order := make([]int, 0, len(s.hooks))
	for k := range s.hooks {
		order = append(order, int(k))
	}
	sort.Ints(order)

	for _, o := range order {
		runWg := sync.WaitGroup{}

		for _, c := range s.hooks[TerminationOrder(o)] {
			runWg.Add(1)

			go func(f terminationFunc) {
				// todo missing panic recovery
				defer runWg.Done()

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				t := time.NewTimer(f.timeout)
				doneCh := make(chan struct{})

				go func() {
					// todo missing panic recovery
					defer close(doneCh)
					f.hookFunc(ctx)
				}()

				select {
				case <-t.C:
					cancel()
					// proceed to the next command
					s.log.Printf("timeout %v for component: %q is over, hook wasn't finished yet - continue to the next component",
						f.timeout, f.componentName)
				case <-doneCh:
					t.Stop() // we don't care if there's anything left in the 't.C' channel
					// proceed to the next command
					s.log.Printf("component: %q finished termination", f.componentName)
				}
			}(c)
		}
		runWg.Wait()
	}
}
