package http

import (
	"encoding/json"
	"strings"
	"time"

	"ingest-srv/internal/dryrun"
	"ingest-srv/internal/model"
	"ingest-srv/pkg/paginator"
)

const timeFormat = "2006-01-02T15:04:05Z07:00"

type triggerReq struct {
	SourceID    string `json:"-"`
	TargetID    string `json:"target_id,omitempty" example:"660e8400-e29b-41d4-a716-446655440002"`
	SampleLimit *int   `json:"sample_limit,omitempty" example:"10"`
	Force       bool   `json:"force"`
}

func (r triggerReq) validate() error {
	if strings.TrimSpace(r.SourceID) == "" {
		return errWrongBody
	}
	if r.SampleLimit != nil && *r.SampleLimit <= 0 {
		return errInvalidSampleLimit
	}
	return nil
}

func (r triggerReq) toInput() dryrun.TriggerInput {
	return dryrun.TriggerInput{
		SourceID:    strings.TrimSpace(r.SourceID),
		TargetID:    strings.TrimSpace(r.TargetID),
		SampleLimit: r.SampleLimit,
		Force:       r.Force,
	}
}

type latestReq struct {
	SourceID string `form:"-"`
	TargetID string `form:"target_id"`
}

func (r latestReq) validate() error {
	if strings.TrimSpace(r.SourceID) == "" {
		return errWrongBody
	}
	return nil
}

func (r latestReq) toInput() dryrun.GetLatestInput {
	return dryrun.GetLatestInput{
		SourceID: strings.TrimSpace(r.SourceID),
		TargetID: strings.TrimSpace(r.TargetID),
	}
}

type historyReq struct {
	SourceID string `form:"-"`
	TargetID string `form:"target_id"`
	paginator.PaginateQuery
}

func (r historyReq) validate() error {
	if strings.TrimSpace(r.SourceID) == "" {
		return errWrongBody
	}
	return nil
}

func (r historyReq) toInput() dryrun.ListHistoryInput {
	return dryrun.ListHistoryInput{
		SourceID:  strings.TrimSpace(r.SourceID),
		TargetID:  strings.TrimSpace(r.TargetID),
		Paginator: r.PaginateQuery,
	}
}

type dryrunResultResp struct {
	ID           string          `json:"id"`
	SourceID     string          `json:"source_id"`
	ProjectID    string          `json:"project_id"`
	TargetID     string          `json:"target_id,omitempty"`
	Status       string          `json:"status"`
	SampleCount  int             `json:"sample_count"`
	TotalFound   *int            `json:"total_found,omitempty"`
	SampleData   json.RawMessage `json:"sample_data,omitempty" swaggertype:"array,object"`
	Warnings     json.RawMessage `json:"warnings,omitempty" swaggertype:"array,object"`
	ErrorMessage string          `json:"error_message,omitempty"`
	RequestedBy  string          `json:"requested_by,omitempty"`
	StartedAt    *string         `json:"started_at,omitempty"`
	CompletedAt  *string         `json:"completed_at,omitempty"`
	CreatedAt    string          `json:"created_at"`
}

type dataSourceSnapshotResp struct {
	ID                 string  `json:"id"`
	Status             string  `json:"status"`
	DryrunStatus       string  `json:"dryrun_status"`
	DryrunLastResultID string  `json:"dryrun_last_result_id,omitempty"`
	CrawlMode          *string `json:"crawl_mode,omitempty"`
}

type triggerResp struct {
	Result     dryrunResultResp       `json:"dryrun_result"`
	DataSource dataSourceSnapshotResp `json:"data_source"`
}

type latestResp struct {
	Result dryrunResultResp `json:"dryrun_result"`
}

type historyResp struct {
	Results   []dryrunResultResp          `json:"dryrun_results"`
	Paginator paginator.PaginatorResponse `json:"paginator"`
}

func (h *handler) newTriggerResp(o dryrun.TriggerOutput) triggerResp {
	return triggerResp{
		Result:     toDryrunResultResp(o.Result),
		DataSource: toDataSourceSnapshotResp(o.DataSource),
	}
}

func (h *handler) newLatestResp(o dryrun.GetLatestOutput) latestResp {
	return latestResp{Result: toDryrunResultResp(o.Result)}
}

func (h *handler) newHistoryResp(o dryrun.ListHistoryOutput) historyResp {
	items := make([]dryrunResultResp, len(o.Results))
	for i, result := range o.Results {
		items[i] = toDryrunResultResp(result)
	}
	return historyResp{
		Results:   items,
		Paginator: o.Paginator.ToResponse(),
	}
}

func toDryrunResultResp(result model.DryrunResult) dryrunResultResp {
	resp := dryrunResultResp{
		ID:           result.ID,
		SourceID:     result.SourceID,
		ProjectID:    result.ProjectID,
		TargetID:     result.TargetID,
		Status:       string(result.Status),
		SampleCount:  result.SampleCount,
		TotalFound:   result.TotalFound,
		SampleData:   result.SampleData,
		Warnings:     result.Warnings,
		ErrorMessage: result.ErrorMessage,
		RequestedBy:  result.RequestedBy,
		CreatedAt:    result.CreatedAt.Format(timeFormat),
	}
	resp.StartedAt = formatTimePtr(result.StartedAt)
	resp.CompletedAt = formatTimePtr(result.CompletedAt)
	return resp
}

func toDataSourceSnapshotResp(source model.DataSource) dataSourceSnapshotResp {
	resp := dataSourceSnapshotResp{
		ID:                 source.ID,
		Status:             string(source.Status),
		DryrunStatus:       string(source.DryrunStatus),
		DryrunLastResultID: source.DryrunLastResultID,
	}
	if source.CrawlMode != nil {
		mode := string(*source.CrawlMode)
		resp.CrawlMode = &mode
	}
	return resp
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(timeFormat)
	return &s
}
