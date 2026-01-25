package consumer

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"

	pkgRabbit "smap-collector/pkg/rabbitmq"
)

type WorkerFunc func(msg amqp.Delivery)

func (c Consumer) consume(exchange pkgRabbit.ExchangeArgs, queueName string, routingKey string, workerFunc WorkerFunc) {
	defer catchPanic()
	ctx := context.Background()

	ch, err := c.conn.Channel()
	if err != nil {
		panic(err)
	}
	defer ch.Close()

	if exchange.Name != "" {
		if err := ch.ExchangeDeclare(exchange); err != nil {
			panic(err)
		}
	}

	q, err := ch.QueueDeclare(pkgRabbit.QueueArgs{
		Name:    queueName,
		Durable: true,
	})
	if err != nil {
		panic(err)
	}

	if err := ch.QueueBind(pkgRabbit.QueueBindArgs{
		Queue:      q.Name,
		Exchange:   exchange.Name,
		RoutingKey: routingKey,
	}); err != nil {
		panic(err)
	}

	msgs, err := ch.Consume(pkgRabbit.ConsumeArgs{
		Queue: q.Name,
	})
	if err != nil {
		panic(err)
	}

	c.l.Infof(ctx, "Results queue %s is being consumed", q.Name)

	var forever chan bool
	go func() {
		for msg := range msgs {
			workerFunc(msg)
		}
	}()

	<-forever
}

func catchPanic() {
	if r := recover(); r != nil {
		// simple panic guard to keep consumer alive
	}
}
