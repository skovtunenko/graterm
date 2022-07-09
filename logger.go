package graterm

// Logger specifies the interface for internal Terminator log operations.
type Logger interface {
	Printf(format string, v ...interface{})
}

// noopLogger is a logger that will do nothing.
type noopLogger struct {
}

// Printf will do nothing.
func (_ noopLogger) Printf(format string, v ...interface{}) {
	// do nothing.
}

var _ Logger = noopLogger{}
