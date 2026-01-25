package log

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ZapConfig struct {
	Level        string
	Mode         string
	Encoding     string
	ColorEnabled bool
}

type zapLogger struct {
	sugarLogger *zap.SugaredLogger
	cfg         *ZapConfig
}

// Init initializes and returns a new Zap logger instance with the provided configuration.
func Init(cfg ZapConfig) Logger {
	logger := zapLogger{
		cfg: &cfg,
	}
	logger.init()
	return &logger
}

// For mapping config logger to app logger levels
var logLevelMap = map[string]zapcore.Level{
	"debug":  zapcore.DebugLevel,
	"info":   zapcore.InfoLevel,
	"warn":   zapcore.WarnLevel,
	"error":  zapcore.ErrorLevel,
	"fatal":  zapcore.FatalLevel,
	"panic":  zapcore.PanicLevel,
	"dpanic": zapcore.DPanicLevel,
}

func (l *zapLogger) getLoggerLevel() zapcore.Level {
	level, exist := logLevelMap[l.cfg.Level]
	if !exist {
		return zapcore.DebugLevel
	}
	return level
}

func (l *zapLogger) init() {
	logLevel := l.getLoggerLevel()

	logWriter := zapcore.AddSync(os.Stderr)

	var encoderCfg zapcore.EncoderConfig
	if l.cfg.Mode == "production" {
		encoderCfg = zap.NewProductionEncoderConfig()
	} else {
		encoderCfg = zap.NewDevelopmentEncoderConfig()
	}

	encoderCfg.LevelKey = "LEVEL"
	encoderCfg.CallerKey = "CALLER"
	encoderCfg.TimeKey = "TIME"
	encoderCfg.NameKey = "NAME"
	encoderCfg.MessageKey = "MESSAGE"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	// Enable colored output when ColorEnabled is true and using console encoding
	if l.cfg.ColorEnabled && l.cfg.Encoding == "console" {
		encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	var encoder zapcore.Encoder
	if l.cfg.Encoding == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	}

	core := zapcore.NewCore(encoder, logWriter, zap.NewAtomicLevelAt(logLevel))
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	l.sugarLogger = logger.Sugar()
	// Note: Sync() can cause issues in some environments like terminals
	// Uncomment the following lines if you need to sync logs
	// if err := l.sugarLogger.Sync(); err != nil {
	// 	l.sugarLogger.Error(err)
	// }
}

// loggerKey holds the context key used for loggers.
type loggerKey struct{}

func (l *zapLogger) ctx(ctx context.Context) *zap.SugaredLogger {
	if ctx == nil {
		panic("nil context passed to Logger")
	}
	if logger, _ := ctx.Value(loggerKey{}).(*zap.SugaredLogger); logger != nil {
		return logger
	}

	return l.sugarLogger
}

func (l *zapLogger) Debug(ctx context.Context, args ...any) {
	l.ctx(ctx).Debug(args...)
}

func (l *zapLogger) Debugf(ctx context.Context, template string, args ...any) {
	l.ctx(ctx).Debugf(template, args...)
}

func (l *zapLogger) Info(ctx context.Context, args ...any) {
	l.ctx(ctx).Info(args...)
}

func (l *zapLogger) Infof(ctx context.Context, template string, args ...any) {
	l.ctx(ctx).Infof(template, args...)
}

func (l *zapLogger) Warn(ctx context.Context, args ...any) {
	l.ctx(ctx).Warn(args...)
}

func (l *zapLogger) Warnf(ctx context.Context, template string, args ...any) {
	l.ctx(ctx).Warnf(template, args...)
}

func (l *zapLogger) Error(ctx context.Context, args ...any) {
	l.ctx(ctx).Error(args...)
}

func (l *zapLogger) Errorf(ctx context.Context, template string, args ...any) {
	l.ctx(ctx).Errorf(template, args...)
}

func (l *zapLogger) DPanic(ctx context.Context, args ...any) {
	l.ctx(ctx).DPanic(args...)
}

func (l *zapLogger) DPanicf(ctx context.Context, template string, args ...any) {
	l.ctx(ctx).DPanicf(template, args...)
}

func (l *zapLogger) Panic(ctx context.Context, args ...any) {
	l.ctx(ctx).Panic(args...)
}

func (l *zapLogger) Panicf(ctx context.Context, template string, args ...any) {
	l.ctx(ctx).Panicf(template, args...)
}

func (l *zapLogger) Fatal(ctx context.Context, args ...any) {
	l.ctx(ctx).Fatal(args...)
}

func (l *zapLogger) Fatalf(ctx context.Context, template string, args ...any) {
	l.ctx(ctx).Fatalf(template, args...)
}
