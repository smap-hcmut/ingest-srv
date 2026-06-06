package rabbitmq

import (
	"github.com/smap-hcmut/shared-libs/go/constants"
	rmq "github.com/smap-hcmut/shared-libs/go/rabbitmq"
)

const (
	IngestDryrunCompletionsConsumerName = "ingest-dryrun-completion-consumer"

	TikTokTasksRoutingKey   = constants.QueueTikTokTasks
	FacebookTasksRoutingKey = constants.QueueFacebookTasks
	YoutubeTasksRoutingKey  = constants.QueueYouTubeTasks
)

// queueWithDLX mirrors execution/rabbitmq/constants.go so the dryrun producer
// declares the same DLX arguments. Without this the second boot path would
// race PRECONDITION_FAILED against the queue created by the execution
// producer.
func queueWithDLX(routingKey string) map[string]interface{} {
	return map[string]interface{}{
		"x-dead-letter-exchange":    "ingest_dlx",
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
	IngestDryrunCompletionsQueue = rmq.QueueArgs{
		Name:    constants.QueueIngestDryrunCompletions,
		Durable: true,
	}
)
