package models

import "time"

// CrawlRequest là payload inbound gốc trước khi collector chuẩn hóa.
type CrawlRequest struct {
	JobID       string         `json:"job_id"`
	TaskType    TaskType       `json:"task_type"`
	Payload     map[string]any `json:"payload"`
	TimeRange   int            `json:"time_range,omitempty"`
	Attempt     int            `json:"attempt,omitempty"`
	MaxAttempts int            `json:"max_attempts,omitempty"`
	EmittedAt   time.Time      `json:"emitted_at"`
}
