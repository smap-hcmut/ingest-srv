package rabbitmq

import rmq "ingest-srv/pkg/rabbitmq"

const (
	TikTokTasksQueueName              = "tiktok_tasks"
	FacebookTasksQueueName            = "facebook_tasks"
	YoutubeTasksQueueName             = "youtube_tasks"
	IngestTaskCompletionsQueueName    = "ingest_task_completions"
	IngestTaskCompletionsConsumerName = "ingest-execution-completion-consumer"
)

var (
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
	IngestTaskCompletionsQueue = rmq.QueueArgs{
		Name:    IngestTaskCompletionsQueueName,
		Durable: true,
	}
)
