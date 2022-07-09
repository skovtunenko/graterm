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
func (tf *Hook) WithName(name string) *Hook {
	tf.name = name
	return tf
}

// Register registers termination hook that should finish execution in less than given timeout.
// Timeout duration must be greater than zero; if not, timeout of 1 min will be used.
func (tf *Hook) Register(timeout time.Duration, hookFunc func(ctx context.Context)) {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	tf.timeout = timeout
	tf.hookFunc = hookFunc

	tf.terminator.hooksMx.Lock()
	defer tf.terminator.hooksMx.Unlock()
	tf.terminator.hooks[tf.order] = append(tf.terminator.hooks[tf.order], *tf)
}

// String returns string representation of a Hook.
func (tf *Hook) String() string {
	if tf == nil {
		return "<nil>"
	}
	if strings.TrimSpace(tf.name) == "" {
		return fmt.Sprintf("nameless component (order: %d)", tf.order)
	}
	return fmt.Sprintf("component: %q (order: %d)", tf.name, tf.order)
}

var _ fmt.Stringer = &Hook{}
