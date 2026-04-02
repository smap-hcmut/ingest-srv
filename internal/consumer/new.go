package consumer

import (
	"database/sql"
	"ingest-srv/config"

	"github.com/smap-hcmut/shared-libs/go/kafka"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/minio"
	"github.com/smap-hcmut/shared-libs/go/rabbitmq"
)

// Server is the consumer server.
type Server struct {
	l            log.Logger
	conn         rabbitmq.IRabbitMQ
	db           *sql.DB
	minio        minio.MinIO
	uapBucket    string
	kafka        kafka.IProducer
	microservice config.MicroserviceConfig
	internalKey  string
}

type ServerConfig struct {
	Conn         rabbitmq.IRabbitMQ
	DB           *sql.DB
	MinIO        minio.MinIO
	UAPBucket    string
	Kafka        kafka.IProducer
	Microservice config.MicroserviceConfig
	InternalKey  string
}

// NewServer creates a new consumer server.
func NewServer(l log.Logger, config ServerConfig) Server {
	return Server{
		l:            l,
		conn:         config.Conn,
		db:           config.DB,
		minio:        config.MinIO,
		uapBucket:    config.UAPBucket,
		kafka:        config.Kafka,
		microservice: config.Microservice,
		internalKey:  config.InternalKey,
	}
}
