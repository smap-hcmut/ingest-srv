package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"reflect"
	"strings"
	"time"

	"ingest-srv/internal/datasource"
	repo "ingest-srv/internal/datasource/repository"
	"ingest-srv/internal/execution"
	"ingest-srv/internal/model"
	"ingest-srv/pkg/microservice"
)

// validateSourceType checks if the given source_type is valid.
func (uc *implUseCase) validateSourceType(st string) error {
	switch model.SourceType(st) {
	case model.SourceTypeTikTok, model.SourceTypeFacebook, model.SourceTypeYouTube,
		model.SourceTypeFileUpload, model.SourceTypeWebhook:
		return nil
	default:
		return datasource.ErrInvalidSourceType
	}
}

// validateSourceCategory checks if the given source_category is valid.
func (uc *implUseCase) validateSourceCategory(cat string) error {
	switch model.SourceCategory(cat) {
	case model.SourceCategoryCrawl, model.SourceCategoryPassive:
		return nil
	default:
		return datasource.ErrInvalidCategory
	}
}

// validateCrawlMode checks if the given crawl_mode is valid.
func (uc *implUseCase) validateCrawlMode(mode string) error {
	switch model.CrawlMode(mode) {
	case model.CrawlModeSleep, model.CrawlModeNormal, model.CrawlModeCrisis:
		return nil
	default:
		return datasource.ErrInvalidCrawlMode
	}
}

// validateTriggerType checks if the given trigger_type is valid.
func (uc *implUseCase) validateTriggerType(triggerType string) error {
	switch model.TriggerType(triggerType) {
	case model.TriggerTypeManual, model.TriggerTypeScheduled, model.TriggerTypeProjectEvent,
		model.TriggerTypeCrisisEvent, model.TriggerTypeWebhookPush:
		return nil
	default:
		return datasource.ErrCrawlModeNotAllowed
	}
}

// inferCategory derives source_category from source_type.
// CRAWL: TIKTOK, FACEBOOK, YOUTUBE
// PASSIVE: FILE_UPLOAD, WEBHOOK
func (uc *implUseCase) inferCategory(sourceType string) string {
	switch model.SourceType(sourceType) {
	case model.SourceTypeFileUpload, model.SourceTypeWebhook:
		return string(model.SourceCategoryPassive)
	default:
		return string(model.SourceCategoryCrawl)
	}
}

// validateTargetType checks if the given target_type is valid.
func (uc *implUseCase) validateTargetType(tt string) error {
	switch model.TargetType(tt) {
	case model.TargetTypeKeyword, model.TargetTypeProfile, model.TargetTypePostURL:
		return nil
	default:
		return datasource.ErrInvalidTargetType
	}
}

func (uc *implUseCase) normalizeTargetValues(values []string, dedupe bool) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if dedupe {
			if _, ok := seen[trimmed]; ok {
				continue
			}
			seen[trimmed] = struct{}{}
		}
		out = append(out, trimmed)
	}
	return out
}

func (uc *implUseCase) validateTargetURLValues(values []string) error {
	for _, value := range values {
		parsed, err := url.ParseRequestURI(value)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return datasource.ErrTargetValuesMustBeURLs
		}
	}
	return nil
}

func (uc *implUseCase) prepareTargetValues(targetType model.TargetType, values []string) ([]string, error) {
	normalized := uc.normalizeTargetValues(values, targetType == model.TargetTypeKeyword)
	if len(normalized) == 0 {
		return nil, datasource.ErrTargetValuesRequired
	}
	switch targetType {
	case model.TargetTypeProfile, model.TargetTypePostURL:
		if err := uc.validateTargetURLValues(normalized); err != nil {
			return nil, err
		}
	}
	return normalized, nil
}

func (uc *implUseCase) areStringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if strings.TrimSpace(left[index]) != strings.TrimSpace(right[index]) {
			return false
		}
	}
	return true
}

func (uc *implUseCase) areJSONRawEqual(left, right json.RawMessage) bool {
	if len(left) == 0 && len(right) == 0 {
		return true
	}

	var leftValue interface{}
	leftErr := json.Unmarshal(left, &leftValue)
	var rightValue interface{}
	rightErr := json.Unmarshal(right, &rightValue)
	if leftErr == nil && rightErr == nil {
		return reflect.DeepEqual(leftValue, rightValue)
	}

	return bytes.Equal(bytes.TrimSpace(left), bytes.TrimSpace(right))
}

func (uc *implUseCase) hasMaterialTargetChange(current model.CrawlTarget, normalizedValues []string, input datasource.UpdateTargetInput) bool {
	if input.Values != nil && !uc.areStringSlicesEqual(current.Values, normalizedValues) {
		return true
	}
	if input.CrawlIntervalMinutes != nil && current.CrawlIntervalMinutes != *input.CrawlIntervalMinutes {
		return true
	}
	if input.PlatformMeta != nil && !uc.areJSONRawEqual(current.PlatformMeta, input.PlatformMeta) {
		return true
	}
	return false
}

func (uc *implUseCase) markDatasourcePendingAfterMaterialTargetChange(ctx context.Context, dataSourceID string) error {
	source, err := uc.repo.DetailDataSource(ctx, strings.TrimSpace(dataSourceID))
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.markDatasourcePendingAfterMaterialTargetChange.repo.DetailDataSource: id=%s err=%v", dataSourceID, err)
		return datasource.ErrTargetUpdateFailed
	}
	if source.ID == "" {
		return datasource.ErrNotFound
	}

	if _, err := uc.repo.UpdateDataSource(ctx, repo.UpdateDataSourceOptions{
		ID:           source.ID,
		Status:       string(model.SourceStatusPending),
		DryrunStatus: string(model.DryrunStatusPending),
	}); err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.markDatasourcePendingAfterMaterialTargetChange.repo.UpdateDataSource: id=%s err=%v", source.ID, err)
		return datasource.ErrTargetUpdateFailed
	}

	return nil
}

func (uc *implUseCase) validCreateTargetGroupInput(input datasource.CreateTargetGroupInput) error {
	if strings.TrimSpace(input.DataSourceID) == "" {
		return datasource.ErrProjectIDRequired
	}
	if input.CrawlIntervalMinutes <= 0 {
		return datasource.ErrInvalidTargetInterval
	}
	return nil
}

func (uc *implUseCase) validDetailTargetInput(input datasource.DetailTargetInput) error {
	if strings.TrimSpace(input.DataSourceID) == "" || strings.TrimSpace(input.ID) == "" {
		return datasource.ErrTargetNotFound
	}
	return nil
}

func (uc *implUseCase) validListTargetsInput(input datasource.ListTargetsInput) error {
	if strings.TrimSpace(input.DataSourceID) == "" {
		return datasource.ErrProjectIDRequired
	}
	if strings.TrimSpace(input.TargetType) != "" {
		if err := uc.validateTargetType(strings.TrimSpace(input.TargetType)); err != nil {
			return err
		}
	}
	return nil
}

func (uc *implUseCase) validUpdateTargetInput(input datasource.UpdateTargetInput) error {
	if strings.TrimSpace(input.DataSourceID) == "" || strings.TrimSpace(input.ID) == "" {
		return datasource.ErrTargetNotFound
	}
	if input.CrawlIntervalMinutes != nil && *input.CrawlIntervalMinutes <= 0 {
		return datasource.ErrInvalidTargetInterval
	}
	return nil
}

func (uc *implUseCase) validActivateTargetInput(input datasource.ActivateTargetInput) error {
	if strings.TrimSpace(input.DataSourceID) == "" || strings.TrimSpace(input.ID) == "" {
		return datasource.ErrTargetNotFound
	}
	return nil
}

func (uc *implUseCase) validDeactivateTargetInput(input datasource.DeactivateTargetInput) error {
	if strings.TrimSpace(input.DataSourceID) == "" || strings.TrimSpace(input.ID) == "" {
		return datasource.ErrTargetNotFound
	}
	return nil
}

func (uc *implUseCase) validDeleteTargetInput(input datasource.DeleteTargetInput) error {
	if strings.TrimSpace(input.DataSourceID) == "" || strings.TrimSpace(input.ID) == "" {
		return datasource.ErrTargetNotFound
	}
	return nil
}

func (uc *implUseCase) ensureDataSourceNotArchived(source model.DataSource) error {
	if source.Status == model.SourceStatusArchived {
		return datasource.ErrSourceArchived
	}
	return nil
}

func (uc *implUseCase) isDatasourceDryrunRunning(source model.DataSource) bool {
	return source.DryrunStatus == model.DryrunStatusRunning
}

func (uc *implUseCase) ensureDatasourceDryrunNotRunning(source model.DataSource) error {
	if uc.isDatasourceDryrunRunning(source) {
		return datasource.ErrSourceDryrunRunning
	}
	return nil
}

func (uc *implUseCase) ensureDatasourceTargetMutationNotRunning(source model.DataSource) error {
	if uc.isDatasourceDryrunRunning(source) {
		return datasource.ErrTargetDryrunRunning
	}
	return nil
}

func (uc *implUseCase) getLatestDryrunStatusByTarget(ctx context.Context, targetID string) (model.DryrunStatus, error) {
	latest, err := uc.repo.GetLatestDryrunByTarget(ctx, strings.TrimSpace(targetID))
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.getLatestDryrunStatusByTarget.repo.GetLatestDryrunByTarget: target_id=%s err=%v", targetID, err)
		return "", datasource.ErrUpdateFailed
	}
	if latest.ID == "" {
		return "", nil
	}
	return latest.Status, nil
}

func (uc *implUseCase) ensureTargetDryrunNotRunning(ctx context.Context, targetID string) error {
	status, err := uc.getLatestDryrunStatusByTarget(ctx, targetID)
	if err != nil {
		return err
	}
	if status == model.DryrunStatusRunning {
		return datasource.ErrTargetDryrunRunning
	}
	return nil
}

func (uc *implUseCase) ensureDatasourceTargetsDryrunNotRunning(ctx context.Context, dataSourceID string) error {
	targets, err := uc.repo.ListTargets(ctx, repo.ListTargetsOptions{DataSourceID: strings.TrimSpace(dataSourceID)})
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.ensureDatasourceTargetsDryrunNotRunning.repo.ListTargets: source_id=%s err=%v", dataSourceID, err)
		return datasource.ErrUpdateFailed
	}
	for _, target := range targets {
		status, statusErr := uc.getLatestDryrunStatusByTarget(ctx, target.ID)
		if statusErr != nil {
			return statusErr
		}
		if status == model.DryrunStatusRunning {
			return datasource.ErrSourceDryrunRunning
		}
	}
	return nil
}

func (uc *implUseCase) ensureProjectAvailableForDatasourceCreate(ctx context.Context, projectID string) error {
	if uc.project == nil {
		uc.l.Errorf(ctx, "datasource.usecase.ensureProjectAvailableForDatasourceCreate: project client is nil")
		return datasource.ErrCreateFailed
	}

	projectDetail, err := uc.project.Detail(ctx, strings.TrimSpace(projectID))
	if err != nil {
		switch {
		case errors.Is(err, microservice.ErrBadRequest):
			return datasource.ErrProjectNotFound
		case errors.Is(err, microservice.ErrUnauthorized), errors.Is(err, microservice.ErrForbidden):
			uc.l.Errorf(ctx, "datasource.usecase.ensureProjectAvailableForDatasourceCreate.project.Detail: project_id=%s auth_err=%v", projectID, err)
			return datasource.ErrCreateFailed
		default:
			uc.l.Errorf(ctx, "datasource.usecase.ensureProjectAvailableForDatasourceCreate.project.Detail: project_id=%s err=%v", projectID, err)
			return datasource.ErrCreateFailed
		}
	}

	if projectDetail.Status == microservice.ProjectStatusArchived {
		return datasource.ErrProjectArchived
	}

	return nil
}

func (uc *implUseCase) normalizeActivationReadinessCommand(command datasource.ActivationReadinessCommand) datasource.ActivationReadinessCommand {
	switch datasource.ActivationReadinessCommand(strings.TrimSpace(string(command))) {
	case "", datasource.ActivationReadinessCommandActivate:
		return datasource.ActivationReadinessCommandActivate
	case datasource.ActivationReadinessCommandResume:
		return datasource.ActivationReadinessCommandResume
	default:
		return datasource.ActivationReadinessCommand(strings.TrimSpace(string(command)))
	}
}

func (uc *implUseCase) validActivationReadinessInput(input datasource.ActivationReadinessInput) error {
	if strings.TrimSpace(input.ProjectID) == "" {
		return datasource.ErrProjectIDRequired
	}

	switch uc.normalizeActivationReadinessCommand(input.Command) {
	case datasource.ActivationReadinessCommandActivate, datasource.ActivationReadinessCommandResume:
		return nil
	default:
		return datasource.ErrInvalidReadinessCommand
	}
}

func (uc *implUseCase) isStatusAllowedForCommand(status model.SourceStatus, command datasource.ActivationReadinessCommand) bool {
	switch command {
	case datasource.ActivationReadinessCommandResume:
		return status == model.SourceStatusPaused || status == model.SourceStatusActive
	default:
		return status == model.SourceStatusReady || status == model.SourceStatusActive
	}
}

func (uc *implUseCase) listProjectLifecycleSources(ctx context.Context, projectID string, action string) ([]model.DataSource, error) {
	sources, err := uc.repo.ListDataSources(ctx, repo.ListDataSourcesOptions{ProjectID: projectID})
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.%s.repo.ListDataSources: project_id=%s err=%v", action, projectID, err)
		return nil, datasource.ErrListFailed
	}

	return sources, nil
}

func (uc *implUseCase) ensureProjectSourcesEligible(ctx context.Context, projectID string, sources []model.DataSource, command datasource.ActivationReadinessCommand, notAllowedErr error, action string) error {
	for _, source := range sources {
		if uc.isStatusAllowedForCommand(source.Status, command) {
			continue
		}

		uc.l.Warnf(ctx, "datasource.usecase.%s: project_id=%s source_id=%s status=%s not eligible", action, projectID, source.ID, source.Status)
		return notAllowedErr
	}

	return nil
}

func (uc *implUseCase) buildProjectLifecycleUpdateOptions(projectID string, action string) repo.ProjectLifecycleUpdateOptions {
	switch action {
	case "activate":
		return repo.ProjectLifecycleUpdateOptions{
			ProjectID:      projectID,
			FromStatuses:   []model.SourceStatus{model.SourceStatusReady},
			ToStatus:       model.SourceStatusActive,
			SetActivatedAt: true,
			ClearPausedAt:  true,
		}
	case "pause":
		return repo.ProjectLifecycleUpdateOptions{
			ProjectID:    projectID,
			FromStatuses: []model.SourceStatus{model.SourceStatusActive},
			ToStatus:     model.SourceStatusPaused,
			SetPausedAt:  true,
		}
	case "resume":
		return repo.ProjectLifecycleUpdateOptions{
			ProjectID:      projectID,
			FromStatuses:   []model.SourceStatus{model.SourceStatusPaused},
			ToStatus:       model.SourceStatusActive,
			SetActivatedAt: true,
			ClearPausedAt:  true,
		}
	default:
		return repo.ProjectLifecycleUpdateOptions{ProjectID: projectID}
	}
}

func (uc *implUseCase) ensureCanRemoveActiveTarget(ctx context.Context, dataSourceID string, isActive bool, notAllowedErr error) error {
	if !isActive {
		return nil
	}

	source, err := uc.repo.DetailDataSource(ctx, strings.TrimSpace(dataSourceID))
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.ensureCanRemoveActiveTarget.repo.DetailDataSource: id=%s err=%v", dataSourceID, err)
		return datasource.ErrTargetUpdateFailed
	}
	if source.ID == "" {
		return datasource.ErrNotFound
	}
	if source.Status != model.SourceStatusActive {
		return nil
	}

	activeTargets, err := uc.repo.CountActiveTargets(ctx, source.ID)
	if err != nil {
		uc.l.Errorf(ctx, "datasource.usecase.ensureCanRemoveActiveTarget.repo.CountActiveTargets: source_id=%s err=%v", source.ID, err)
		return datasource.ErrTargetUpdateFailed
	}
	if activeTargets <= 1 {
		return notAllowedErr
	}

	return nil
}

func (uc *implUseCase) validDataSourceID(id string) error {
	if strings.TrimSpace(id) == "" {
		return datasource.ErrNotFound
	}
	return nil
}

func (uc *implUseCase) validUpdateCrawlModeInput(input datasource.UpdateCrawlModeInput) error {
	if strings.TrimSpace(input.ID) == "" {
		return datasource.ErrNotFound
	}
	if err := uc.validateCrawlMode(strings.TrimSpace(input.CrawlMode)); err != nil {
		return err
	}
	if err := uc.validateTriggerType(strings.TrimSpace(input.TriggerType)); err != nil {
		return err
	}
	return nil
}

func (uc *implUseCase) ensureRuntimePrerequisites(ctx context.Context, source model.DataSource, notAllowedErr error) error {
	switch source.SourceCategory {
	case model.SourceCategoryCrawl:
		if source.CrawlMode == nil || source.CrawlIntervalMinutes == nil || *source.CrawlIntervalMinutes <= 0 {
			return notAllowedErr
		}

		targets, err := uc.repo.ListTargets(ctx, repo.ListTargetsOptions{DataSourceID: source.ID})
		if err != nil {
			uc.l.Errorf(ctx, "datasource.usecase.ensureRuntimePrerequisites.repo.ListTargets: source_id=%s err=%v", source.ID, err)
			return datasource.ErrUpdateFailed
		}

		activeTargetCount := 0
		for _, target := range targets {
			if target.IsActive {
				activeTargetCount++
			}

			// Skip dryrun validation for source/target combos that have no dryrun worker.
			if !model.IsDryrunRequired(source.SourceType, target.TargetType) {
				continue
			}

			latest, latestErr := uc.repo.GetLatestDryrunByTarget(ctx, target.ID)
			if latestErr != nil {
				uc.l.Errorf(ctx, "datasource.usecase.ensureRuntimePrerequisites.repo.GetLatestDryrunByTarget: source_id=%s target_id=%s err=%v", source.ID, target.ID, latestErr)
				return datasource.ErrUpdateFailed
			}
			if latest.ID == "" || latest.Status == model.DryrunStatusFailed {
				return notAllowedErr
			}
		}

		if activeTargetCount <= 0 {
			return notAllowedErr
		}
	case model.SourceCategoryPassive:
		switch source.SourceType {
		case model.SourceTypeWebhook:
			if source.WebhookID == "" || source.WebhookSecretEncrypted == "" {
				return notAllowedErr
			}
		default:
			return notAllowedErr
		}
	default:
		return notAllowedErr
	}

	return nil
}

func (uc *implUseCase) cancelProjectRuntime(ctx context.Context, projectID, reason string, canceledAt time.Time) error {
	if uc.exec == nil {
		uc.l.Warnf(ctx, "datasource.usecase.cancelProjectRuntime: execution usecase is nil project_id=%s", projectID)
		return nil
	}

	return uc.exec.CancelProjectRuntime(ctx, execution.CancelProjectRuntimeInput{
		ProjectID:  strings.TrimSpace(projectID),
		Reason:     strings.TrimSpace(reason),
		CanceledAt: canceledAt,
	})
}
