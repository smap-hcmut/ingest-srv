package log

const (
	// ModeProduction is the production logger mode.
	ModeProduction = "production"
	// ModeDevelopment is the development logger mode.
	ModeDevelopment = "development"
	// EncodingConsole is console (human-readable) encoding.
	EncodingConsole = "console"
	// EncodingJSON is JSON encoding.
	EncodingJSON = "json"
)

// Log level names (for config mapping).
const (
	LevelDebug  = "debug"
	LevelInfo   = "info"
	LevelWarn   = "warn"
	LevelError  = "error"
	LevelFatal  = "fatal"
	LevelPanic  = "panic"
	LevelDPanic = "dpanic"
)
