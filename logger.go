package graterm

// Logger specifies the interface for internal Terminator log operations.
//
// By default, library will not log anything.
// To set the logger, use Terminator.SetLogger() method.
type Logger interface {
	Printf(format string, v ...interface{})
}

// noopLogger is a logger that will do nothing.
type noopLogger struct{}

// Printf will do nothing.
func (noopLogger) Printf(format string, v ...interface{}) {
	// do nothing.
}

var _ Logger = noopLogger{}
