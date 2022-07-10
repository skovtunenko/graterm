package graterm

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	// defaultTimeout is a default timeout for a registered hook.
	defaultTimeout = time.Minute
)

// Order is an application components termination order.
// Termination hooks registered with the same order will be executed concurrently.
//
// Lower order - higher priority.
type Order int

// Hook is a registered termination hook.
//
// Do not create a Hook instance manually, use Terminator.WithOrder() method instead to get a Hook instance.
type Hook struct {
	terminator *Terminator // terminator is a pointer to Terminator instance that holds registered Hooks.

	order    Order                     // order is Hook order.
	name     string                    // name is an optional component name for pretty-printing in logs.
	timeout  time.Duration             // timeout is max hookFunc execution timeout.
	hookFunc func(ctx context.Context) // hookFunc is a user-defined termination hook function.
}

// WithName sets (optional) human-readable name of the registered termination hook.
func (h *Hook) WithName(name string) *Hook {
	h.name = name
	return h
}

// Register registers termination hook that should finish execution in less than given timeout.
// Timeout duration must be greater than zero; if not, timeout of 1 min will be used.
func (h *Hook) Register(timeout time.Duration, hookFunc func(ctx context.Context)) {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	h.timeout = timeout
	h.hookFunc = hookFunc

	h.terminator.hooksMx.Lock()
	defer h.terminator.hooksMx.Unlock()
	h.terminator.hooks[h.order] = append(h.terminator.hooks[h.order], *h)
}

// String returns string representation of a Hook.
func (h *Hook) String() string {
	if h == nil {
		return "<nil>"
	}
	if strings.TrimSpace(h.name) == "" {
		return fmt.Sprintf("nameless component (order: %d)", h.order)
	}
	return fmt.Sprintf("component: %q (order: %d)", h.name, h.order)
}

var _ fmt.Stringer = &Hook{}
