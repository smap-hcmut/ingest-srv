package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// IRabbitMQ is the RabbitMQ interface. Implementations are safe for concurrent use.
type IRabbitMQ interface {
	Close()
	IsReady() bool
	IsClosed() bool
	Channel() (IChannel, error)
}

// IChannel is the RabbitMQ channel interface. Implementations are safe for concurrent use.
type IChannel interface {
	ExchangeDeclare(exc ExchangeArgs) error
	QueueDeclare(queue QueueArgs) (amqp.Queue, error)
	QueueBind(queueBind QueueBindArgs) error
	Publish(ctx context.Context, publish PublishArgs) error
	Consume(consume ConsumeArgs) (<-chan amqp.Delivery, error)
	Close() error
	NotifyReconnect(receiver chan bool) <-chan bool
}

// NewRabbitMQ creates a new RabbitMQ connection. Returns IRabbitMQ.
func NewRabbitMQ(url string, retryWithoutTimeout bool) (IRabbitMQ, error) {
	conn := &connectionImpl{
		url:                 url,
		retryWithoutTimeout: retryWithoutTimeout,
	}
	if err := conn.connect(); err != nil {
		return nil, err
	}
	return conn, nil
}
