package producer

import (
	"context"

	"smap-collector/internal/models"
	pkgLog "smap-collector/pkg/log"
	pkgRabbit "smap-collector/pkg/rabbitmq"
)

// Producer is a interface that represents a producer.
//
//go:generate mockery --name=Producer
type Producer interface {
	PublishTikTokTask(ctx context.Context, task models.TikTokCollectorTask) error
	PublishYouTubeTask(ctx context.Context, task models.YouTubeCollectorTask) error
	// Run chuẩn bị writer/publisher.
	Run() error
	// Close đóng tài nguyên MQ.
	Close()
}

type implProducer struct {
	l      pkgLog.Logger
	conn   *pkgRabbit.Connection
	writer *pkgRabbit.Channel
}

// New creates a new producer.
func New(l pkgLog.Logger, conn *pkgRabbit.Connection) Producer {
	return &implProducer{
		l:    l,
		conn: conn,
	}
}
