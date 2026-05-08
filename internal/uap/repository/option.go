package repository

import (
	"encoding/json"
	"time"

	"ingest-srv/internal/model"
)

type MarkRawBatchDownloadedOptions struct {
	RawBatchID string
}

type MarkRawBatchParsedOptions struct {
	RawBatchID         string
	ParsedAt           time.Time
	PublishRecordCount int
	PublishStatus      model.PublishStatus
	PublishError       string
	RawMetadata        json.RawMessage
}

type MarkRawBatchFailedOptions struct {
	RawBatchID   string
	ErrorMessage string
	PublishError string
	RawMetadata  json.RawMessage
}
