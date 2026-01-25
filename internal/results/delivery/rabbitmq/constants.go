package rabbitmq

import pkgRabbit "smap-collector/pkg/rabbitmq"

const (
	// TikTok Results - matches tiktok-scraper publisher config
	ExchangeTikTokResults   = "tiktok_exchange"
	QueueTikTokResults      = "tiktok_result_queue"
	RoutingKeyTikTokResults = "tiktok.res"

	// YouTube Results - matches youtube-scraper publisher config
	ExchangeYouTubeResults   = "youtube_exchange"
	QueueYouTubeResults      = "youtube_result_queue"
	RoutingKeyYouTubeResults = "youtube.res"
)

var (
	TikTokResultsExchangeArgs = pkgRabbit.ExchangeArgs{
		Name:       ExchangeTikTokResults,
		Type:       pkgRabbit.ExchangeTypeDirect,
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
	}

	YouTubeResultsExchangeArgs = pkgRabbit.ExchangeArgs{
		Name:       ExchangeYouTubeResults,
		Type:       pkgRabbit.ExchangeTypeDirect,
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
	}
)
