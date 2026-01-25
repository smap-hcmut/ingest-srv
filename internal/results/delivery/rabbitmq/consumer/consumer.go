package consumer

import (
	"smap-collector/internal/results/delivery/rabbitmq"
)

// Consume starts consuming from TikTok and YouTube result queues.
func (c Consumer) Consume() {
	// Consume TikTok results
	go c.consume(
		rabbitmq.TikTokResultsExchangeArgs,
		rabbitmq.QueueTikTokResults,
		rabbitmq.RoutingKeyTikTokResults,
		c.resultWorker,
	)

	// Consume YouTube results
	go c.consume(
		rabbitmq.YouTubeResultsExchangeArgs,
		rabbitmq.QueueYouTubeResults,
		rabbitmq.RoutingKeyYouTubeResults,
		c.resultWorker,
	)
}
