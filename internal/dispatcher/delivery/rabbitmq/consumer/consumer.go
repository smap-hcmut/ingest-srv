package consumer

import (
	"smap-collector/internal/dispatcher/delivery/rabbitmq"
)

// Consume start consume inbound queue.
func (c Consumer) Consume() {
	go c.consume(rabbitmq.InboundExchangeArgs, rabbitmq.QueueInbound, rabbitmq.RoutingKeyInbound, c.dispatchWorker)
}
