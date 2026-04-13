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
	}
	FacebookTasksQueue = rmq.QueueArgs{
		Name:    constants.QueueFacebookTasks,
		Durable: true,
	}
	YoutubeTasksQueue = rmq.QueueArgs{
		Name:    constants.QueueYouTubeTasks,
		Durable: true,
	}
	IngestDryrunCompletionsQueue = rmq.QueueArgs{
		Name:    constants.QueueIngestDryrunCompletions,
		Durable: true,
	}
)
