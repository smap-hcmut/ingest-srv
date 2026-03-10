package rabbitmq

import (
	"context"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (c *connectionImpl) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
	c.isRetrying = false
}

func (c *connectionImpl) IsReady() bool {
	return c.conn != nil && !c.conn.IsClosed()
}

func (c *connectionImpl) IsClosed() bool {
	return !c.IsReady() && !c.isRetrying
}

func (c *connectionImpl) Channel() (IChannel, error) {
	ch, err := c.channel()
	if err != nil {
		return nil, err
	}
	chImpl := &channelImpl{conn: c, ch: ch}
	chImpl.listenNotifyReconnect()
	return chImpl, nil
}

func (c *connectionImpl) dial(url string, connChan chan *amqp.Connection, cancelChan chan bool) {
	count := 0
	for {
		select {
		case <-cancelChan:
			return
		default:
			log.Printf("Connecting to RabbitMQ, attempt: %d ...\n", count+1)
			conn, err := amqp.Dial(url)
			if err != nil {
				log.Printf("Connection to RabbitMQ failed: %v\n", err)
				time.Sleep(RetryConnectionDelay)
				count++
				continue
			}
			log.Println("Connected to RabbitMQ!")
			connChan <- conn
			return
		}
	}
}

func (c *connectionImpl) connectWithoutTimeout() error {
	connChan := make(chan *amqp.Connection)
	go c.dial(c.url, connChan, make(chan bool))
	conn := <-connChan
	c.conn = conn
	c.listenNotifyClose()
	return nil
}

func (c *connectionImpl) connect() error {
	connChan := make(chan *amqp.Connection)
	cancelChan := make(chan bool)
	go c.dial(c.url, connChan, cancelChan)
	select {
	case conn := <-connChan:
		c.conn = conn
		c.listenNotifyClose()
		return nil
	case <-time.After(RetryConnectionTimeout):
		cancelChan <- true
		return ErrConnectionTimeout
	}
}

func (c *connectionImpl) listenNotifyClose() {
	fn := c.connect
	if c.retryWithoutTimeout {
		fn = c.connectWithoutTimeout
	}
	notifyClose := make(chan *amqp.Error)
	c.conn.NotifyClose(notifyClose)
	go func() {
		for err := range notifyClose {
			if err != nil {
				c.conn = nil
				c.isRetrying = true
				log.Printf("Connection to RabbitMQ closed: %v\n", err)
				if err := fn(); err != nil {
					log.Printf("Connection to RabbitMQ failed: %v\n", err)
				}
				for _, reconnect := range c.reconnects {
					reconnect <- true
				}
				c.isRetrying = false
				return
			}
		}
	}()
}

func (c *connectionImpl) channel() (*amqp.Channel, error) {
	return c.conn.Channel()
}

func (c *connectionImpl) notifyReconnect(receiver chan bool) <-chan bool {
	c.reconnects = append(c.reconnects, receiver)
	return receiver
}

func (ch *channelImpl) ExchangeDeclare(exc ExchangeArgs) error {
	return ch.ch.ExchangeDeclare(exc.spread())
}

func (ch *channelImpl) QueueDeclare(queue QueueArgs) (amqp.Queue, error) {
	return ch.ch.QueueDeclare(queue.spread())
}

func (ch *channelImpl) QueueBind(queueBind QueueBindArgs) error {
	return ch.ch.QueueBind(queueBind.spread())
}

func (ch *channelImpl) Publish(ctx context.Context, publish PublishArgs) error {
	return ch.ch.PublishWithContext(publish.spread(ctx))
}

func (ch *channelImpl) Consume(consume ConsumeArgs) (<-chan amqp.Delivery, error) {
	return ch.ch.Consume(consume.spread())
}

func (ch *channelImpl) Close() error {
	return ch.ch.Close()
}

func (ch *channelImpl) NotifyReconnect(receiver chan bool) <-chan bool {
	ch.reconnects = append(ch.reconnects, receiver)
	return receiver
}

func (ch *channelImpl) listenNotifyReconnect() {
	reconnNoti := make(chan bool)
	ch.conn.notifyReconnect(reconnNoti)
	go func() {
		for {
			<-reconnNoti
			log.Println("Retry creating RabbitMQ channel...")
			channel, err := ch.conn.channel()
			if err != nil {
				log.Printf("RabbitMQ channel failed: %v\n", err)
				continue
			}
			_ = ch.ch.Close()
			ch.ch = channel
		}
	}()
}
