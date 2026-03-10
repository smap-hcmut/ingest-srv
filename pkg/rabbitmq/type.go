package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// connectionImpl implements IConnection.
type connectionImpl struct {
	url                 string
	retryWithoutTimeout bool
	conn                *amqp.Connection
	isRetrying          bool
	reconnects          []chan bool
}

// channelImpl implements IChannel.
type channelImpl struct {
	conn       *connectionImpl
	ch         *amqp.Channel
	reconnects []chan bool
}

// ExchangeArgs holds arguments for ExchangeDeclare.
type ExchangeArgs struct {
	Name       string
	Type       string
	Durable    bool
	AutoDelete bool
	Internal   bool
	NoWait     bool
	Args       map[string]interface{}
}

func (e ExchangeArgs) spread() (name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) {
	return e.Name, e.Type, e.Durable, e.AutoDelete, e.Internal, e.NoWait, e.Args
}

// QueueArgs holds arguments for QueueDeclare.
type QueueArgs struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       map[string]interface{}
}

func (q QueueArgs) spread() (name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) {
	return q.Name, q.Durable, q.AutoDelete, q.Exclusive, q.NoWait, q.Args
}

// Publishing is an alias for amqp.Publishing.
type Publishing = amqp.Publishing

// PublishArgs holds arguments for Publish.
type PublishArgs struct {
	Exchange   string
	RoutingKey string
	Mandatory  bool
	Immediate  bool
	Msg        Publishing
}

func (p PublishArgs) spread(ctx context.Context) (c context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) {
	return ctx, p.Exchange, p.RoutingKey, p.Mandatory, p.Immediate, p.Msg
}

// ConsumeArgs holds arguments for Consume.
type ConsumeArgs struct {
	Queue     string
	Consumer  string
	AutoAck   bool
	Exclusive bool
	NoLocal   bool
	NoWait    bool
	Args      map[string]interface{}
}

func (c ConsumeArgs) spread() (queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) {
	return c.Queue, c.Consumer, c.AutoAck, c.Exclusive, c.NoLocal, c.NoWait, c.Args
}

// QueueBindArgs holds arguments for QueueBind.
type QueueBindArgs struct {
	Queue      string
	Exchange   string
	RoutingKey string
	NoWait     bool
	Args       map[string]interface{}
}

func (q QueueBindArgs) spread() (queue, key, exchange string, noWait bool, args amqp.Table) {
	return q.Queue, q.RoutingKey, q.Exchange, q.NoWait, q.Args
}
