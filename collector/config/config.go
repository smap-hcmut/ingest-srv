package config

import (
	"fmt"

	"github.com/caarlos0/env/v9"
)

type Config struct {
	// Logger Configuration
	Logger LoggerConfig

	// Message Queue Configuration
	RabbitMQConfig RabbitMQConfig

	// Redis Configuration for State Management
	Redis RedisConfig

	// External Services
	Project ProjectConfig

	// Monitoring & Notification Configuration
	Discord DiscordConfig

	// Crawl Limits Configuration
	CrawlLimits CrawlLimitsConfig
}

// CrawlLimitsConfig chứa các giới hạn cho crawling.
// Tất cả giá trị được load từ env vars, không hardcode trong code.
type CrawlLimitsConfig struct {
	// Default limits (used for production crawling)
	DefaultLimitPerKeyword int `env:"DEFAULT_LIMIT_PER_KEYWORD" envDefault:"50"`
	DefaultMaxComments     int `env:"DEFAULT_MAX_COMMENTS" envDefault:"100"`
	DefaultMaxAttempts     int `env:"DEFAULT_MAX_ATTEMPTS" envDefault:"3"`

	// Dry-run limits (for testing/preview)
	DryRunLimitPerKeyword int `env:"DRYRUN_LIMIT_PER_KEYWORD" envDefault:"3"`
	DryRunMaxComments     int `env:"DRYRUN_MAX_COMMENTS" envDefault:"5"`

	// Hard limits (safety caps - cannot be exceeded)
	MaxLimitPerKeyword int `env:"MAX_LIMIT_PER_KEYWORD" envDefault:"500"`
	MaxMaxComments     int `env:"MAX_MAX_COMMENTS" envDefault:"1000"`

	// Feature flags
	IncludeComments bool `env:"INCLUDE_COMMENTS" envDefault:"true"`
	DownloadMedia   bool `env:"DOWNLOAD_MEDIA" envDefault:"false"`
}

// LoggerConfig is the configuration for the logger.
type LoggerConfig struct {
	Level    string `env:"LOGGER_LEVEL" envDefault:"debug"`
	Mode     string `env:"LOGGER_MODE" envDefault:"debug"`
	Encoding string `env:"LOGGER_ENCODING" envDefault:"console"`
}

// DiscordConfig is the configuration for Discord webhooks.
type DiscordConfig struct {
	ReportBugID    string `env:"DISCORD_REPORT_BUG_ID"`
	ReportBugToken string `env:"DISCORD_REPORT_BUG_TOKEN"`
}

// RabbitMQConfig is the configuration for RabbitMQ,
// which is used to connect to RabbitMQ server.
type RabbitMQConfig struct {
	URL string `env:"RABBITMQ_URL"`
}

// RedisConfig is the configuration for Redis state management.
// Used for tracking project execution state (DB 1).
// Note: Only standalone mode is supported
type RedisConfig struct {
	Host         string `env:"REDIS_HOST" envDefault:"localhost:6379"`
	Password     string `env:"REDIS_PASSWORD"`
	DB           int    `env:"REDIS_DB" envDefault:"0"`
	StateDB      int    `env:"REDIS_STATE_DB" envDefault:"1"`
	MinIdleConns int    `env:"REDIS_MIN_IDLE_CONNS" envDefault:"10"`
	PoolSize     int    `env:"REDIS_POOL_SIZE" envDefault:"100"`
	PoolTimeout  int    `env:"REDIS_POOL_TIMEOUT" envDefault:"30"`
}

// ProjectConfig is the configuration for the Project Service.
type ProjectConfig struct {
	BaseURL              string `env:"PROJECT_SERVICE_URL" envDefault:"http://localhost:8080"`
	Timeout              int    `env:"PROJECT_TIMEOUT" envDefault:"10"`
	InternalKey          string `env:"PROJECT_INTERNAL_KEY"`
	WebhookRetryAttempts int    `env:"WEBHOOK_RETRY_ATTEMPTS" envDefault:"5"`
	WebhookRetryDelay    int    `env:"WEBHOOK_RETRY_DELAY" envDefault:"1"`
}

// Load is the function to load the configuration from the environment variables.
func Load() (*Config, error) {
	cfg := &Config{}
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}
	// Print all config for testing
	fmt.Printf("%+v\n", cfg)
	return cfg, nil
}
