package http

import (
	"strings"

	"ingest-srv/internal/execution"
)

type dispatchReq struct {
	DataSourceID string
	TargetID     string
}

func (r dispatchReq) validate() error {
	if strings.TrimSpace(r.DataSourceID) == "" || strings.TrimSpace(r.TargetID) == "" {
		return errWrongPath
	}
	return nil
}

func (r dispatchReq) toInput() execution.DispatchTargetManuallyInput {
	return execution.DispatchTargetManuallyInput{
		DataSourceID: strings.TrimSpace(r.DataSourceID),
		TargetID:     strings.TrimSpace(r.TargetID),
	}
}

type dispatchResp struct {
	ScheduledJobID string `json:"scheduled_job_id"`
	ExternalTaskID string `json:"external_task_id"`
	TaskID         string `json:"task_id"`
	Queue          string `json:"queue"`
	Action         string `json:"action"`
	Status         string `json:"status"`
}

func (h *handler) newDispatchResp(o execution.DispatchTargetManuallyOutput) dispatchResp {
	return dispatchResp{
		ScheduledJobID: o.ScheduledJobID,
		ExternalTaskID: o.ExternalTaskID,
		TaskID:         o.TaskID,
		Queue:          o.Queue,
		Action:         o.Action,
		Status:         o.Status,
	}
}
