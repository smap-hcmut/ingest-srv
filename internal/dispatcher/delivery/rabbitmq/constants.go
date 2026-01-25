package rabbitmq

import pkgRabbit "smap-collector/pkg/rabbitmq"

const (
	// ============================================================================
	// SMAP Events Exchange (Event-Driven Architecture)
	// Central exchange for all SMAP events using topic routing
	// ============================================================================
	ExchangeSMAPEvents = "smap.events"

	// Project Created Event - consumed by Collector from Project Service
	QueueProjectCreated      = "collector.project.created"
	RoutingKeyProjectCreated = "project.created"

	// Note: data.collected event is published by Crawler services, not Collector

	// ============================================================================
	// Legacy Inbound (Ingress) - DEPRECATED
	// Will be removed after migration to event-driven architecture
	// ============================================================================
	ExchangeInbound   = "collector.inbound"
	QueueInbound      = "collector.inbound.queue"
	RoutingKeyInbound = "crawler.#"

	// ============================================================================
	// Worker Queues (Outbound to scrapers)
	// ============================================================================

	// TikTok
	ExchangeTikTok   = "tiktok_exchange"
	QueueTikTok      = "tiktok_crawl_queue"
	RoutingKeyTikTok = "tiktok.crawl"

	// YouTube
	ExchangeYouTube   = "youtube_exchange"
	QueueYouTube      = "youtube_crawl_queue"
	RoutingKeyYouTube = "youtube.crawl"
)

var (
	// ============================================================================
	// SMAP Events Exchange Args (Event-Driven Architecture)
	// ============================================================================

	// SMAPEventsExchangeArgs for the central event exchange
	SMAPEventsExchangeArgs = pkgRabbit.ExchangeArgs{
		Name:       ExchangeSMAPEvents,
		Type:       pkgRabbit.ExchangeTypeTopic,
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
	}

	// ============================================================================
	// Legacy Inbound Exchange Args - DEPRECATED
	// ============================================================================

	InboundExchangeArgs = pkgRabbit.ExchangeArgs{
		Name:       ExchangeInbound,
		Type:       pkgRabbit.ExchangeTypeTopic,
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
	}

	// ============================================================================
	// Worker Exchange Args
	// ============================================================================

	TikTokExchangeArgs = pkgRabbit.ExchangeArgs{
		Name:       ExchangeTikTok,
		Type:       pkgRabbit.ExchangeTypeDirect,
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
	}

	YouTubeExchangeArgs = pkgRabbit.ExchangeArgs{
		Name:       ExchangeYouTube,
		Type:       pkgRabbit.ExchangeTypeDirect,
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
	}
)
