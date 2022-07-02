package graterm

// Logger specifies the interface for all log operations.
type Logger interface {
	Printf(format string, v ...interface{})
}

// noopLogger is a logger that will do nothing.
type noopLogger struct {
}

func (_ noopLogger) Printf(format string, v ...interface{}) {
	// do nothing.
}

var _ Logger = noopLogger{}
