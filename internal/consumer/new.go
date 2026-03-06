package consumer

import (
	"database/sql"
	"fmt"

	"ingest-srv/config"
	"ingest-srv/pkg/kafka"
	"ingest-srv/pkg/log"
	"ingest-srv/pkg/minio"
	"ingest-srv/pkg/rabbitmq"
	"ingest-srv/pkg/redis"
)

// ConsumerServer is the ingest consumer orchestration server.
type ConsumerServer struct {
	l log.Logger

	kafkaConfig   config.KafkaConfig
	kafkaConsumer kafka.IConsumer

	postgresDB *sql.DB
	redis      redis.IRedis
	minio      minio.MinIO
	rabbitmq   rabbitmq.IRabbitMQ
}

// Config holds dependencies for ConsumerServer.
type Config struct {
	Logger log.Logger

	KafkaConfig   config.KafkaConfig
	KafkaConsumer kafka.IConsumer

	PostgresDB *sql.DB
	Redis      redis.IRedis
	MinIO      minio.MinIO
	RabbitMQ   rabbitmq.IRabbitMQ
}

// New creates and validates consumer server.
func New(cfg Config) (*ConsumerServer, error) {
	srv := &ConsumerServer{
		l: cfg.Logger,

		kafkaConfig:   cfg.KafkaConfig,
		kafkaConsumer: cfg.KafkaConsumer,

		postgresDB: cfg.PostgresDB,
		redis:      cfg.Redis,
		minio:      cfg.MinIO,
		rabbitmq:   cfg.RabbitMQ,
	}

	if err := srv.validate(); err != nil {
		return nil, err
	}

	return srv, nil
}

func (srv *ConsumerServer) validate() error {
	if srv.l == nil {
		return fmt.Errorf("logger is required")
	}
	if len(srv.kafkaConfig.Brokers) == 0 {
		return fmt.Errorf("kafka brokers are required")
	}
	if srv.kafkaConfig.GroupID == "" {
		return fmt.Errorf("kafka group_id is required")
	}
	if srv.kafkaConfig.Topic == "" {
		return fmt.Errorf("kafka topic is required")
	}
	return nil
}
