package logger

/*
Logger is the minimal logging interface used by Processor.
*/
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

// NopLogger is a no-operation logger that silently discards all logged messages.
type NopLogger struct{}

// Debug logs a debug-level message with optional arguments, but the implementation discards the message and does nothing.
func (l NopLogger) Debug(msg string, args ...any) {}

// Error logs an error message with optional arguments, but the implementation discards the message and does nothing.
func (NopLogger) Error(string, ...any) {}

// Info logs an informational message with optional arguments, but the implementation discards the message and does nothing.
func (NopLogger) Info(string, ...any) {}
