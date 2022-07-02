package graterm

import (
	"context"
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
