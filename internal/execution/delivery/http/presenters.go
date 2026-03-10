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
	Status         string `json:"status"`
	TaskCount      int    `json:"task_count"`
	PublishedCount int    `json:"published_count"`
	FailedCount    int    `json:"failed_count"`
	Tasks          []dispatchTaskResp `json:"tasks"`
}

type dispatchTaskResp struct {
	ExternalTaskID string `json:"external_task_id"`
	TaskID         string `json:"task_id"`
	Queue          string `json:"queue"`
	Action         string `json:"action"`
	Status         string `json:"status"`
	Keyword        string `json:"keyword,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

func (h *handler) newDispatchResp(o execution.DispatchTargetManuallyOutput) dispatchResp {
	tasks := make([]dispatchTaskResp, 0, len(o.Tasks))
	for _, task := range o.Tasks {
		tasks = append(tasks, dispatchTaskResp{
			ExternalTaskID: task.ExternalTaskID,
			TaskID:         task.TaskID,
			Queue:          task.Queue,
			Action:         task.Action,
			Status:         task.Status,
			Keyword:        task.Keyword,
			ErrorMessage:   task.ErrorMessage,
		})
	}

	return dispatchResp{
		ScheduledJobID: o.ScheduledJobID,
		Status:         o.Status,
		TaskCount:      o.TaskCount,
		PublishedCount: o.PublishedCount,
		FailedCount:    o.FailedCount,
		Tasks:          tasks,
	}
}
