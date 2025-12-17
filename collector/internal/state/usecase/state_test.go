package usecase

import (
	"context"
	"testing"
	"time"

	"smap-collector/internal/models"
	"smap-collector/internal/state"
)

// mockStateRepository is a test mock for StateRepository
type mockStateRepository struct {
	existsFunc         func(ctx context.Context, key string) (bool, error)
	getStateFunc       func(ctx context.Context, key string) (*models.ProjectState, error)
	initStateFunc      func(ctx context.Context, key string, state models.ProjectState, ttl time.Duration) error
	setFieldFunc       func(ctx context.Context, key, field string, value any) error
	setFieldsFunc      func(ctx context.Context, key string, fields map[string]any) error
	incrementFieldFunc func(ctx context.Context, key, field string, delta int64) (int64, error)
	setStringFunc      func(ctx context.Context, key, value string, ttl time.Duration) error
	getStringFunc      func(ctx context.Context, key string) (string, error)
}

func (m *mockStateRepository) Exists(ctx context.Context, key string) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, key)
	}
	return false, nil
}

func (m *mockStateRepository) GetState(ctx context.Context, key string) (*models.ProjectState, error) {
	if m.getStateFunc != nil {
		return m.getStateFunc(ctx, key)
	}
	return nil, nil
}

func (m *mockStateRepository) InitState(ctx context.Context, key string, s models.ProjectState, ttl time.Duration) error {
	if m.initStateFunc != nil {
		return m.initStateFunc(ctx, key, s, ttl)
	}
	return nil
}

func (m *mockStateRepository) SetField(ctx context.Context, key, field string, value any) error {
	if m.setFieldFunc != nil {
		return m.setFieldFunc(ctx, key, field, value)
	}
	return nil
}

func (m *mockStateRepository) SetFields(ctx context.Context, key string, fields map[string]any) error {
	if m.setFieldsFunc != nil {
		return m.setFieldsFunc(ctx, key, fields)
	}
	return nil
}

func (m *mockStateRepository) IncrementField(ctx context.Context, key, field string, delta int64) (int64, error) {
	if m.incrementFieldFunc != nil {
		return m.incrementFieldFunc(ctx, key, field, delta)
	}
	return 0, nil
}

func (m *mockStateRepository) SetTTL(ctx context.Context, key string, ttl time.Duration) error {
	return nil
}

func (m *mockStateRepository) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *mockStateRepository) SetString(ctx context.Context, key, value string, ttl time.Duration) error {
	if m.setStringFunc != nil {
		return m.setStringFunc(ctx, key, value, ttl)
	}
	return nil
}

func (m *mockStateRepository) GetString(ctx context.Context, key string) (string, error) {
	if m.getStringFunc != nil {
		return m.getStringFunc(ctx, key)
	}
	return "", nil
}

// mockLogger is a test mock for Logger
type mockLogger struct{}

func (m *mockLogger) Debug(ctx context.Context, arg ...any)                   {}
func (m *mockLogger) Debugf(ctx context.Context, template string, arg ...any) {}
func (m *mockLogger) Info(ctx context.Context, arg ...any)                    {}
func (m *mockLogger) Infof(ctx context.Context, template string, arg ...any)  {}
func (m *mockLogger) Warn(ctx context.Context, arg ...any)                    {}
func (m *mockLogger) Warnf(ctx context.Context, template string, arg ...any)  {}
func (m *mockLogger) Error(ctx context.Context, arg ...any)                   {}
func (m *mockLogger) Errorf(ctx context.Context, template string, arg ...any) {}
func (m *mockLogger) Fatal(ctx context.Context, arg ...any)                   {}
func (m *mockLogger) Fatalf(ctx context.Context, template string, arg ...any) {}

func TestInitState(t *testing.T) {
	ctx := context.Background()

	t.Run("success - new state", func(t *testing.T) {
		mockRepo := &mockStateRepository{
			existsFunc: func(ctx context.Context, key string) (bool, error) {
				return false, nil
			},
			initStateFunc: func(ctx context.Context, key string, s models.ProjectState, ttl time.Duration) error {
				return nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.InitState(ctx, "proj_1")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("error - empty project ID", func(t *testing.T) {
		mockRepo := &mockStateRepository{}
		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.InitState(ctx, "")
		if err != state.ErrInvalidProjectID {
			t.Errorf("expected ErrInvalidProjectID, got %v", err)
		}
	})

	t.Run("error - state already exists and active", func(t *testing.T) {
		activeState := &models.ProjectState{Status: models.ProjectStatusProcessing}
		mockRepo := &mockStateRepository{
			existsFunc: func(ctx context.Context, key string) (bool, error) {
				return true, nil
			},
			getStateFunc: func(ctx context.Context, key string) (*models.ProjectState, error) {
				return activeState, nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.InitState(ctx, "proj_1")
		if err != state.ErrStateAlreadyExists {
			t.Errorf("expected ErrStateAlreadyExists, got %v", err)
		}
	})
}

func TestSetCrawlTotal(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockStateRepository{
			setFieldsFunc: func(ctx context.Context, key string, fields map[string]any) error {
				return nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.SetCrawlTotal(ctx, "proj_1", 100)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("error - empty project ID", func(t *testing.T) {
		mockRepo := &mockStateRepository{}
		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.SetCrawlTotal(ctx, "", 100)
		if err != state.ErrInvalidProjectID {
			t.Errorf("expected ErrInvalidProjectID, got %v", err)
		}
	})

	t.Run("error - negative total", func(t *testing.T) {
		mockRepo := &mockStateRepository{}
		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.SetCrawlTotal(ctx, "proj_1", -1)
		if err != state.ErrInvalidTotal {
			t.Errorf("expected ErrInvalidTotal, got %v", err)
		}
	})
}

func TestIncrementCrawlDoneBy(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockStateRepository{
			incrementFieldFunc: func(ctx context.Context, key, field string, delta int64) (int64, error) {
				return 50, nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.IncrementCrawlDoneBy(ctx, "proj_1", 10)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("error - empty project ID", func(t *testing.T) {
		mockRepo := &mockStateRepository{}
		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.IncrementCrawlDoneBy(ctx, "", 10)
		if err != state.ErrInvalidProjectID {
			t.Errorf("expected ErrInvalidProjectID, got %v", err)
		}
	})

	t.Run("error - invalid count", func(t *testing.T) {
		mockRepo := &mockStateRepository{}
		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.IncrementCrawlDoneBy(ctx, "proj_1", 0)
		if err != state.ErrInvalidCount {
			t.Errorf("expected ErrInvalidCount, got %v", err)
		}
	})
}

func TestIncrementCrawlErrorsBy(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockStateRepository{
			incrementFieldFunc: func(ctx context.Context, key, field string, delta int64) (int64, error) {
				return 5, nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.IncrementCrawlErrorsBy(ctx, "proj_1", 2)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestIncrementAnalyzeTotalBy(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockStateRepository{
			incrementFieldFunc: func(ctx context.Context, key, field string, delta int64) (int64, error) {
				return 100, nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.IncrementAnalyzeTotalBy(ctx, "proj_1", 50)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestIncrementAnalyzeDoneBy(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockStateRepository{
			incrementFieldFunc: func(ctx context.Context, key, field string, delta int64) (int64, error) {
				return 30, nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.IncrementAnalyzeDoneBy(ctx, "proj_1", 10)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestIncrementAnalyzeErrorsBy(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockStateRepository{
			incrementFieldFunc: func(ctx context.Context, key, field string, delta int64) (int64, error) {
				return 3, nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.IncrementAnalyzeErrorsBy(ctx, "proj_1", 1)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestGetState(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		expectedState := &models.ProjectState{
			Status:        models.ProjectStatusProcessing,
			CrawlTotal:    100,
			CrawlDone:     50,
			CrawlErrors:   2,
			AnalyzeTotal:  48,
			AnalyzeDone:   20,
			AnalyzeErrors: 1,
		}
		mockRepo := &mockStateRepository{
			getStateFunc: func(ctx context.Context, key string) (*models.ProjectState, error) {
				return expectedState, nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		result, err := uc.GetState(ctx, "proj_1")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if result.CrawlTotal != 100 || result.CrawlDone != 50 {
			t.Errorf("unexpected state: %+v", result)
		}
	})

	t.Run("error - not found", func(t *testing.T) {
		mockRepo := &mockStateRepository{
			getStateFunc: func(ctx context.Context, key string) (*models.ProjectState, error) {
				return nil, nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		_, err := uc.GetState(ctx, "proj_1")
		if err != state.ErrStateNotFound {
			t.Errorf("expected ErrStateNotFound, got %v", err)
		}
	})
}

func TestCheckCompletion(t *testing.T) {
	ctx := context.Background()

	t.Run("complete - both phases done", func(t *testing.T) {
		completeState := &models.ProjectState{
			Status:        models.ProjectStatusProcessing,
			CrawlTotal:    10,
			CrawlDone:     10,
			CrawlErrors:   0,
			AnalyzeTotal:  10,
			AnalyzeDone:   10,
			AnalyzeErrors: 0,
		}
		mockRepo := &mockStateRepository{
			getStateFunc: func(ctx context.Context, key string) (*models.ProjectState, error) {
				return completeState, nil
			},
			setFieldFunc: func(ctx context.Context, key, field string, value any) error {
				return nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		completed, err := uc.CheckCompletion(ctx, "proj_1")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if !completed {
			t.Error("expected completed to be true")
		}
	})

	t.Run("not complete - crawl done but analyze not", func(t *testing.T) {
		incompleteState := &models.ProjectState{
			Status:        models.ProjectStatusProcessing,
			CrawlTotal:    10,
			CrawlDone:     10,
			CrawlErrors:   0,
			AnalyzeTotal:  10,
			AnalyzeDone:   5,
			AnalyzeErrors: 0,
		}
		mockRepo := &mockStateRepository{
			getStateFunc: func(ctx context.Context, key string) (*models.ProjectState, error) {
				return incompleteState, nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		completed, err := uc.CheckCompletion(ctx, "proj_1")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if completed {
			t.Error("expected completed to be false")
		}
	})

	t.Run("not complete - neither phase done", func(t *testing.T) {
		incompleteState := &models.ProjectState{
			Status:        models.ProjectStatusProcessing,
			CrawlTotal:    10,
			CrawlDone:     5,
			CrawlErrors:   0,
			AnalyzeTotal:  5,
			AnalyzeDone:   2,
			AnalyzeErrors: 0,
		}
		mockRepo := &mockStateRepository{
			getStateFunc: func(ctx context.Context, key string) (*models.ProjectState, error) {
				return incompleteState, nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		completed, err := uc.CheckCompletion(ctx, "proj_1")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if completed {
			t.Error("expected completed to be false")
		}
	})
}

func TestStoreUserMapping(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockStateRepository{
			setStringFunc: func(ctx context.Context, key, value string, ttl time.Duration) error {
				return nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.StoreUserMapping(ctx, "proj_1", "user_1")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestGetUserID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := &mockStateRepository{
			getStringFunc: func(ctx context.Context, key string) (string, error) {
				return "user_1", nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		userID, err := uc.GetUserID(ctx, "proj_1")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if userID != "user_1" {
			t.Errorf("expected user_1, got %s", userID)
		}
	})

	t.Run("error - not found", func(t *testing.T) {
		mockRepo := &mockStateRepository{
			getStringFunc: func(ctx context.Context, key string) (string, error) {
				return "", nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		_, err := uc.GetUserID(ctx, "proj_1")
		if err != state.ErrUserMappingNotFound {
			t.Errorf("expected ErrUserMappingNotFound, got %v", err)
		}
	})
}

// ============================================================================
// Tests for Task-Level Methods (Phase 8.2.1)
// ============================================================================

func TestSetTasksTotal(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		var capturedFields map[string]any
		mockRepo := &mockStateRepository{
			setFieldsFunc: func(ctx context.Context, key string, fields map[string]any) error {
				capturedFields = fields
				return nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.SetTasksTotal(ctx, "proj_1", 10, 500)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Verify fields were set correctly
		if capturedFields[state.FieldTasksTotal] != int64(10) {
			t.Errorf("expected tasks_total=10, got %v", capturedFields[state.FieldTasksTotal])
		}
		if capturedFields[state.FieldItemsExpected] != int64(500) {
			t.Errorf("expected items_expected=500, got %v", capturedFields[state.FieldItemsExpected])
		}
		if capturedFields[state.FieldStatus] != string(models.ProjectStatusProcessing) {
			t.Errorf("expected status=PROCESSING, got %v", capturedFields[state.FieldStatus])
		}
	})

	t.Run("error - empty project ID", func(t *testing.T) {
		mockRepo := &mockStateRepository{}
		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.SetTasksTotal(ctx, "", 10, 500)
		if err != state.ErrInvalidProjectID {
			t.Errorf("expected ErrInvalidProjectID, got %v", err)
		}
	})

	t.Run("error - negative total", func(t *testing.T) {
		mockRepo := &mockStateRepository{}
		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.SetTasksTotal(ctx, "proj_1", -1, 500)
		if err != state.ErrInvalidTotal {
			t.Errorf("expected ErrInvalidTotal, got %v", err)
		}
	})
}

func TestIncrementTasksDone(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		var capturedField string
		var capturedDelta int64
		mockRepo := &mockStateRepository{
			incrementFieldFunc: func(ctx context.Context, key, field string, delta int64) (int64, error) {
				capturedField = field
				capturedDelta = delta
				return 5, nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.IncrementTasksDone(ctx, "proj_1")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if capturedField != state.FieldTasksDone {
			t.Errorf("expected field=%s, got %s", state.FieldTasksDone, capturedField)
		}
		if capturedDelta != 1 {
			t.Errorf("expected delta=1, got %d", capturedDelta)
		}
	})

	t.Run("error - empty project ID", func(t *testing.T) {
		mockRepo := &mockStateRepository{}
		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.IncrementTasksDone(ctx, "")
		if err != state.ErrInvalidProjectID {
			t.Errorf("expected ErrInvalidProjectID, got %v", err)
		}
	})
}

func TestIncrementTasksErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		var capturedField string
		var capturedDelta int64
		mockRepo := &mockStateRepository{
			incrementFieldFunc: func(ctx context.Context, key, field string, delta int64) (int64, error) {
				capturedField = field
				capturedDelta = delta
				return 2, nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.IncrementTasksErrors(ctx, "proj_1")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if capturedField != state.FieldTasksErrors {
			t.Errorf("expected field=%s, got %s", state.FieldTasksErrors, capturedField)
		}
		if capturedDelta != 1 {
			t.Errorf("expected delta=1, got %d", capturedDelta)
		}
	})

	t.Run("error - empty project ID", func(t *testing.T) {
		mockRepo := &mockStateRepository{}
		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.IncrementTasksErrors(ctx, "")
		if err != state.ErrInvalidProjectID {
			t.Errorf("expected ErrInvalidProjectID, got %v", err)
		}
	})
}

func TestIncrementItemsActualBy(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		incrementCalls := make(map[string]int64)
		mockRepo := &mockStateRepository{
			incrementFieldFunc: func(ctx context.Context, key, field string, delta int64) (int64, error) {
				incrementCalls[field] = delta
				return 50, nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.IncrementItemsActualBy(ctx, "proj_1", 10)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if incrementCalls[state.FieldItemsActual] != 10 {
			t.Errorf("expected items_actual delta=10, got %d", incrementCalls[state.FieldItemsActual])
		}
		// Also check legacy field is updated
		if incrementCalls[state.FieldCrawlDone] != 10 {
			t.Errorf("expected crawl_done delta=10, got %d", incrementCalls[state.FieldCrawlDone])
		}
	})

	t.Run("skip - zero count", func(t *testing.T) {
		mockRepo := &mockStateRepository{
			incrementFieldFunc: func(ctx context.Context, key, field string, delta int64) (int64, error) {
				t.Error("should not call incrementField for zero count")
				return 0, nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.IncrementItemsActualBy(ctx, "proj_1", 0)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("skip - negative count", func(t *testing.T) {
		mockRepo := &mockStateRepository{
			incrementFieldFunc: func(ctx context.Context, key, field string, delta int64) (int64, error) {
				t.Error("should not call incrementField for negative count")
				return 0, nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.IncrementItemsActualBy(ctx, "proj_1", -5)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("error - empty project ID", func(t *testing.T) {
		mockRepo := &mockStateRepository{}
		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.IncrementItemsActualBy(ctx, "", 10)
		if err != state.ErrInvalidProjectID {
			t.Errorf("expected ErrInvalidProjectID, got %v", err)
		}
	})
}

func TestIncrementItemsErrorsBy(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		incrementCalls := make(map[string]int64)
		mockRepo := &mockStateRepository{
			incrementFieldFunc: func(ctx context.Context, key, field string, delta int64) (int64, error) {
				incrementCalls[field] = delta
				return 5, nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.IncrementItemsErrorsBy(ctx, "proj_1", 3)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if incrementCalls[state.FieldItemsErrors] != 3 {
			t.Errorf("expected items_errors delta=3, got %d", incrementCalls[state.FieldItemsErrors])
		}
		// Also check legacy field is updated
		if incrementCalls[state.FieldCrawlErrors] != 3 {
			t.Errorf("expected crawl_errors delta=3, got %d", incrementCalls[state.FieldCrawlErrors])
		}
	})

	t.Run("skip - zero count", func(t *testing.T) {
		mockRepo := &mockStateRepository{
			incrementFieldFunc: func(ctx context.Context, key, field string, delta int64) (int64, error) {
				t.Error("should not call incrementField for zero count")
				return 0, nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.IncrementItemsErrorsBy(ctx, "proj_1", 0)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("error - empty project ID", func(t *testing.T) {
		mockRepo := &mockStateRepository{}
		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		err := uc.IncrementItemsErrorsBy(ctx, "", 3)
		if err != state.ErrInvalidProjectID {
			t.Errorf("expected ErrInvalidProjectID, got %v", err)
		}
	})
}

func TestCheckCompletion_HybridState(t *testing.T) {
	ctx := context.Background()

	t.Run("complete - task-level based", func(t *testing.T) {
		completeState := &models.ProjectState{
			Status:        models.ProjectStatusProcessing,
			TasksTotal:    10,
			TasksDone:     10,
			TasksErrors:   0,
			ItemsExpected: 500,
			ItemsActual:   450,
			ItemsErrors:   50,
			AnalyzeTotal:  450,
			AnalyzeDone:   450,
			AnalyzeErrors: 0,
		}
		mockRepo := &mockStateRepository{
			getStateFunc: func(ctx context.Context, key string) (*models.ProjectState, error) {
				return completeState, nil
			},
			setFieldFunc: func(ctx context.Context, key, field string, value any) error {
				return nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		completed, err := uc.CheckCompletion(ctx, "proj_1")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if !completed {
			t.Error("expected completed to be true")
		}
	})

	t.Run("not complete - tasks not done", func(t *testing.T) {
		incompleteState := &models.ProjectState{
			Status:        models.ProjectStatusProcessing,
			TasksTotal:    10,
			TasksDone:     5,
			TasksErrors:   0,
			ItemsExpected: 500,
			ItemsActual:   250,
			ItemsErrors:   0,
			AnalyzeTotal:  250,
			AnalyzeDone:   250,
			AnalyzeErrors: 0,
		}
		mockRepo := &mockStateRepository{
			getStateFunc: func(ctx context.Context, key string) (*models.ProjectState, error) {
				return incompleteState, nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		completed, err := uc.CheckCompletion(ctx, "proj_1")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if completed {
			t.Error("expected completed to be false")
		}
	})

	t.Run("not complete - analyze not done", func(t *testing.T) {
		incompleteState := &models.ProjectState{
			Status:        models.ProjectStatusProcessing,
			TasksTotal:    10,
			TasksDone:     10,
			TasksErrors:   0,
			ItemsExpected: 500,
			ItemsActual:   500,
			ItemsErrors:   0,
			AnalyzeTotal:  500,
			AnalyzeDone:   250,
			AnalyzeErrors: 0,
		}
		mockRepo := &mockStateRepository{
			getStateFunc: func(ctx context.Context, key string) (*models.ProjectState, error) {
				return incompleteState, nil
			},
		}

		uc := NewUseCase(&mockLogger{}, mockRepo, state.Options{TTL: time.Hour})

		completed, err := uc.CheckCompletion(ctx, "proj_1")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if completed {
			t.Error("expected completed to be false")
		}
	})
}
