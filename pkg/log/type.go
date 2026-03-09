package log

import "go.uber.org/zap"

// ZapConfig holds configuration for the Zap logger.
type ZapConfig struct {
	Level        string
	Mode         string
	Encoding     string
	ColorEnabled bool
}

// zapLogger implements Logger.
type zapLogger struct {
	sugarLogger *zap.SugaredLogger
	cfg         *ZapConfig
}
