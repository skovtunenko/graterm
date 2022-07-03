package graterm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
	componentName string
	timeout       time.Duration
	hookFunc      func(ctx context.Context)
}

// Stopper is a component stopper that executes registered termination hooks in a specified order.
type Stopper struct {
	hooksMx *sync.Mutex
	hooks   map[TerminationOrder][]terminationFunc

	wg *sync.WaitGroup

	cancelFunc context.CancelFunc

	log Logger
}

// Server is a generic Server. Well-known implementation is a built-in http.Server.
// Interface was introduced for testing purposes mainly.
type Server interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

// NewWithSignals creates a new instance of component Stopper.
//
// Example of useful signals might be: syscall.SIGINT, syscall.SIGTERM.
//
// Note: this method will start internal monitoring goroutine.
func NewWithSignals(appCtx context.Context, sig ...os.Signal) (*Stopper, context.Context) {
	chSignals := make(chan os.Signal, 1)
	ctx, cancel := withSignals(appCtx, chSignals, sig...)
	return &Stopper{
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
// If log is nil, then NOOP logger will be used.
func (s *Stopper) SetLogger(log Logger) {
	if log == nil {
		log = noopLogger{}
	}

	s.hooksMx.Lock()
	defer s.hooksMx.Unlock()
	s.log = log
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

// Wait waits (with timeout) for Stopper to finish termination after the appCtx is done.
func (s *Stopper) Wait(appCtx context.Context, timeout time.Duration) error {
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
func (s *Stopper) waitShutdown(appCtx context.Context) {
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

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				t := time.NewTimer(f.timeout)

				go func() {
					defer func() {
						defer cancel()

						if err := recover(); err != nil {
							s.log.Printf("registered hook for component: %q (priority: %d) panicked, recovered: %+v",
								f.componentName, o, err)
						}
					}()

					f.hookFunc(ctx)
				}()

				select {
				case <-t.C:
					cancel()
					s.log.Printf("registered hook for component: %q (priority: %d) timed out after %v",
						f.componentName, o, f.timeout)
					// proceed to the next command (if any left)
				case <-ctx.Done():
					t.Stop() // we don't care if there's anything left in the 't.C' channel
					s.log.Printf("registered hook for component: %q (priority: %d) finished termination in time",
						f.componentName, o)
					// proceed to the next command (if any left)
				}
			}(c)
		}

		runWg.Wait()
	}
}

// serverWrapper provides a methods to start underlying srv asynchronously.
type serverWrapper struct {
	log Logger

	srv Server
}

// ServerAsyncStarterFunc creates an serverWrapper instance with graceful shutdown capabilities.
// The HTTP server will be gracefully shutdown once appCtx is done.
//
// Note: this method will start internal server shutdown monitoring goroutine.
func (s *Stopper) ServerAsyncStarterFunc(appCtx context.Context, srv Server) func() {
	srvWithShutdown := &serverWrapper{
		log: s.log,
		srv: srv,
	}

	{
		s.wg.Add(1)
		go srvWithShutdown.waitShutdown(appCtx, s.wg)
	}

	return srvWithShutdown.startAsync
}

// startAsync starts Server asynchronously.
func (w *serverWrapper) startAsync() {
	go func() {
		w.log.Printf("starting server asynchronously")
		if err := w.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			w.log.Printf("server ListenAndServe() finished: %+v", err)
		}
	}()
}

func (w *serverWrapper) waitShutdown(appCtx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	<-appCtx.Done()

	if err := w.srv.Shutdown(context.Background()); err != nil {
		w.log.Printf("server Shutdown() finished with error: %+v", err)
	}
}
