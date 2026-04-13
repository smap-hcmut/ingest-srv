package dryrun

import (
	"time"

	"ingest-srv/internal/model"

	"github.com/smap-hcmut/shared-libs/go/constants"
	"github.com/smap-hcmut/shared-libs/go/paginator"
)

// TriggerInput triggers one async dryrun. For crawl sources, TargetID refers to one grouped crawl target.
type TriggerInput struct {
	SourceID    string
	TargetID    string
	SampleLimit *int
	Force       bool
}

// TriggerOutput returns the persisted dryrun result and updated datasource snapshot.
type TriggerOutput struct {
	Result     model.DryrunResult
	DataSource model.DataSource
}

type QueueName string

const (
	QueueNameTikTokTasks   QueueName = QueueName(constants.QueueTikTokTasks)
	QueueNameFacebookTasks QueueName = QueueName(constants.QueueFacebookTasks)
)

type ActionName string

const (
	ActionNamePostDetail ActionName = "post_detail"
	ActionNameFullFlow   ActionName = "full_flow"
)

type RuntimeKind string

const (
	RuntimeKindDryrun RuntimeKind = "dryrun"
)

type WarningCode string

const (
	WarningCodeNoSampleData         WarningCode = "no_sample_data_extracted"
	WarningCodeInvalidArtifact      WarningCode = "invalid_dryrun_artifact"
	WarningCodeMultiValueKeyword    WarningCode = "multi_value_keyword_target"
	WarningCodeObjectSampleFallback WarningCode = "object_sample_fallback"
)

const (
	ParamKeyRuntimeKind          = "runtime_kind"
	ParamKeyDryrunWarningCode    = "dryrun_warning_code"
	ParamKeyDryrunWarningMessage = "dryrun_warning_message"
)

// DispatchSpec is the normalized runtime dispatch shape for one dryrun.
type DispatchSpec struct {
	Queue  QueueName
	Action ActionName
	Params map[string]interface{}
}

// PublishDispatchInput is the usecase-facing payload passed to the RabbitMQ producer.
type PublishDispatchInput struct {
	Queue     QueueName
	TaskID    string
	Action    ActionName
	Params    map[string]interface{}
	CreatedAt time.Time
}

// HandleCompletionInput is the usecase-facing completion payload consumed by ingest.
type HandleCompletionInput struct {
	TaskID        string
	Status        string
	CompletedAt   string
	StorageBucket string
	StoragePath   string
	BatchID       string
	Checksum      string
	ItemCount     *int
	Error         string
	Metadata      map[string]interface{}
}

// GetLatestInput retrieves the latest dryrun result for a source or source-target(group) pair.
type GetLatestInput struct {
	SourceID string
	TargetID string
}

// GetLatestOutput contains the latest dryrun result.
type GetLatestOutput struct {
	Result model.DryrunResult
}

// ListHistoryInput retrieves paginated dryrun history.
type ListHistoryInput struct {
	SourceID  string
	TargetID  string
	Paginator paginator.PaginateQuery
}

// ListHistoryOutput contains paginated dryrun history.
type ListHistoryOutput struct {
	Results   []model.DryrunResult
	Paginator paginator.Paginator
}
