package repository

import (
	"encoding/json"
	"time"
)

type MarkRawBatchDownloadedOptions struct {
	RawBatchID string
}

type MarkRawBatchParsedOptions struct {
	RawBatchID          string
	ParsedAt            time.Time
	PublishRecordCount  int
	RawMetadata         json.RawMessage
}

type MarkRawBatchFailedOptions struct {
	RawBatchID    string
	ErrorMessage  string
	PublishError  string
	RawMetadata   json.RawMessage
}
