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

// ============================================================================
// Task-Level Methods (for completion check)
// ============================================================================

// SetTasksTotal set tổng số tasks và items expected, chuyển status sang PROCESSING.
func (uc *implUseCase) SetTasksTotal(ctx context.Context, projectID string, tasksTotal, itemsExpected int64) error {
	if projectID == "" {
		return state.ErrInvalidProjectID
	}
	if tasksTotal < 0 {
		return state.ErrInvalidTotal
	}

	key := state.BuildStateKey(projectID)

	// Update task-level, item-level fields and status in one operation
	fields := map[string]any{
		state.FieldTasksTotal:    tasksTotal,
		state.FieldItemsExpected: itemsExpected,
		state.FieldStatus:        string(models.ProjectStatusProcessing),
		// Also set legacy crawl_total for backward compatibility
		state.FieldCrawlTotal: tasksTotal,
	}

	if err := uc.repo.SetFields(ctx, key, fields); err != nil {
		uc.l.Errorf(ctx, "Failed to set tasks total: project_id=%s, tasks_total=%d, items_expected=%d, error=%v",
			projectID, tasksTotal, itemsExpected, err)
		return err
	}

	uc.l.Infof(ctx, "Set tasks total: project_id=%s, tasks_total=%d, items_expected=%d, status=PROCESSING",
		projectID, tasksTotal, itemsExpected)
	return nil
}

// IncrementTasksDone tăng counter tasks_done lên 1.
func (uc *implUseCase) IncrementTasksDone(ctx context.Context, projectID string) error {
	if projectID == "" {
		return state.ErrInvalidProjectID
	}

	key := state.BuildStateKey(projectID)

	newValue, err := uc.repo.IncrementField(ctx, key, state.FieldTasksDone, 1)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to increment tasks_done: project_id=%s, error=%v", projectID, err)
		return err
	}

	uc.l.Debugf(ctx, "Incremented tasks_done: project_id=%s, new_value=%d", projectID, newValue)
	return nil
}

// IncrementTasksErrors tăng counter tasks_errors lên 1.
func (uc *implUseCase) IncrementTasksErrors(ctx context.Context, projectID string) error {
	if projectID == "" {
		return state.ErrInvalidProjectID
	}

	key := state.BuildStateKey(projectID)

	newValue, err := uc.repo.IncrementField(ctx, key, state.FieldTasksErrors, 1)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to increment tasks_errors: project_id=%s, error=%v", projectID, err)
		return err
	}

	uc.l.Debugf(ctx, "Incremented tasks_errors: project_id=%s, new_value=%d", projectID, newValue)
	return nil
}

// ============================================================================
// Item-Level Methods (for progress display)
// ============================================================================

// IncrementItemsActualBy tăng counter items_actual lên N.
func (uc *implUseCase) IncrementItemsActualBy(ctx context.Context, projectID string, count int64) error {
	if projectID == "" {
		return state.ErrInvalidProjectID
	}
	if count <= 0 {
		return nil // Skip if count is 0 or negative
	}

	key := state.BuildStateKey(projectID)

	newValue, err := uc.repo.IncrementField(ctx, key, state.FieldItemsActual, count)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to increment items_actual: project_id=%s, count=%d, error=%v", projectID, count, err)
		return err
	}

	// Also increment legacy crawl_done for backward compatibility
	if _, err := uc.repo.IncrementField(ctx, key, state.FieldCrawlDone, count); err != nil {
		uc.l.Warnf(ctx, "Failed to increment legacy crawl_done: project_id=%s, count=%d, error=%v", projectID, count, err)
	}

	uc.l.Debugf(ctx, "Incremented items_actual: project_id=%s, count=%d, new_value=%d", projectID, count, newValue)
	return nil
}

// IncrementItemsErrorsBy tăng counter items_errors lên N.
func (uc *implUseCase) IncrementItemsErrorsBy(ctx context.Context, projectID string, count int64) error {
	if projectID == "" {
		return state.ErrInvalidProjectID
	}
	if count <= 0 {
		return nil // Skip if count is 0 or negative
	}

	key := state.BuildStateKey(projectID)

	newValue, err := uc.repo.IncrementField(ctx, key, state.FieldItemsErrors, count)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to increment items_errors: project_id=%s, count=%d, error=%v", projectID, count, err)
		return err
	}

	// Also increment legacy crawl_errors for backward compatibility
	if _, err := uc.repo.IncrementField(ctx, key, state.FieldCrawlErrors, count); err != nil {
		uc.l.Warnf(ctx, "Failed to increment legacy crawl_errors: project_id=%s, count=%d, error=%v", projectID, count, err)
	}

	uc.l.Debugf(ctx, "Incremented items_errors: project_id=%s, count=%d, new_value=%d", projectID, count, newValue)
	return nil
}

// ============================================================================
// Legacy Crawl Phase Methods (for backward compatibility)
// ============================================================================

// SetCrawlTotal set tổng số items cần crawl và chuyển status sang PROCESSING.
// Deprecated: Use SetTasksTotal instead.
func (uc *implUseCase) SetCrawlTotal(ctx context.Context, projectID string, total int64) error {
	if projectID == "" {
		return state.ErrInvalidProjectID
	}
	if total < 0 {
		return state.ErrInvalidTotal
	}

	key := state.BuildStateKey(projectID)

	// Update crawl_total and status in one operation
	fields := map[string]any{
		state.FieldCrawlTotal: total,
		state.FieldStatus:     string(models.ProjectStatusProcessing),
	}

	if err := uc.repo.SetFields(ctx, key, fields); err != nil {
		uc.l.Errorf(ctx, "Failed to set crawl total: project_id=%s, total=%d, error=%v",
			projectID, total, err)
		return err
	}

	uc.l.Infof(ctx, "Set crawl total: project_id=%s, crawl_total=%d, status=PROCESSING", projectID, total)
	return nil
}

// IncrementCrawlDoneBy tăng counter crawl_done lên N.
// Deprecated: Use IncrementTasksDone and IncrementItemsActualBy instead.
func (uc *implUseCase) IncrementCrawlDoneBy(ctx context.Context, projectID string, count int64) error {
	if projectID == "" {
		return state.ErrInvalidProjectID
	}
	if count <= 0 {
		return state.ErrInvalidCount
	}

	key := state.BuildStateKey(projectID)

	newValue, err := uc.repo.IncrementField(ctx, key, state.FieldCrawlDone, count)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to increment crawl_done: project_id=%s, count=%d, error=%v", projectID, count, err)
		return err
	}

	uc.l.Debugf(ctx, "Incremented crawl_done: project_id=%s, count=%d, new_value=%d", projectID, count, newValue)
	return nil
}

// IncrementCrawlErrorsBy tăng counter crawl_errors lên N.
// Deprecated: Use IncrementTasksErrors and IncrementItemsErrorsBy instead.
func (uc *implUseCase) IncrementCrawlErrorsBy(ctx context.Context, projectID string, count int64) error {
	if projectID == "" {
		return state.ErrInvalidProjectID
	}
	if count <= 0 {
		return state.ErrInvalidCount
	}

	key := state.BuildStateKey(projectID)

	newValue, err := uc.repo.IncrementField(ctx, key, state.FieldCrawlErrors, count)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to increment crawl_errors: project_id=%s, count=%d, error=%v", projectID, count, err)
		return err
	}

	uc.l.Debugf(ctx, "Incremented crawl_errors: project_id=%s, count=%d, new_value=%d", projectID, count, newValue)
	return nil
}

// ============================================================================
// Analyze Phase Methods
// ============================================================================

// IncrementAnalyzeTotalBy tăng counter analyze_total lên N.
func (uc *implUseCase) IncrementAnalyzeTotalBy(ctx context.Context, projectID string, count int64) error {
	if projectID == "" {
		return state.ErrInvalidProjectID
	}
	if count <= 0 {
		return state.ErrInvalidCount
	}

	key := state.BuildStateKey(projectID)

	newValue, err := uc.repo.IncrementField(ctx, key, state.FieldAnalyzeTotal, count)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to increment analyze_total: project_id=%s, count=%d, error=%v", projectID, count, err)
		return err
	}

	uc.l.Debugf(ctx, "Incremented analyze_total: project_id=%s, count=%d, new_value=%d", projectID, count, newValue)
	return nil
}

// IncrementAnalyzeDoneBy tăng counter analyze_done lên N.
func (uc *implUseCase) IncrementAnalyzeDoneBy(ctx context.Context, projectID string, count int64) error {
	if projectID == "" {
		return state.ErrInvalidProjectID
	}
	if count <= 0 {
		return state.ErrInvalidCount
	}

	key := state.BuildStateKey(projectID)

	newValue, err := uc.repo.IncrementField(ctx, key, state.FieldAnalyzeDone, count)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to increment analyze_done: project_id=%s, count=%d, error=%v", projectID, count, err)
		return err
	}

	uc.l.Debugf(ctx, "Incremented analyze_done: project_id=%s, count=%d, new_value=%d", projectID, count, newValue)
	return nil
}

// IncrementAnalyzeErrorsBy tăng counter analyze_errors lên N.
func (uc *implUseCase) IncrementAnalyzeErrorsBy(ctx context.Context, projectID string, count int64) error {
	if projectID == "" {
		return state.ErrInvalidProjectID
	}
	if count <= 0 {
		return state.ErrInvalidCount
	}

	key := state.BuildStateKey(projectID)

	newValue, err := uc.repo.IncrementField(ctx, key, state.FieldAnalyzeErrors, count)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to increment analyze_errors: project_id=%s, count=%d, error=%v", projectID, count, err)
		return err
	}

	uc.l.Debugf(ctx, "Incremented analyze_errors: project_id=%s, count=%d, new_value=%d", projectID, count, newValue)
	return nil
}

// ============================================================================
// Status & State Methods
// ============================================================================

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

// CheckCompletion kiểm tra và update status nếu cả crawl và analyze đều complete.
// Sử dụng task-level để check crawl completion (reliable hơn item-level).
func (uc *implUseCase) CheckCompletion(ctx context.Context, projectID string) (bool, error) {
	if projectID == "" {
		return false, state.ErrInvalidProjectID
	}

	// Get current state
	s, err := uc.GetState(ctx, projectID)
	if err != nil {
		return false, err
	}

	// Check if both phases complete
	if s.IsComplete() {
		// Update status to DONE
		if err := uc.UpdateStatus(ctx, projectID, models.ProjectStatusDone); err != nil {
			uc.l.Errorf(ctx, "Failed to update completion status: project_id=%s, error=%v", projectID, err)
			return false, ErrUpdateCompletionFailed
		}

		uc.l.Infof(ctx, "Project completed: project_id=%s, tasks=[%d/%d/%d], items=[%d/%d/%d], analyze=[%d/%d/%d]",
			projectID,
			s.TasksDone, s.TasksErrors, s.TasksTotal,
			s.ItemsActual, s.ItemsErrors, s.ItemsExpected,
			s.AnalyzeDone, s.AnalyzeErrors, s.AnalyzeTotal)
		return true, nil
	}

	return false, nil
}

// ============================================================================
// User Mapping Methods
// ============================================================================

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
