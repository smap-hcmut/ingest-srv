package usecase

import (
	"context"

	"smap-collector/internal/models"
	"smap-collector/internal/state"
)

// InitState khởi tạo state cho project mới.
func (uc *implUseCase) InitState(ctx context.Context, projectID string) error {
	if projectID == "" {
		return state.ErrInvalidProjectID
	}

	key := state.BuildStateKey(projectID)

	// Check if state already exists
	exists, err := uc.repo.Exists(ctx, key)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to check state existence: project_id=%s, error=%v", projectID, err)
		return err
	}

	if exists {
		// Get current state to check if it's terminal
		currentState, err := uc.repo.GetState(ctx, key)
		if err != nil {
			return err
		}

		// Only allow re-init if previous state was terminal
		if currentState != nil && !currentState.Status.IsTerminal() {
			uc.l.Warnf(ctx, "State already exists and is active: project_id=%s, status=%s",
				projectID, currentState.Status)
			return state.ErrStateAlreadyExists
		}
	}

	// Initialize new state
	newState := models.NewProjectState()
	if err := uc.repo.InitState(ctx, key, newState, uc.opts.TTL); err != nil {
		uc.l.Errorf(ctx, "Failed to init state: project_id=%s, error=%v", projectID, err)
		return err
	}

	uc.l.Infof(ctx, "Initialized state: project_id=%s, status=%s", projectID, newState.Status)
	return nil
}

// UpdateTotal cập nhật tổng số items và chuyển status sang CRAWLING.
func (uc *implUseCase) UpdateTotal(ctx context.Context, projectID string, total int64) error {
	if projectID == "" {
		return state.ErrInvalidProjectID
	}
	if total < 0 {
		return state.ErrInvalidTotal
	}

	key := state.BuildStateKey(projectID)

	// Update total and status in one operation
	fields := map[string]any{
		state.FieldTotal:  total,
		state.FieldStatus: string(models.ProjectStatusCrawling),
	}

	if err := uc.repo.SetFields(ctx, key, fields); err != nil {
		uc.l.Errorf(ctx, "Failed to update total: project_id=%s, total=%d, error=%v",
			projectID, total, err)
		return err
	}

	uc.l.Infof(ctx, "Updated total: project_id=%s, total=%d, status=CRAWLING", projectID, total)
	return nil
}

// IncrementDone tăng counter done lên 1.
func (uc *implUseCase) IncrementDone(ctx context.Context, projectID string) error {
	if projectID == "" {
		return state.ErrInvalidProjectID
	}

	key := state.BuildStateKey(projectID)

	newValue, err := uc.repo.IncrementField(ctx, key, state.FieldDone, 1)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to increment done: project_id=%s, error=%v", projectID, err)
		return err
	}

	uc.l.Debugf(ctx, "Incremented done: project_id=%s, new_value=%d", projectID, newValue)
	return nil
}

// IncrementErrors tăng counter errors lên 1.
func (uc *implUseCase) IncrementErrors(ctx context.Context, projectID string) error {
	if projectID == "" {
		return state.ErrInvalidProjectID
	}

	key := state.BuildStateKey(projectID)

	newValue, err := uc.repo.IncrementField(ctx, key, state.FieldErrors, 1)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to increment errors: project_id=%s, error=%v", projectID, err)
		return err
	}

	uc.l.Debugf(ctx, "Incremented errors: project_id=%s, new_value=%d", projectID, newValue)
	return nil
}

// UpdateStatus cập nhật status của project.
func (uc *implUseCase) UpdateStatus(ctx context.Context, projectID string, status models.ProjectStatus) error {
	if projectID == "" {
		return state.ErrInvalidProjectID
	}
	if status == "" {
		return state.ErrInvalidStatus
	}

	key := state.BuildStateKey(projectID)

	if err := uc.repo.SetField(ctx, key, state.FieldStatus, string(status)); err != nil {
		uc.l.Errorf(ctx, "Failed to update status: project_id=%s, status=%s, error=%v",
			projectID, status, err)
		return err
	}

	uc.l.Infof(ctx, "Updated status: project_id=%s, status=%s", projectID, status)
	return nil
}

// GetState lấy state hiện tại của project.
func (uc *implUseCase) GetState(ctx context.Context, projectID string) (*models.ProjectState, error) {
	if projectID == "" {
		return nil, state.ErrInvalidProjectID
	}

	key := state.BuildStateKey(projectID)

	s, err := uc.repo.GetState(ctx, key)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to get state: project_id=%s, error=%v", projectID, err)
		return nil, err
	}

	if s == nil {
		return nil, state.ErrStateNotFound
	}

	return s, nil
}

// CheckAndUpdateCompletion kiểm tra và update status nếu complete.
func (uc *implUseCase) CheckAndUpdateCompletion(ctx context.Context, projectID string) (bool, error) {
	if projectID == "" {
		return false, state.ErrInvalidProjectID
	}

	// Get current state
	s, err := uc.GetState(ctx, projectID)
	if err != nil {
		return false, err
	}

	// Check if complete: done + errors >= total && total > 0
	if s.IsComplete() {
		// Update status to DONE
		if err := uc.UpdateStatus(ctx, projectID, models.ProjectStatusDone); err != nil {
			uc.l.Errorf(ctx, "Failed to update completion status: project_id=%s, error=%v", projectID, err)
			return false, ErrUpdateCompletionFailed
		}

		uc.l.Infof(ctx, "Project completed: project_id=%s, total=%d, done=%d, errors=%d",
			projectID, s.Total, s.Done, s.Errors)
		return true, nil
	}

	return false, nil
}

// StoreUserMapping lưu mapping project_id -> user_id.
func (uc *implUseCase) StoreUserMapping(ctx context.Context, projectID, userID string) error {
	if projectID == "" {
		return state.ErrInvalidProjectID
	}
	if userID == "" {
		return ErrInvalidUserID
	}

	key := state.BuildUserMappingKey(projectID)

	if err := uc.repo.SetString(ctx, key, userID, uc.opts.TTL); err != nil {
		uc.l.Errorf(ctx, "Failed to store user mapping: project_id=%s, user_id=%s, error=%v",
			projectID, userID, err)
		return err
	}

	uc.l.Debugf(ctx, "Stored user mapping: project_id=%s, user_id=%s", projectID, userID)
	return nil
}

// GetUserID lấy user_id từ project_id.
func (uc *implUseCase) GetUserID(ctx context.Context, projectID string) (string, error) {
	if projectID == "" {
		return "", state.ErrInvalidProjectID
	}

	key := state.BuildUserMappingKey(projectID)

	userID, err := uc.repo.GetString(ctx, key)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to get user mapping: project_id=%s, error=%v", projectID, err)
		return "", state.ErrUserMappingNotFound
	}

	if userID == "" {
		return "", state.ErrUserMappingNotFound
	}

	return userID, nil
}
