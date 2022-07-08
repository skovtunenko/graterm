package graterm

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// TerminationOrder is an application components termination order.
//
// Lower order - higher priority.
type TerminationOrder int

// terminationHook is a registered termination hook.
type terminationHook struct {
	terminator *Terminator

	order    TerminationOrder
	name     string
	timeout  time.Duration
	hookFunc func(ctx context.Context)
}

var _ fmt.Stringer = &terminationHook{}

// String returns string representation of terminationFunc.
func (tf *terminationHook) String() string {
	if tf == nil {
		return "<nil>"
	}
	if strings.TrimSpace(tf.name) == "" {
		return fmt.Sprintf("nameless component (order: %d)", tf.order)
	}
	return fmt.Sprintf("component: %q (order: %d)", tf.name, tf.order)
}

// WithName sets (optional) human-readable name of the registered termination hook.
func (tf *terminationHook) WithName(name string) *terminationHook {
	tf.name = name
	return tf
}

// Register registers termination hook that should finish execution in less than given timeout.
// Timeout duration must be greater than zero; if not, timeout of 1 min will be used.
func (tf *terminationHook) Register(timeout time.Duration, hookFunc func(ctx context.Context)) {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	tf.timeout = timeout
	tf.hookFunc = hookFunc

	tf.terminator.hooksMx.Lock()
	defer tf.terminator.hooksMx.Unlock()
	tf.terminator.hooks[tf.order] = append(tf.terminator.hooks[tf.order], *tf)
}
