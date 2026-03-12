package http

import (
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"ingest-srv/internal/datasource"
	"ingest-srv/internal/model"

	"github.com/smap-hcmut/shared-libs/go/paginator"
)

// --- Request DTOs ---

// createReq represents data source creation request.
type createReq struct {
	ProjectID            string          `json:"project_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name                 string          `json:"name" binding:"required" example:"TikTok VinFast Crawler"`
	Description          string          `json:"description" example:"Crawl TikTok posts about VinFast"`
	SourceType           string          `json:"source_type" binding:"required" example:"TIKTOK" enums:"TIKTOK,FACEBOOK,YOUTUBE,FILE_UPLOAD,WEBHOOK"`
	SourceCategory       string          `json:"source_category" example:"CRAWL" enums:"CRAWL,PASSIVE"`
	Config               json.RawMessage `json:"config,omitempty" swaggertype:"object"`
	AccountRef           json.RawMessage `json:"account_ref,omitempty" swaggertype:"object"`
	MappingRules         json.RawMessage `json:"mapping_rules,omitempty" swaggertype:"object"`
	CrawlMode            string          `json:"crawl_mode" example:"NORMAL" enums:"SLEEP,NORMAL,CRISIS"`
	CrawlIntervalMinutes int             `json:"crawl_interval_minutes" example:"11"`
}

func (r createReq) validate() error {
	if strings.TrimSpace(r.ProjectID) == "" {
		return errProjectIDRequired
	}
	if strings.TrimSpace(r.Name) == "" {
		return errNameRequired
	}
	if strings.TrimSpace(r.SourceType) == "" {
		return errSourceTypeRequired
	}

	sourceType := strings.TrimSpace(r.SourceType)
	if !isValidSourceType(sourceType) {
		return errInvalidSourceType
	}

	category := strings.TrimSpace(r.SourceCategory)
	if category != "" && !isValidSourceCategory(category) {
		return errInvalidCategory
	}
	if category == "" {
		category = inferSourceCategory(sourceType)
	}

	if strings.TrimSpace(r.CrawlMode) != "" && !isValidCrawlMode(strings.TrimSpace(r.CrawlMode)) {
		return errInvalidCrawlMode
	}
	if r.CrawlIntervalMinutes < 0 {
		return errInvalidCrawlInterval
	}
	if category == string(model.SourceCategoryCrawl) {
		if strings.TrimSpace(r.CrawlMode) == "" || r.CrawlIntervalMinutes <= 0 {
			return errCrawlConfigRequired
		}
	}
	return nil
}

func (r createReq) toInput() datasource.CreateInput {
	return datasource.CreateInput{
		ProjectID:            strings.TrimSpace(r.ProjectID),
		Name:                 strings.TrimSpace(r.Name),
		Description:          r.Description,
		SourceType:           strings.TrimSpace(r.SourceType),
		SourceCategory:       strings.TrimSpace(r.SourceCategory),
		Config:               r.Config,
		AccountRef:           r.AccountRef,
		MappingRules:         r.MappingRules,
		CrawlMode:            strings.TrimSpace(r.CrawlMode),
		CrawlIntervalMinutes: r.CrawlIntervalMinutes,
	}
}

// detailReq extracts ID from path param.
type detailReq struct {
	ID string
}

func (r detailReq) validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return errWrongBody
	}
	return nil
}

func (r detailReq) toInput() string {
	return strings.TrimSpace(r.ID)
}

// listReq binds query params for listing data sources.
type listReq struct {
	paginator.PaginateQuery
	ProjectID      string `form:"project_id"`
	Status         string `form:"status"`
	SourceType     string `form:"source_type"`
	SourceCategory string `form:"source_category"`
	CrawlMode      string `form:"crawl_mode"`
	Name           string `form:"name"`
}

func (r listReq) validate() error {
	if strings.TrimSpace(r.SourceType) != "" && !isValidSourceType(strings.TrimSpace(r.SourceType)) {
		return errInvalidSourceType
	}
	if strings.TrimSpace(r.SourceCategory) != "" && !isValidSourceCategory(strings.TrimSpace(r.SourceCategory)) {
		return errInvalidCategory
	}
	if strings.TrimSpace(r.CrawlMode) != "" && !isValidCrawlMode(strings.TrimSpace(r.CrawlMode)) {
		return errInvalidCrawlMode
	}
	return nil
}

func (r listReq) toInput() datasource.ListInput {
	return datasource.ListInput{
		ProjectID:      strings.TrimSpace(r.ProjectID),
		Status:         strings.TrimSpace(r.Status),
		SourceType:     strings.TrimSpace(r.SourceType),
		SourceCategory: strings.TrimSpace(r.SourceCategory),
		CrawlMode:      strings.TrimSpace(r.CrawlMode),
		Name:           strings.TrimSpace(r.Name),
		Paginator:      r.PaginateQuery,
	}
}

// updateReq represents data source update request.
type updateReq struct {
	ID           string          `json:"-"`
	Name         string          `json:"name" example:"TikTok VinFast Crawler v2"`
	Description  string          `json:"description" example:"Updated description"`
	Config       json.RawMessage `json:"config,omitempty" swaggertype:"object"`
	AccountRef   json.RawMessage `json:"account_ref,omitempty" swaggertype:"object"`
	MappingRules json.RawMessage `json:"mapping_rules,omitempty" swaggertype:"object"`
}

func (r updateReq) toInput() datasource.UpdateInput {
	return datasource.UpdateInput{
		ID:           strings.TrimSpace(r.ID),
		Name:         strings.TrimSpace(r.Name),
		Description:  r.Description,
		Config:       r.Config,
		AccountRef:   r.AccountRef,
		MappingRules: r.MappingRules,
	}
}

// archiveReq extracts ID from path param.
type archiveReq struct {
	ID string
}

func (r archiveReq) validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return errWrongBody
	}
	return nil
}

func (r archiveReq) toInput() string {
	return strings.TrimSpace(r.ID)
}

// --- Response DTOs ---

// dataSourceResp represents data source data in API responses.
type dataSourceResp struct {
	ID                   string          `json:"id" example:"550e8400-e29b-41d4-a716-446655440001"`
	ProjectID            string          `json:"project_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name                 string          `json:"name" example:"TikTok VinFast Crawler"`
	Description          string          `json:"description,omitempty"`
	SourceType           string          `json:"source_type" example:"TIKTOK"`
	SourceCategory       string          `json:"source_category" example:"CRAWL"`
	Status               string          `json:"status" example:"PENDING"`
	Config               json.RawMessage `json:"config,omitempty" swaggertype:"object"`
	AccountRef           json.RawMessage `json:"account_ref,omitempty" swaggertype:"object"`
	MappingRules         json.RawMessage `json:"mapping_rules,omitempty" swaggertype:"object"`
	OnboardingStatus     string          `json:"onboarding_status" example:"NOT_REQUIRED"`
	DryrunStatus         string          `json:"dryrun_status" example:"NOT_REQUIRED"`
	DryrunLastResultID   string          `json:"dryrun_last_result_id,omitempty"`
	CrawlMode            *string         `json:"crawl_mode,omitempty" example:"NORMAL"`
	CrawlIntervalMinutes *int            `json:"crawl_interval_minutes,omitempty" example:"11"`
	NextCrawlAt          *string         `json:"next_crawl_at,omitempty"`
	LastCrawlAt          *string         `json:"last_crawl_at,omitempty"`
	LastSuccessAt        *string         `json:"last_success_at,omitempty"`
	LastErrorAt          *string         `json:"last_error_at,omitempty"`
	LastErrorMessage     string          `json:"last_error_message,omitempty"`
	WebhookID            string          `json:"webhook_id,omitempty"`
	CreatedBy            string          `json:"created_by,omitempty"`
	ActivatedAt          *string         `json:"activated_at,omitempty"`
	PausedAt             *string         `json:"paused_at,omitempty"`
	ArchivedAt           *string         `json:"archived_at,omitempty"`
	CreatedAt            string          `json:"created_at" example:"2026-03-03T00:00:00Z"`
	UpdatedAt            string          `json:"updated_at" example:"2026-03-03T00:00:00Z"`
}

// createResp wraps data source creation response.
type createResp struct {
	DataSource dataSourceResp `json:"data_source"`
}

// detailResp wraps data source detail response.
type detailResp struct {
	DataSource dataSourceResp `json:"data_source"`
}

// listResp wraps paginated data source list response.
type listResp struct {
	DataSources []dataSourceResp            `json:"data_sources"`
	Paginator   paginator.PaginatorResponse `json:"paginator"`
}

// updateResp wraps data source update response.
type updateResp struct {
	DataSource dataSourceResp `json:"data_source"`
}

type updateCrawlModeReq struct {
	ID          string `json:"-"`
	CrawlMode   string `json:"crawl_mode" binding:"required" example:"NORMAL" enums:"SLEEP,NORMAL,CRISIS"`
	TriggerType string `json:"trigger_type" binding:"required" example:"MANUAL" enums:"MANUAL,SCHEDULED,PROJECT_EVENT,CRISIS_EVENT,WEBHOOK_PUSH"`
	Reason      string `json:"reason,omitempty" example:"manual reset after crisis"`
	EventRef    string `json:"event_ref,omitempty" example:"incident-20260306-001"`
}

func (r updateCrawlModeReq) validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return errWrongBody
	}
	if !isValidCrawlMode(strings.TrimSpace(r.CrawlMode)) {
		return errInvalidCrawlMode
	}
	if !isValidTriggerType(strings.TrimSpace(r.TriggerType)) {
		return errCrawlModeNotAllowed
	}
	return nil
}

func (r updateCrawlModeReq) toInput() datasource.UpdateCrawlModeInput {
	return datasource.UpdateCrawlModeInput{
		ID:          strings.TrimSpace(r.ID),
		CrawlMode:   strings.TrimSpace(r.CrawlMode),
		TriggerType: strings.TrimSpace(r.TriggerType),
		Reason:      strings.TrimSpace(r.Reason),
		EventRef:    strings.TrimSpace(r.EventRef),
	}
}

type updateCrawlModeResp struct {
	DataSource dataSourceResp `json:"data_source"`
}

// --- Response Mappers ---

func (h *handler) newCreateResp(o datasource.CreateOutput) createResp {
	return createResp{DataSource: toDataSourceResp(o.DataSource)}
}

func (h *handler) newDetailResp(o datasource.DetailOutput) detailResp {
	return detailResp{DataSource: toDataSourceResp(o.DataSource)}
}

func (h *handler) newListResp(o datasource.ListOutput) listResp {
	items := make([]dataSourceResp, len(o.DataSources))
	for i, ds := range o.DataSources {
		items[i] = toDataSourceResp(ds)
	}
	return listResp{
		DataSources: items,
		Paginator:   o.Paginator.ToResponse(),
	}
}

func (h *handler) newUpdateResp(o datasource.UpdateOutput) updateResp {
	return updateResp{DataSource: toDataSourceResp(o.DataSource)}
}

func (h *handler) newUpdateCrawlModeResp(o datasource.UpdateCrawlModeOutput) updateCrawlModeResp {
	return updateCrawlModeResp{DataSource: toDataSourceResp(o.DataSource)}
}

// --- Internal Mapper ---

const timeFormat = "2006-01-02T15:04:05Z07:00"

func toDataSourceResp(ds model.DataSource) dataSourceResp {
	resp := dataSourceResp{
		ID:                 ds.ID,
		ProjectID:          ds.ProjectID,
		Name:               ds.Name,
		SourceType:         string(ds.SourceType),
		SourceCategory:     string(ds.SourceCategory),
		Status:             string(ds.Status),
		OnboardingStatus:   string(ds.OnboardingStatus),
		DryrunStatus:       string(ds.DryrunStatus),
		DryrunLastResultID: ds.DryrunLastResultID,
		LastErrorMessage:   ds.LastErrorMessage,
		WebhookID:          ds.WebhookID,
		CreatedBy:          ds.CreatedBy,
		CreatedAt:          ds.CreatedAt.Format(timeFormat),
		UpdatedAt:          ds.UpdatedAt.Format(timeFormat),
	}

	if ds.Description != "" {
		resp.Description = ds.Description
	}
	if len(ds.Config) > 0 {
		resp.Config = ds.Config
	}
	if len(ds.AccountRef) > 0 {
		resp.AccountRef = ds.AccountRef
	}
	if len(ds.MappingRules) > 0 {
		resp.MappingRules = ds.MappingRules
	}
	if ds.CrawlMode != nil {
		mode := string(*ds.CrawlMode)
		resp.CrawlMode = &mode
	}
	if ds.CrawlIntervalMinutes != nil {
		resp.CrawlIntervalMinutes = ds.CrawlIntervalMinutes
	}
	resp.NextCrawlAt = formatTimePtr(ds.NextCrawlAt)
	resp.LastCrawlAt = formatTimePtr(ds.LastCrawlAt)
	resp.LastSuccessAt = formatTimePtr(ds.LastSuccessAt)
	resp.LastErrorAt = formatTimePtr(ds.LastErrorAt)
	resp.ActivatedAt = formatTimePtr(ds.ActivatedAt)
	resp.PausedAt = formatTimePtr(ds.PausedAt)
	resp.ArchivedAt = formatTimePtr(ds.ArchivedAt)

	return resp
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(timeFormat)
	return &s
}

func isValidSourceType(value string) bool {
	switch model.SourceType(value) {
	case model.SourceTypeTikTok, model.SourceTypeFacebook, model.SourceTypeYouTube,
		model.SourceTypeFileUpload, model.SourceTypeWebhook:
		return true
	default:
		return false
	}
}

func isValidSourceCategory(value string) bool {
	switch model.SourceCategory(value) {
	case model.SourceCategoryCrawl, model.SourceCategoryPassive:
		return true
	default:
		return false
	}
}

func inferSourceCategory(sourceType string) string {
	switch model.SourceType(sourceType) {
	case model.SourceTypeFileUpload, model.SourceTypeWebhook:
		return string(model.SourceCategoryPassive)
	default:
		return string(model.SourceCategoryCrawl)
	}
}

func isValidCrawlMode(value string) bool {
	switch model.CrawlMode(value) {
	case model.CrawlModeSleep, model.CrawlModeNormal, model.CrawlModeCrisis:
		return true
	default:
		return false
	}
}

func isValidTargetType(value string) bool {
	switch model.TargetType(value) {
	case model.TargetTypeKeyword, model.TargetTypeProfile, model.TargetTypePostURL:
		return true
	default:
		return false
	}
}

func isValidTriggerType(value string) bool {
	switch model.TriggerType(value) {
	case model.TriggerTypeManual, model.TriggerTypeScheduled, model.TriggerTypeProjectEvent,
		model.TriggerTypeCrisisEvent, model.TriggerTypeWebhookPush:
		return true
	default:
		return false
	}
}

func normalizeRequestValues(values []string, dedupe bool) []string {
	if values == nil {
		return nil
	}
	items := make([]string, 0, len(values))
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
		items = append(items, trimmed)
	}
	return items
}

func validateRequestURLValues(values []string) error {
	for _, value := range values {
		parsed, err := url.ParseRequestURI(value)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return errTargetValuesMustBeURLs
		}
	}
	return nil
}

// --- CrawlTarget Request DTOs ---

// createTargetGroupReq represents grouped crawl target creation request.
type createTargetGroupReq struct {
	DataSourceID         string          `json:"-"`
	Values               []string        `json:"values" binding:"required" example:"vinfast"`
	Label                string          `json:"label" example:"VinFast keyword"`
	PlatformMeta         json.RawMessage `json:"platform_meta,omitempty" swaggertype:"object"`
	IsActive             bool            `json:"is_active" example:"true"`
	Priority             int             `json:"priority" example:"0"`
	CrawlIntervalMinutes int             `json:"crawl_interval_minutes" example:"11"`
}

func (r createTargetGroupReq) validate(targetType model.TargetType) error {
	if strings.TrimSpace(r.DataSourceID) == "" {
		return errWrongBody
	}
	values := normalizeRequestValues(r.Values, targetType == model.TargetTypeKeyword)
	if len(values) == 0 {
		return errTargetValuesRequired
	}
	if targetType == model.TargetTypeProfile || targetType == model.TargetTypePostURL {
		if err := validateRequestURLValues(values); err != nil {
			return err
		}
	}
	if r.CrawlIntervalMinutes <= 0 {
		return errInvalidTargetInterval
	}
	return nil
}

func (r createTargetGroupReq) toInput(targetType model.TargetType) datasource.CreateTargetGroupInput {
	return datasource.CreateTargetGroupInput{
		DataSourceID:         strings.TrimSpace(r.DataSourceID),
		Values:               normalizeRequestValues(r.Values, targetType == model.TargetTypeKeyword),
		Label:                r.Label,
		PlatformMeta:         r.PlatformMeta,
		IsActive:             r.IsActive,
		Priority:             r.Priority,
		CrawlIntervalMinutes: r.CrawlIntervalMinutes,
	}
}

// listTargetsReq binds query params for listing crawl targets.
type listTargetsReq struct {
	DataSourceID string `form:"-"`
	TargetType   string `form:"target_type"`
	IsActive     *bool  `form:"is_active"`
}

func (r listTargetsReq) validate() error {
	if strings.TrimSpace(r.DataSourceID) == "" {
		return errWrongBody
	}
	if strings.TrimSpace(r.TargetType) != "" && !isValidTargetType(strings.TrimSpace(r.TargetType)) {
		return errInvalidTargetType
	}
	return nil
}

func (r listTargetsReq) toInput() datasource.ListTargetsInput {
	return datasource.ListTargetsInput{
		DataSourceID: strings.TrimSpace(r.DataSourceID),
		TargetType:   strings.TrimSpace(r.TargetType),
		IsActive:     r.IsActive,
	}
}

// detailTargetReq extracts datasource + target identity from path params.
type detailTargetReq struct {
	DataSourceID string
	ID           string
}

func (r detailTargetReq) validate() error {
	if strings.TrimSpace(r.DataSourceID) == "" || strings.TrimSpace(r.ID) == "" {
		return errWrongBody
	}
	return nil
}

func (r detailTargetReq) toInput() datasource.DetailTargetInput {
	return datasource.DetailTargetInput{
		DataSourceID: strings.TrimSpace(r.DataSourceID),
		ID:           strings.TrimSpace(r.ID),
	}
}

// updateTargetReq represents crawl target update request.
type updateTargetReq struct {
	DataSourceID         string          `json:"-"`
	ID                   string          `json:"-"`
	Values               []string        `json:"values,omitempty"`
	Label                string          `json:"label" example:"Updated label"`
	PlatformMeta         json.RawMessage `json:"platform_meta,omitempty" swaggertype:"object"`
	IsActive             *bool           `json:"is_active,omitempty"`
	Priority             *int            `json:"priority,omitempty"`
	CrawlIntervalMinutes *int            `json:"crawl_interval_minutes,omitempty"`
}

func (r updateTargetReq) validate() error {
	if strings.TrimSpace(r.DataSourceID) == "" || strings.TrimSpace(r.ID) == "" {
		return errWrongBody
	}
	if r.Values != nil && len(normalizeRequestValues(r.Values, false)) == 0 {
		return errTargetValuesRequired
	}
	if r.CrawlIntervalMinutes != nil && *r.CrawlIntervalMinutes <= 0 {
		return errInvalidTargetInterval
	}
	return nil
}

func (r updateTargetReq) toInput() datasource.UpdateTargetInput {
	return datasource.UpdateTargetInput{
		DataSourceID:         strings.TrimSpace(r.DataSourceID),
		ID:                   strings.TrimSpace(r.ID),
		Values:               normalizeRequestValues(r.Values, false),
		Label:                r.Label,
		PlatformMeta:         r.PlatformMeta,
		IsActive:             r.IsActive,
		Priority:             r.Priority,
		CrawlIntervalMinutes: r.CrawlIntervalMinutes,
	}
}

// deleteTargetReq extracts target_id from path param.
type deleteTargetReq struct {
	DataSourceID string
	ID           string
}

func (r deleteTargetReq) validate() error {
	if strings.TrimSpace(r.DataSourceID) == "" || strings.TrimSpace(r.ID) == "" {
		return errWrongBody
	}
	return nil
}

func (r deleteTargetReq) toInput() datasource.DeleteTargetInput {
	return datasource.DeleteTargetInput{
		DataSourceID: strings.TrimSpace(r.DataSourceID),
		ID:           strings.TrimSpace(r.ID),
	}
}

type detailTargetResp struct {
	Target crawlTargetResp `json:"target"`
}

// --- CrawlTarget Response DTOs ---

// crawlTargetResp represents crawl target data in API responses.
type crawlTargetResp struct {
	ID                   string          `json:"id" example:"660e8400-e29b-41d4-a716-446655440002"`
	DataSourceID         string          `json:"data_source_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	TargetType           string          `json:"target_type" example:"KEYWORD"`
	Values               []string        `json:"values"`
	Label                string          `json:"label,omitempty"`
	PlatformMeta         json.RawMessage `json:"platform_meta,omitempty" swaggertype:"object"`
	IsActive             bool            `json:"is_active" example:"true"`
	Priority             int             `json:"priority" example:"0"`
	CrawlIntervalMinutes int             `json:"crawl_interval_minutes" example:"11"`
	NextCrawlAt          *string         `json:"next_crawl_at,omitempty"`
	LastCrawlAt          *string         `json:"last_crawl_at,omitempty"`
	LastSuccessAt        *string         `json:"last_success_at,omitempty"`
	LastErrorAt          *string         `json:"last_error_at,omitempty"`
	LastErrorMessage     string          `json:"last_error_message,omitempty"`
	CreatedAt            string          `json:"created_at" example:"2026-03-05T00:00:00Z"`
	UpdatedAt            string          `json:"updated_at" example:"2026-03-05T00:00:00Z"`
}

type createTargetResp struct {
	Target crawlTargetResp `json:"target"`
}

type listTargetsResp struct {
	Targets []crawlTargetResp `json:"targets"`
}

type updateTargetResp struct {
	Target crawlTargetResp `json:"target"`
}

// --- CrawlTarget Response Mappers ---

func (h *handler) newCreateTargetResp(o datasource.CreateTargetOutput) createTargetResp {
	return createTargetResp{Target: toCrawlTargetResp(o.Target)}
}

func (h *handler) newListTargetsResp(o datasource.ListTargetsOutput) listTargetsResp {
	items := make([]crawlTargetResp, len(o.Targets))
	for i, t := range o.Targets {
		items[i] = toCrawlTargetResp(t)
	}
	return listTargetsResp{Targets: items}
}

func (h *handler) newDetailTargetResp(o datasource.DetailTargetOutput) detailTargetResp {
	return detailTargetResp{Target: toCrawlTargetResp(o.Target)}
}

func (h *handler) newUpdateTargetResp(o datasource.UpdateTargetOutput) updateTargetResp {
	return updateTargetResp{Target: toCrawlTargetResp(o.Target)}
}

func toCrawlTargetResp(t model.CrawlTarget) crawlTargetResp {
	resp := crawlTargetResp{
		ID:                   t.ID,
		DataSourceID:         t.DataSourceID,
		TargetType:           string(t.TargetType),
		Values:               t.Values,
		Label:                t.Label,
		IsActive:             t.IsActive,
		Priority:             t.Priority,
		CrawlIntervalMinutes: t.CrawlIntervalMinutes,
		LastErrorMessage:     t.LastErrorMessage,
		CreatedAt:            t.CreatedAt.Format(timeFormat),
		UpdatedAt:            t.UpdatedAt.Format(timeFormat),
	}

	if len(t.PlatformMeta) > 0 {
		resp.PlatformMeta = t.PlatformMeta
	}
	resp.NextCrawlAt = formatTimePtr(t.NextCrawlAt)
	resp.LastCrawlAt = formatTimePtr(t.LastCrawlAt)
	resp.LastSuccessAt = formatTimePtr(t.LastSuccessAt)
	resp.LastErrorAt = formatTimePtr(t.LastErrorAt)

	return resp
}
