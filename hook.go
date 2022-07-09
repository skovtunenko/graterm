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
type Hook struct {
	terminator *Terminator

	order    Order
	name     string
	timeout  time.Duration
	hookFunc func(ctx context.Context)
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

// String returns string representation of terminationFunc.
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
