package processor

/*
Logger is the minimal logging interface used by Processor.
*/
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// NopLogger is a no-operation logger that silently discards all logged messages.
type nopLogger struct{}

// Error logs an error message with optional arguments, but the implementation discards the message and does nothing.
func (nopLogger) Error(string, ...any) {}

// Info logs an informational message with optional arguments, but the implementation discards the message and does nothing.
func (nopLogger) Info(string, ...any) {}
