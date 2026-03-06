package repository

import (
	"encoding/json"

	"ingest-srv/pkg/paginator"
)

// CreateDataSourceOptions contains the data needed to insert a new data source.
type CreateDataSourceOptions struct {
	ProjectID              string
	Name                   string
	Description            string
	SourceType             string
	SourceCategory         string
	Config                 json.RawMessage
	AccountRef             json.RawMessage
	MappingRules           json.RawMessage
	CrawlMode              string // nullable: empty string = NULL
	CrawlIntervalMinutes   int    // nullable: 0 = NULL
	WebhookID              string
	WebhookSecretEncrypted string
	CreatedBy              string
}

// GetOneDataSourceOptions contains filters for fetching a single data source.
// All non-empty fields are applied as AND conditions.
type GetOneDataSourceOptions struct {
	ID        string
	ProjectID string
	WebhookID string
	Name      string
}

// GetDataSourcesOptions contains filters and pagination for listing data sources.
type GetDataSourcesOptions struct {
	ProjectID      string
	Status         string
	SourceType     string
	SourceCategory string
	CrawlMode      string
	Name           string // ILIKE search
	Paginator      paginator.PaginateQuery
}

// ListDataSourcesOptions contains filters for listing data sources without pagination.
type ListDataSourcesOptions struct {
	ProjectID      string
	Status         string
	SourceType     string
	SourceCategory string
	CrawlMode      string
	Limit          int
}

// UpdateDataSourceOptions contains the data for updating a data source.
// Only non-zero/non-empty fields are applied.
type UpdateDataSourceOptions struct {
	ID                     string
	Name                   string
	Description            string
	Status                 string
	Config                 json.RawMessage
	AccountRef             json.RawMessage
	MappingRules           json.RawMessage
	OnboardingStatus       string
	DryrunStatus           string
	DryrunLastResultID     string
	CrawlMode              string
	CrawlIntervalMinutes   *int // pointer to distinguish 0 from unset
	WebhookID              string
	WebhookSecretEncrypted string
}

// --- CrawlTarget Options ---

// CreateTargetOptions contains data needed to insert a new crawl target.
type CreateTargetOptions struct {
	DataSourceID         string
	TargetType           string
	Value                string
	Label                string
	PlatformMeta         json.RawMessage
	IsActive             bool
	Priority             int
	CrawlIntervalMinutes int // 0 = use DB default (11)
}

// ListTargetsOptions contains filters for listing crawl targets of a data source.
type ListTargetsOptions struct {
	DataSourceID string
	TargetType   string
	IsActive     *bool // nil = all, true = active only, false = inactive only
}

// UpdateTargetOptions contains data for updating a crawl target.
// Only non-zero/non-empty fields are applied.
type UpdateTargetOptions struct {
	ID                   string
	Value                string
	Label                string
	PlatformMeta         json.RawMessage
	IsActive             *bool
	Priority             *int
	CrawlIntervalMinutes *int
}
