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

// Terminator is a component terminator that executes registered termination Hooks in a specified order.
type Terminator struct {
	hooksMx *sync.Mutex
	hooks   map[Order][]Hook

	wg *sync.WaitGroup

	cancelFunc context.CancelFunc

	log Logger
}

// NewWithSignals creates a new instance of component Terminator.
//
// If the given appCtx parameter is canceled, the termination process will start for already registered Hook instances
// after calling Terminator.Wait method.
//
// Example of useful signals might be: [syscall.SIGINT], [syscall.SIGTERM].
//
// Note: this method will start internal monitoring goroutine.
func NewWithSignals(appCtx context.Context, sig ...os.Signal) (*Terminator, context.Context) {
	ctx, cancel := signal.NotifyContext(appCtx, sig...)
	return &Terminator{
		hooksMx:    &sync.Mutex{},
		hooks:      make(map[Order][]Hook),
		wg:         &sync.WaitGroup{},
		cancelFunc: cancel,
		log:        noopLogger{},
	}, ctx
}

// SetLogger sets the Logger implementation.
//
// If log is nil, then NOOP logger implementation will be used.
func (t *Terminator) SetLogger(log Logger) {
	if log == nil {
		log = noopLogger{}
	}

	t.hooksMx.Lock()
	defer t.hooksMx.Unlock()
	t.log = log
}

// WithOrder sets the Order for the termination hook.
// It starts registration chain to register termination hook with priority.
//
// The lower the Order the higher the execution priority, the earlier it will be executed.
// If there are multiple hooks with the same Order they will be executed in parallel.
func (t *Terminator) WithOrder(order Order) *Hook {
	return &Hook{
		terminator: t,
		order:      order,
	}
}

// Wait blocks execution until the provided appCtx is canceled and then executes all registered termination hooks.
//
// This function initiates the shutdown sequence when appCtx is done, typically due to receiving an [os.Signal] events.
// After appCtx is canceled, it waits for all registered hooks to complete execution within the specified shutdownTimeout.
//
// Hooks are executed in order of priority (lower order values execute first). Hooks with the same order run concurrently.
// If the shutdownTimeout expires before all hooks complete, the function returns an error.
//
// This is a blocking call that should be placed at the end of the application's lifecycle to ensure a proper shutdown.
//
// Parameters:
//   - appCtx: The application context that, when canceled, triggers the termination process.
//   - shutdownTimeout: The maximum time allowed for all hooks to complete execution.
//
// Returns:
//   - error: If termination exceeds the shutdownTimeout, an error is returned indicating a timeout.
func (t *Terminator) Wait(appCtx context.Context, shutdownTimeout time.Duration) error {
	{
		t.wg.Add(1)
		go t.waitShutdown(appCtx)
	}

	<-appCtx.Done()

	wgChan := make(chan struct{})
	go func() {
		defer close(wgChan)
		t.wg.Wait()
	}()

	select {
	case <-time.After(shutdownTimeout):
		return fmt.Errorf("termination timed out after %v", shutdownTimeout)
	case <-wgChan:
		return nil
	}
}

// waitShutdown waits for the context to be canceled and then executes the registered shutdown hooks sequentially.
func (t *Terminator) waitShutdown(appCtx context.Context) {
	defer t.wg.Done()

	<-appCtx.Done() // Block until application context is done (most likely, when the registered os.Signal will be received)

	t.hooksMx.Lock()
	defer t.hooksMx.Unlock()

	for _, order := range t.getSortedOrders() {
		t.processOrder(order)
	}
}

// getSortedOrders returns a slice of hook orders sorted in ascending order.
func (t *Terminator) getSortedOrders() []Order {
	orders := make([]Order, 0, len(t.hooks))
	for order := range t.hooks {
		orders = append(orders, order)
	}
	sort.Slice(orders, func(i, j int) bool {
		return orders[i] < orders[j]
	})
	return orders
}

// processOrder executes all hooks associated with the given order concurrently and waits for all to finish.
func (t *Terminator) processOrder(order Order) {
	var wg sync.WaitGroup
	for _, hook := range t.hooks[order] {
		wg.Add(1)

		go func(h Hook) {
			defer wg.Done()

			t.executeHook(h)
		}(hook)
	}
	wg.Wait()
}

// executeHook runs a single hook with a timeout, recovers from panics, and logs the outcome.
func (t *Terminator) executeHook(hook Hook) {
	ctx, cancel := context.WithTimeout(context.Background(), hook.timeout)
	defer cancel()

	var duration time.Duration

	start := time.Now()

	go func() {
		defer func() {
			defer cancel()

			duration = time.Since(start)

			if r := recover(); r != nil {
				t.log.Printf("registered hook panicked after %v for %v, recovered: %+v", duration, &hook, r)
			}
		}()

		hook.hookFunc(ctx)
	}()

	<-ctx.Done() // block until the hookFunc is over OR timeout has been expired

	switch err := ctx.Err(); {
	case errors.Is(err, context.DeadlineExceeded):
		t.log.Printf("registered hook timed out after %v for %v", hook.timeout, &hook)
	case errors.Is(err, context.Canceled):
		t.log.Printf("registered hook finished termination in %v (out of maximum %v) for %v", duration, hook.timeout, &hook)
	}
}
