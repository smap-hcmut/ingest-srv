package rabbitmq

import (
	"github.com/smap-hcmut/shared-libs/go/constants"
	rmq "github.com/smap-hcmut/shared-libs/go/rabbitmq"
)

const (
	IngestTaskCompletionsConsumerName = "ingest-execution-completion-consumer"

	TikTokTasksRoutingKey   = constants.QueueTikTokTasks
	FacebookTasksRoutingKey = constants.QueueFacebookTasks
	YoutubeTasksRoutingKey  = constants.QueueYouTubeTasks

	// IngestDLXExchange is the topic exchange that catches messages rejected
	// from any task queue (nack with requeue=false, TTL expiry, or queue length
	// overflow). One exchange + per-queue dead-letter queues lets operators
	// triage failed dispatches without losing them.
	IngestDLXExchange = "ingest_dlx"

	TikTokTasksDLQ   = constants.QueueTikTokTasks + ".dlq"
	FacebookTasksDLQ = constants.QueueFacebookTasks + ".dlq"
	YouTubeTasksDLQ  = constants.QueueYouTubeTasks + ".dlq"
)

// queueWithDLX builds queue Args wiring the queue to IngestDLXExchange so any
// nack(requeue=false) (e.g. parse failure, permanent crawler error) is routed
// to the matching dead-letter queue instead of being silently dropped.
//
// NOTE: Pre-existing queues without these args must be deleted before the
// updated topology can take effect — RabbitMQ rejects QueueDeclare with
// PRECONDITION_FAILED if args differ from the live queue.
func queueWithDLX(routingKey string) map[string]interface{} {
	return map[string]interface{}{
		"x-dead-letter-exchange":    IngestDLXExchange,
		"x-dead-letter-routing-key": routingKey,
	}
}

var (
	TikTokTasksExchange = rmq.ExchangeArgs{
		Name:       constants.ExchangeTikTokTasks,
		Type:       rmq.ExchangeTypeDirect,
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
	}
	FacebookTasksExchange = rmq.ExchangeArgs{
		Name:       constants.ExchangeFacebookTasks,
		Type:       rmq.ExchangeTypeDirect,
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
	}
	YoutubeTasksExchange = rmq.ExchangeArgs{
		Name:       constants.ExchangeYouTubeTasks,
		Type:       rmq.ExchangeTypeDirect,
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
	}

	// DLXExchange is shared across all task platforms; topic type so a single
	// exchange can fan out to per-platform DLQs based on routing key.
	DLXExchange = rmq.ExchangeArgs{
		Name:       IngestDLXExchange,
		Type:       rmq.ExchangeTypeTopic,
		Durable:    true,
		AutoDelete: false,
	}

	TikTokTasksQueue = rmq.QueueArgs{
		Name:    constants.QueueTikTokTasks,
		Durable: true,
		Args:    queueWithDLX(TikTokTasksRoutingKey),
	}
	FacebookTasksQueue = rmq.QueueArgs{
		Name:    constants.QueueFacebookTasks,
		Durable: true,
		Args:    queueWithDLX(FacebookTasksRoutingKey),
	}
	YoutubeTasksQueue = rmq.QueueArgs{
		Name:    constants.QueueYouTubeTasks,
		Durable: true,
		Args:    queueWithDLX(YoutubeTasksRoutingKey),
	}
	IngestTaskCompletionsQueue = rmq.QueueArgs{
		Name:    constants.QueueIngestTaskCompletions,
		Durable: true,
	}

	TikTokTasksDLQQueue = rmq.QueueArgs{
		Name:    TikTokTasksDLQ,
		Durable: true,
	}
	FacebookTasksDLQQueue = rmq.QueueArgs{
		Name:    FacebookTasksDLQ,
		Durable: true,
	}
	YouTubeTasksDLQQueue = rmq.QueueArgs{
		Name:    YouTubeTasksDLQ,
		Durable: true,
	}
)
