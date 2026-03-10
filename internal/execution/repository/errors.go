package repository

import "errors"

var (
	ErrDataSourceNotFound    = errors.New("execution.repository: datasource not found")
	ErrTargetNotFound        = errors.New("execution.repository: target not found")
	ErrExternalTaskNotFound  = errors.New("execution.repository: external task not found")
	ErrRawBatchAlreadyExists = errors.New("execution.repository: raw batch already exists")
	ErrListDueTargets       = errors.New("execution.repository: failed to list due targets")
	ErrClaimTarget          = errors.New("execution.repository: failed to claim target")
	ErrCreateDispatch       = errors.New("execution.repository: failed to create dispatch")
	ErrUpdateDispatch       = errors.New("execution.repository: failed to update dispatch")
	ErrGetCompletionTask    = errors.New("execution.repository: failed to get completion task")
	ErrCompleteTask         = errors.New("execution.repository: failed to persist completion")
)
