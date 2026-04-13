package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all service configuration.
type Config struct {
	Environment EnvironmentConfig
	HTTPServer  HTTPServerConfig
	Logger      LoggerConfig

	Postgres     PostgresConfig
	Redis        RedisConfig
	MinIO        MinIOConfig
	Kafka        KafkaConfig
	RabbitMQ     RabbitMQConfig
	Microservice MicroserviceConfig

	JWT            JWTConfig
	Cookie         CookieConfig
	Encrypter      EncrypterConfig
	InternalConfig InternalConfig
	Discord        DiscordConfig
	Scheduler      SchedulerConfig
}

type EnvironmentConfig struct {
	Name string
}

type HTTPServerConfig struct {
	Host string
	Port int
	Mode string
}

type LoggerConfig struct {
	Level        string
	Mode         string
	Encoding     string
	ColorEnabled bool
}

type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	Schema   string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type MinIOConfig struct {
	Endpoint             string
	AccessKey            string
	SecretKey            string
	UseSSL               bool
	Region               string
	Bucket               string
	AsyncUploadWorkers   int
	AsyncUploadQueueSize int
}

type KafkaConfig struct {
	Brokers  []string
	Topic    string
	UAPTopic string
	GroupID  string
}

type RabbitMQConfig struct {
	URL                 string
	RetryWithoutTimeout bool
}

type MicroserviceConfig struct {
	Project ProjectMicroserviceConfig
}

type ProjectMicroserviceConfig struct {
	BaseURL   string
	TimeoutMS int
}

type JWTConfig struct {
	SecretKey string
}

// CookieConfig is the configuration for HttpOnly cookie authentication
// Note: Secure and SameSite are now dynamically determined by auth.Middleware
// based on the request Origin header. Bearer token acceptance is controlled by ENVIRONMENT_NAME.
type CookieConfig struct {
	Name   string // Cookie name (e.g., "smap_auth_token")
	MaxAge int    // Cookie max age in seconds (e.g., 28800 for 8 hours)
	Domain string // Production domain for cookies (e.g., ".tantai.dev")
}

type EncrypterConfig struct {
	Key string
}

type InternalConfig struct {
	InternalKey string
}

type DiscordConfig struct {
	WebhookURL string
}

type SchedulerConfig struct {
	HeartbeatCron  string
	Timezone       string
	HeartbeatLimit int
}

// Load loads configuration using Viper.
func Load() (*Config, error) {
	if configFile := os.Getenv("INGEST_CONFIG_FILE"); configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("ingest-config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("./config")
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/smap/")
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	cfg := &Config{}

	cfg.Environment.Name = viper.GetString("environment.name")
	cfg.HTTPServer.Host = viper.GetString("http_server.host")
	cfg.HTTPServer.Port = viper.GetInt("http_server.port")
	cfg.HTTPServer.Mode = viper.GetString("http_server.mode")

	cfg.Logger.Level = viper.GetString("logger.level")
	cfg.Logger.Mode = viper.GetString("logger.mode")
	cfg.Logger.Encoding = viper.GetString("logger.encoding")
	cfg.Logger.ColorEnabled = viper.GetBool("logger.color_enabled")

	cfg.Postgres.Host = viper.GetString("postgres.host")
	cfg.Postgres.Port = viper.GetInt("postgres.port")
	cfg.Postgres.User = viper.GetString("postgres.user")
	cfg.Postgres.Password = viper.GetString("postgres.password")
	cfg.Postgres.DBName = viper.GetString("postgres.dbname")
	cfg.Postgres.SSLMode = viper.GetString("postgres.sslmode")
	cfg.Postgres.Schema = viper.GetString("postgres.schema")

	cfg.Redis.Host = viper.GetString("redis.host")
	cfg.Redis.Port = viper.GetInt("redis.port")
	cfg.Redis.Password = viper.GetString("redis.password")
	cfg.Redis.DB = viper.GetInt("redis.db")

	cfg.MinIO.Endpoint = viper.GetString("minio.endpoint")
	cfg.MinIO.AccessKey = viper.GetString("minio.access_key")
	cfg.MinIO.SecretKey = viper.GetString("minio.secret_key")
	cfg.MinIO.UseSSL = viper.GetBool("minio.use_ssl")
	cfg.MinIO.Region = viper.GetString("minio.region")
	cfg.MinIO.Bucket = viper.GetString("minio.bucket")
	cfg.MinIO.AsyncUploadWorkers = viper.GetInt("minio.async_upload_workers")
	cfg.MinIO.AsyncUploadQueueSize = viper.GetInt("minio.async_upload_queue_size")

	cfg.Kafka.Brokers = viper.GetStringSlice("kafka.brokers")
	cfg.Kafka.Topic = viper.GetString("kafka.topic")
	cfg.Kafka.UAPTopic = viper.GetString("kafka.uap_topic")
	cfg.Kafka.GroupID = viper.GetString("kafka.group_id")

	cfg.RabbitMQ.URL = viper.GetString("rabbitmq.url")
	cfg.RabbitMQ.RetryWithoutTimeout = viper.GetBool("rabbitmq.retry_without_timeout")

	cfg.Microservice.Project.BaseURL = viper.GetString("microservice.project.base_url")
	cfg.Microservice.Project.TimeoutMS = viper.GetInt("microservice.project.timeout_ms")

	cfg.JWT.SecretKey = viper.GetString("jwt.secret_key")

	cfg.Cookie.Name = viper.GetString("cookie.name")
	cfg.Cookie.MaxAge = viper.GetInt("cookie.max_age")
	cfg.Cookie.Domain = viper.GetString("cookie.domain")

	cfg.Encrypter.Key = viper.GetString("encrypter.key")
	cfg.InternalConfig.InternalKey = viper.GetString("internal.internal_key")

	cfg.Discord.WebhookURL = viper.GetString("discord.webhook_url")

	cfg.Scheduler.HeartbeatCron = viper.GetString("scheduler.heartbeat_cron")
	cfg.Scheduler.Timezone = viper.GetString("scheduler.timezone")
	cfg.Scheduler.HeartbeatLimit = viper.GetInt("scheduler.heartbeat_limit")

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func setDefaults() {
	viper.SetDefault("environment.name", "development")

	viper.SetDefault("http_server.host", "0.0.0.0")
	viper.SetDefault("http_server.port", 8080)
	viper.SetDefault("http_server.mode", "debug")

	viper.SetDefault("logger.level", "debug")
	viper.SetDefault("logger.mode", "debug")
	viper.SetDefault("logger.encoding", "console")
	viper.SetDefault("logger.color_enabled", true)

	viper.SetDefault("postgres.host", "172.16.19.10")
	viper.SetDefault("postgres.port", 5432)
	viper.SetDefault("postgres.user", "ingest_master")
	viper.SetDefault("postgres.password", "ingest_master_pwd")
	viper.SetDefault("postgres.dbname", "smap")
	viper.SetDefault("postgres.sslmode", "disable")
	viper.SetDefault("postgres.schema", "ingest")

	viper.SetDefault("redis.host", "redis.tantai.dev")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "21042004")
	viper.SetDefault("redis.db", 1)

	viper.SetDefault("minio.endpoint", "172.16.21.10:9000")
	viper.SetDefault("minio.access_key", "tantai")
	viper.SetDefault("minio.secret_key", "21042004")
	viper.SetDefault("minio.use_ssl", false)
	viper.SetDefault("minio.region", "us-east-1")
	viper.SetDefault("minio.bucket", "ingest-data")
	viper.SetDefault("minio.async_upload_workers", 4)
	viper.SetDefault("minio.async_upload_queue_size", 100)

	viper.SetDefault("kafka.brokers", []string{"kafka.tantai.dev:9094"})
	viper.SetDefault("kafka.topic", "ingest.events")
	viper.SetDefault("kafka.uap_topic", "smap.collector.output")
	viper.SetDefault("kafka.group_id", "ingest-consumer")

	viper.SetDefault("rabbitmq.url", "amqp://admin:21042004@172.16.21.206:5672/")
	viper.SetDefault("rabbitmq.retry_without_timeout", true)

	viper.SetDefault("microservice.project.base_url", "http://localhost:8082")
	viper.SetDefault("microservice.project.timeout_ms", 5000)

	viper.SetDefault("cookie.name", "smap_auth_token")
	viper.SetDefault("cookie.max_age", 28800) // 8 hours
	viper.SetDefault("cookie.domain", ".tantai.dev")

	viper.SetDefault("scheduler.heartbeat_cron", "*/1 * * * *")
	viper.SetDefault("scheduler.timezone", "Asia/Ho_Chi_Minh")
	viper.SetDefault("scheduler.heartbeat_limit", 20)
}

func validate(cfg *Config) error {
	if cfg.JWT.SecretKey == "" {
		return fmt.Errorf("jwt.secret_key is required")
	}
	if len(cfg.JWT.SecretKey) < 32 {
		return fmt.Errorf("jwt.secret_key must be at least 32 characters for security")
	}

	if cfg.Encrypter.Key == "" {
		return fmt.Errorf("encrypter.key is required")
	}
	if len(cfg.Encrypter.Key) < 32 {
		return fmt.Errorf("encrypter.key must be at least 32 characters for security")
	}
	if cfg.InternalConfig.InternalKey == "" {
		return fmt.Errorf("internal.internal_key is required")
	}

	if cfg.Postgres.Host == "" {
		return fmt.Errorf("postgres.host is required")
	}
	if cfg.Postgres.Port == 0 {
		return fmt.Errorf("postgres.port is required")
	}
	if cfg.Postgres.DBName == "" {
		return fmt.Errorf("postgres.dbname is required")
	}
	if cfg.Postgres.User == "" {
		return fmt.Errorf("postgres.user is required")
	}

	if cfg.Redis.Host == "" {
		return fmt.Errorf("redis.host is required")
	}
	if cfg.Redis.Port == 0 {
		return fmt.Errorf("redis.port is required")
	}

	if cfg.MinIO.Endpoint == "" {
		return fmt.Errorf("minio.endpoint is required")
	}
	if cfg.MinIO.AccessKey == "" {
		return fmt.Errorf("minio.access_key is required")
	}
	if cfg.MinIO.SecretKey == "" {
		return fmt.Errorf("minio.secret_key is required")
	}
	if cfg.MinIO.Bucket == "" {
		return fmt.Errorf("minio.bucket is required")
	}

	if len(cfg.Kafka.Brokers) == 0 {
		return fmt.Errorf("kafka.brokers is required")
	}
	if cfg.Kafka.Topic == "" {
		return fmt.Errorf("kafka.topic is required")
	}
	if cfg.Kafka.UAPTopic == "" {
		return fmt.Errorf("kafka.uap_topic is required")
	}
	if cfg.Kafka.GroupID == "" {
		return fmt.Errorf("kafka.group_id is required")
	}

	if cfg.RabbitMQ.URL == "" {
		return fmt.Errorf("rabbitmq.url is required")
	}
	if cfg.Microservice.Project.BaseURL == "" {
		return fmt.Errorf("microservice.project.base_url is required")
	}
	if cfg.Microservice.Project.TimeoutMS <= 0 {
		return fmt.Errorf("microservice.project.timeout_ms must be greater than 0")
	}

	if cfg.Cookie.Name == "" {
		return fmt.Errorf("cookie.name is required")
	}
	if cfg.Scheduler.HeartbeatCron == "" {
		return fmt.Errorf("scheduler.heartbeat_cron is required")
	}
	if cfg.Scheduler.HeartbeatLimit <= 0 {
		return fmt.Errorf("scheduler.heartbeat_limit must be greater than 0")
	}

	return nil
}
