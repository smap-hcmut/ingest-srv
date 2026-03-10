package log

import "context"

// Logger defines the interface for structured logging.
// Implementations are safe for concurrent use.
type Logger interface {
	Debug(ctx context.Context, arg ...any)
	Debugf(ctx context.Context, template string, arg ...any)
	Info(ctx context.Context, arg ...any)
	Infof(ctx context.Context, template string, arg ...any)
	Warn(ctx context.Context, arg ...any)
	Warnf(ctx context.Context, template string, arg ...any)
	Error(ctx context.Context, arg ...any)
	Errorf(ctx context.Context, template string, arg ...any)
	DPanic(ctx context.Context, arg ...any)
	DPanicf(ctx context.Context, template string, arg ...any)
	Panic(ctx context.Context, arg ...any)
	Panicf(ctx context.Context, template string, arg ...any)
	Fatal(ctx context.Context, arg ...any)
	Fatalf(ctx context.Context, template string, arg ...any)
}

// Init initializes and returns a new Logger with the provided Zap configuration.
func Init(cfg ZapConfig) Logger {
	logger := &zapLogger{cfg: &cfg}
	logger.init()
	return logger
}
