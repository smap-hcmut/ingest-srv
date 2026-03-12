package datasource

import (
	"encoding/json"

	"ingest-srv/internal/model"

	"github.com/smap-hcmut/shared-libs/go/paginator"
)

// CreateInput is the input for creating a data source.
type CreateInput struct {
	ProjectID              string
	Name                   string
	Description            string
	SourceType             string
	SourceCategory         string
	Config                 json.RawMessage
	AccountRef             json.RawMessage
	MappingRules           json.RawMessage
	CrawlMode              string
	CrawlIntervalMinutes   int
	WebhookID              string
	WebhookSecretEncrypted string
}

// CreateOutput is the output after creating a data source.
type CreateOutput struct {
	DataSource model.DataSource
}

// DetailOutput is the output for getting data source detail.
type DetailOutput struct {
	DataSource model.DataSource
}

// ListInput is the input for listing data sources.
type ListInput struct {
	ProjectID      string
	Status         string
	SourceType     string
	SourceCategory string
	CrawlMode      string
	Name           string
	Paginator      paginator.PaginateQuery
}

// ListOutput is the output for listing data sources.
type ListOutput struct {
	DataSources []model.DataSource
	Paginator   paginator.Paginator
}

// UpdateInput is the input for updating a data source.
type UpdateInput struct {
	ID           string
	Name         string
	Description  string
	Config       json.RawMessage
	AccountRef   json.RawMessage
	MappingRules json.RawMessage
}

// UpdateOutput is the output after updating a data source.
type UpdateOutput struct {
	DataSource model.DataSource
}

// ActivateInput transitions a datasource into ACTIVE when preconditions are met.
type ActivateInput struct {
	ID string
}

// ActivateOutput is the output after activating a datasource.
type ActivateOutput struct {
	DataSource model.DataSource
}

// PauseInput transitions a datasource into PAUSED.
type PauseInput struct {
	ID string
}

// PauseOutput is the output after pausing a datasource.
type PauseOutput struct {
	DataSource model.DataSource
}

// ResumeInput transitions a datasource from PAUSED back to ACTIVE.
type ResumeInput struct {
	ID string
}

// ResumeOutput is the output after resuming a datasource.
type ResumeOutput struct {
	DataSource model.DataSource
}

// UpdateCrawlModeInput changes crawl_mode for a crawl datasource.
type UpdateCrawlModeInput struct {
	ID          string
	CrawlMode   string
	TriggerType string
	Reason      string
	EventRef    string
}

// UpdateCrawlModeOutput is the output after changing crawl_mode.
type UpdateCrawlModeOutput struct {
	DataSource model.DataSource
}

// --- CrawlTarget Types ---

// CreateTargetGroupInput is the shared input for creating a grouped crawl target.
type CreateTargetGroupInput struct {
	DataSourceID         string
	Values               []string
	Label                string
	PlatformMeta         json.RawMessage
	IsActive             bool
	Priority             int
	CrawlIntervalMinutes int
}

// CreateTargetOutput is the output after creating a crawl target.
type CreateTargetOutput struct {
	Target model.CrawlTarget
}

// DetailTargetInput is the input for getting crawl target detail.
type DetailTargetInput struct {
	DataSourceID string
	ID           string
}

// DetailTargetOutput is the output for getting target detail.
type DetailTargetOutput struct {
	Target model.CrawlTarget
}

// ListTargetsInput is the input for listing crawl targets.
type ListTargetsInput struct {
	DataSourceID string
	TargetType   string
	IsActive     *bool
}

// ListTargetsOutput is the output for listing crawl targets.
type ListTargetsOutput struct {
	Targets []model.CrawlTarget
}

// UpdateTargetInput is the input for updating a crawl target.
type UpdateTargetInput struct {
	DataSourceID         string
	ID                   string
	Values               []string
	Label                string
	PlatformMeta         json.RawMessage
	IsActive             *bool
	Priority             *int
	CrawlIntervalMinutes *int
}

// UpdateTargetOutput is the output after updating a crawl target.
type UpdateTargetOutput struct {
	Target model.CrawlTarget
}

// DeleteTargetInput is the input for deleting a crawl target.
type DeleteTargetInput struct {
	DataSourceID string
	ID           string
}
