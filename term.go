package graterm

import (
	"context"
	"os"
	"os/signal"
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

// Stopper is a service stopper that executes shutdown hooks sequentially in a specified order.
type Stopper struct {
	termComponentsMx *sync.Mutex
	termComponents   map[TerminationOrder][]terminationFunc

	wg *sync.WaitGroup

	cancelFunc context.CancelFunc // todo check later on if this needed?

	log Logger
}

// NewWithSignals creates a new instance of application component stopper.
//
// Note: this method will start internal monitoring goroutine.
func NewWithSignals(appCtx context.Context, log Logger, sig ...os.Signal) (*Stopper, context.Context) {
	ctx, cancel := withSignals(appCtx, sig...)
	return &Stopper{
		termComponentsMx: &sync.Mutex{},
		termComponents:   make(map[TerminationOrder][]terminationFunc),
		wg:               &sync.WaitGroup{},
		cancelFunc:       cancel,
		log:              log,
	}, ctx
}

// withSignals return a copy of the parent context that will be canceled by signal.
// If no signals are provided, any incoming signal will cause cancel.
// Otherwise, just the provided signals will.
//
// Note: this method will start internal monitoring goroutine.
func withSignals(ctx context.Context, sig ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	chSignals := make(chan os.Signal, 1)
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
