package rabbitmq

import rmq "github.com/smap-hcmut/shared-libs/go/rabbitmq"

const (
	TikTokTasksQueueName                = "tiktok_tasks"
	FacebookTasksQueueName              = "facebook_tasks"
	YoutubeTasksQueueName               = "youtube_tasks"
	IngestDryrunCompletionsQueueName    = "ingest_dryrun_completions"
	IngestDryrunCompletionsConsumerName = "ingest-dryrun-completion-consumer"

	TikTokTasksExchangeName   = "ingest_tiktok_tasks_exc"
	FacebookTasksExchangeName = "ingest_facebook_tasks_exc"
	YoutubeTasksExchangeName  = "ingest_youtube_tasks_exc"

	TikTokTasksRoutingKey   = TikTokTasksQueueName
	FacebookTasksRoutingKey = FacebookTasksQueueName
	YoutubeTasksRoutingKey  = YoutubeTasksQueueName
)

var (
	TikTokTasksExchange = rmq.ExchangeArgs{
		Name:       TikTokTasksExchangeName,
		Type:       rmq.ExchangeTypeDirect,
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
	}
	FacebookTasksExchange = rmq.ExchangeArgs{
		Name:       FacebookTasksExchangeName,
		Type:       rmq.ExchangeTypeDirect,
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
	}
	YoutubeTasksExchange = rmq.ExchangeArgs{
		Name:       YoutubeTasksExchangeName,
		Type:       rmq.ExchangeTypeDirect,
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
	}
	TikTokTasksQueue = rmq.QueueArgs{
		Name:    TikTokTasksQueueName,
		Durable: true,
	}
	FacebookTasksQueue = rmq.QueueArgs{
		Name:    FacebookTasksQueueName,
		Durable: true,
	}
	YoutubeTasksQueue = rmq.QueueArgs{
		Name:    YoutubeTasksQueueName,
		Durable: true,
	}
	IngestDryrunCompletionsQueue = rmq.QueueArgs{
		Name:    IngestDryrunCompletionsQueueName,
		Durable: true,
	}
)
