package consumer

import (
	"database/sql"

	"ingest-srv/pkg/log"
	"ingest-srv/pkg/minio"
	"ingest-srv/pkg/rabbitmq"
)

// Server is the consumer server.
type Server struct {
	l     log.Logger
	conn  rabbitmq.IRabbitMQ
	db    *sql.DB
	minio minio.MinIO
}

type ServerConfig struct {
	Conn  rabbitmq.IRabbitMQ
	DB    *sql.DB
	MinIO minio.MinIO
}

// NewServer creates a new consumer server.
func NewServer(l log.Logger, config ServerConfig) Server {
	return Server{
		l:     l,
		conn:  config.Conn,
		db:    config.DB,
		minio: config.MinIO,
	}
}
