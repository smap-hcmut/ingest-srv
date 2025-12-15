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
// Crawl Phase Methods
// ============================================================================

// SetCrawlTotal set tổng số items cần crawl và chuyển status sang PROCESSING.
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

		uc.l.Infof(ctx, "Project completed: project_id=%s, crawl=[%d/%d/%d], analyze=[%d/%d/%d]",
			projectID,
			s.CrawlDone, s.CrawlErrors, s.CrawlTotal,
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
