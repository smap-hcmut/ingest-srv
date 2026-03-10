package log

import (
	"context"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var vietnamZone = time.FixedZone("ICT", 7*3600)

func rfc2822TimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.In(vietnamZone).Format(time.RFC1123Z))
}

var logLevelMap = map[string]zapcore.Level{
	LevelDebug:  zapcore.DebugLevel,
	LevelInfo:   zapcore.InfoLevel,
	LevelWarn:   zapcore.WarnLevel,
	LevelError:  zapcore.ErrorLevel,
	LevelFatal:  zapcore.FatalLevel,
	LevelPanic:  zapcore.PanicLevel,
	LevelDPanic: zapcore.DPanicLevel,
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

	encoderCfg := zap.NewDevelopmentEncoderConfig()
	if l.cfg.Mode == ModeProduction {
		encoderCfg = zap.NewProductionEncoderConfig()
	}
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = rfc2822TimeEncoder

	// In JSON mode, we leave other keys empty so orderedCore can add them in specific order
	if l.cfg.Encoding == EncodingJSON {
		encoderCfg.LevelKey = ""
		encoderCfg.CallerKey = ""
		encoderCfg.NameKey = ""
		encoderCfg.MessageKey = ""
	} else {
		encoderCfg.LevelKey = "level"
		encoderCfg.CallerKey = "caller"
		encoderCfg.NameKey = "name"
		encoderCfg.MessageKey = "message"
		encoderCfg.EncodeLevel = zapcore.LowercaseLevelEncoder
	}

	if l.cfg.ColorEnabled && l.cfg.Encoding == EncodingConsole {
		encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	var encoder zapcore.Encoder
	if l.cfg.Encoding == EncodingConsole {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	}
	core := zapcore.NewCore(encoder, logWriter, zap.NewAtomicLevelAt(logLevel))

	serviceName := os.Getenv("CONTAINER_NAME")
	if serviceName == "" {
		serviceName = "ingest-srv"
	}

	// Use our orderedCore to enforce field order in JSON mode
	if l.cfg.Encoding == EncodingJSON {
		core = &orderedCore{
			Core:        core,
			serviceName: serviceName,
		}
	}

	var logger *zap.Logger
	if l.cfg.Encoding == EncodingJSON {
		logger = zap.New(core)
	} else {
		logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1)).With(
			zap.String("service", serviceName),
			zap.String("trace_id", ""),
		)
	}

	l.sugarLogger = logger.Sugar()
}

type orderedCore struct {
	zapcore.Core
	serviceName string
	fields      []zapcore.Field
}

func (c *orderedCore) With(fields []zapcore.Field) zapcore.Core {
	// Clone and append fields
	newFields := make([]zapcore.Field, len(c.fields)+len(fields))
	copy(newFields, c.fields)
	copy(newFields[len(c.fields):], fields)

	return &orderedCore{
		Core:        c.Core,
		serviceName: c.serviceName,
		fields:      newFields,
	}
}

func (c *orderedCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *orderedCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	// Reorder fields: trace_id, level, caller, message, service
	// timestamp is already handled by the encoder if TimeKey is set

	allFields := make([]zapcore.Field, 0, len(c.fields)+len(fields))
	allFields = append(allFields, c.fields...)
	allFields = append(allFields, fields...)

	var traceID string
	// Find trace_id in all fields (last one wins)
	for _, f := range allFields {
		if f.Key == "trace_id" {
			traceID = f.String
		}
	}

	orderedFields := make([]zapcore.Field, 0, len(allFields)+5)
	orderedFields = append(orderedFields, zap.String("trace_id", traceID))
	orderedFields = append(orderedFields, zap.String("level", ent.Level.String()))
	if ent.Caller.Defined {
		orderedFields = append(orderedFields, zap.String("caller", ent.Caller.TrimmedPath()))
	} else {
		orderedFields = append(orderedFields, zap.String("caller", ""))
	}
	orderedFields = append(orderedFields, zap.String("message", ent.Message))
	orderedFields = append(orderedFields, zap.String("service", c.serviceName))

	// Add other fields, skipping trace_id as we already added it
	for _, f := range allFields {
		if f.Key != "trace_id" {
			orderedFields = append(orderedFields, f)
		}
	}

	return c.Core.Write(ent, orderedFields)
}

type loggerKey struct{}

func (l *zapLogger) ctx(ctx context.Context) *zap.SugaredLogger {
	if ctx == nil {
		panic("nil context passed to Logger")
	}

	logger := l.sugarLogger
	if ctxLogger, _ := ctx.Value(loggerKey{}).(*zap.SugaredLogger); ctxLogger != nil {
		logger = ctxLogger
	}

	if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
		return logger.With("trace_id", traceID)
	}

	return logger
}

func (l *zapLogger) Debug(ctx context.Context, args ...any)   { l.ctx(ctx).Debug(args...) }
func (l *zapLogger) Debugf(ctx context.Context, template string, args ...any) {
	l.ctx(ctx).Debugf(template, args...)
}
func (l *zapLogger) Info(ctx context.Context, args ...any)   { l.ctx(ctx).Info(args...) }
func (l *zapLogger) Infof(ctx context.Context, template string, args ...any) {
	l.ctx(ctx).Infof(template, args...)
}
func (l *zapLogger) Warn(ctx context.Context, args ...any)   { l.ctx(ctx).Warn(args...) }
func (l *zapLogger) Warnf(ctx context.Context, template string, args ...any) {
	l.ctx(ctx).Warnf(template, args...)
}
func (l *zapLogger) Error(ctx context.Context, args ...any)   { l.ctx(ctx).Error(args...) }
func (l *zapLogger) Errorf(ctx context.Context, template string, args ...any) {
	l.ctx(ctx).Errorf(template, args...)
}
func (l *zapLogger) DPanic(ctx context.Context, args ...any) { l.ctx(ctx).DPanic(args...) }
func (l *zapLogger) DPanicf(ctx context.Context, template string, args ...any) {
	l.ctx(ctx).DPanicf(template, args...)
}
func (l *zapLogger) Panic(ctx context.Context, args ...any)   { l.ctx(ctx).Panic(args...) }
func (l *zapLogger) Panicf(ctx context.Context, template string, args ...any) {
	l.ctx(ctx).Panicf(template, args...)
}
func (l *zapLogger) Fatal(ctx context.Context, args ...any) { l.ctx(ctx).Fatal(args...) }
func (l *zapLogger) Fatalf(ctx context.Context, template string, args ...any) {
	l.ctx(ctx).Fatalf(template, args...)
}
