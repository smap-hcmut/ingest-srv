package http

import (
	"errors"
	"net/http"
	"testing"

	"ingest-srv/internal/datasource"
	"ingest-srv/internal/model"

	pkgErrors "github.com/smap-hcmut/shared-libs/go/errors"
	"github.com/stretchr/testify/require"
)

func TestMapError(t *testing.T) {
	type mockData struct{}

	tcs := map[string]struct {
		input  error
		mock   mockData
		output int
		err    error
	}{
		"not_found":                     {input: datasource.ErrNotFound, output: http.StatusNotFound},
		"name_required":                 {input: datasource.ErrNameRequired, output: http.StatusBadRequest},
		"project_id_required":           {input: datasource.ErrProjectIDRequired, output: http.StatusBadRequest},
		"project_not_found":             {input: datasource.ErrProjectNotFound, output: http.StatusNotFound},
		"project_archived":              {input: datasource.ErrProjectArchived, output: http.StatusBadRequest},
		"source_dryrun_running":         {input: datasource.ErrSourceDryrunRunning, output: http.StatusBadRequest},
		"source_type_required":          {input: datasource.ErrSourceTypeRequired, output: http.StatusBadRequest},
		"invalid_source_type":           {input: datasource.ErrInvalidSourceType, output: http.StatusBadRequest},
		"invalid_category":              {input: datasource.ErrInvalidCategory, output: http.StatusBadRequest},
		"invalid_crawl_mode":            {input: datasource.ErrInvalidCrawlMode, output: http.StatusBadRequest},
		"crawl_config_required":         {input: datasource.ErrCrawlConfigRequired, output: http.StatusBadRequest},
		"create_failed":                 {input: datasource.ErrCreateFailed, output: http.StatusInternalServerError},
		"update_failed":                 {input: datasource.ErrUpdateFailed, output: http.StatusInternalServerError},
		"delete_failed":                 {input: datasource.ErrDeleteFailed, output: http.StatusInternalServerError},
		"delete_requires_archived":      {input: datasource.ErrDeleteRequiresArchived, output: http.StatusBadRequest},
		"source_archived":               {input: datasource.ErrSourceArchived, output: http.StatusBadRequest},
		"list_failed":                   {input: datasource.ErrListFailed, output: http.StatusInternalServerError},
		"update_not_allowed":            {input: datasource.ErrUpdateNotAllowed, output: http.StatusBadRequest},
		"invalid_transition":            {input: datasource.ErrInvalidTransition, output: http.StatusBadRequest},
		"activate_not_allowed":          {input: datasource.ErrActivateNotAllowed, output: http.StatusBadRequest},
		"pause_not_allowed":             {input: datasource.ErrPauseNotAllowed, output: http.StatusBadRequest},
		"resume_not_allowed":            {input: datasource.ErrResumeNotAllowed, output: http.StatusBadRequest},
		"crawl_mode_not_allowed":        {input: datasource.ErrCrawlModeNotAllowed, output: http.StatusBadRequest},
		"activation_readiness_failed":   {input: datasource.ErrActivationReadinessFailed, output: http.StatusBadRequest},
		"invalid_readiness_command":     {input: datasource.ErrInvalidReadinessCommand, output: http.StatusBadRequest},
		"target_not_found":              {input: datasource.ErrTargetNotFound, output: http.StatusNotFound},
		"target_values_required":        {input: datasource.ErrTargetValuesRequired, output: http.StatusBadRequest},
		"target_values_must_be_urls":    {input: datasource.ErrTargetValuesMustBeURLs, output: http.StatusBadRequest},
		"invalid_target_type":           {input: datasource.ErrInvalidTargetType, output: http.StatusBadRequest},
		"source_not_crawl":              {input: datasource.ErrSourceNotCrawl, output: http.StatusBadRequest},
		"target_create_failed":          {input: datasource.ErrTargetCreateFailed, output: http.StatusInternalServerError},
		"target_update_failed":          {input: datasource.ErrTargetUpdateFailed, output: http.StatusInternalServerError},
		"target_delete_failed":          {input: datasource.ErrTargetDeleteFailed, output: http.StatusInternalServerError},
		"target_list_failed":            {input: datasource.ErrTargetListFailed, output: http.StatusInternalServerError},
		"invalid_target_interval":       {input: datasource.ErrInvalidTargetInterval, output: http.StatusBadRequest},
		"target_activate_not_allowed":   {input: datasource.ErrTargetActivateNotAllowed, output: http.StatusBadRequest},
		"target_deactivate_not_allowed": {input: datasource.ErrTargetDeactivateNotAllowed, output: http.StatusBadRequest},
		"target_delete_not_allowed":     {input: datasource.ErrTargetDeleteNotAllowed, output: http.StatusBadRequest},
		"target_dryrun_running":         {input: datasource.ErrTargetDryrunRunning, output: http.StatusBadRequest},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, _ := newTestHandler(t)

			got := h.mapError(tc.input)

			var httpErr *pkgErrors.HTTPError
			require.ErrorAs(t, got, &httpErr)
			require.Equal(t, tc.output, httpErr.StatusCode)
		})
	}
}

func TestMapErrorPanic(t *testing.T) {
	tcs := map[string]struct {
		input  error
		mock   struct{}
		output struct{}
		err    error
	}{
		"unknown_error": {input: errors.New("unknown")},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, _ := newTestHandler(t)

			require.Panics(t, func() { _ = h.mapError(tc.input) })
		})
	}
}

func TestCreateReqValidate(t *testing.T) {
	tcs := map[string]struct {
		input  createReq
		mock   struct{}
		output error
		err    error
	}{
		"success_passive_infer":    {input: createReq{ProjectID: testProjectID, Name: "Webhook", SourceType: string(model.SourceTypeWebhook)}, output: nil},
		"project_id_required":      {input: createReq{Name: "Source", SourceType: string(model.SourceTypeTikTok)}, output: errProjectIDRequired},
		"wrong_project_id":         {input: createReq{ProjectID: "bad", Name: "Source", SourceType: string(model.SourceTypeTikTok)}, output: errWrongBody},
		"name_required":            {input: createReq{ProjectID: testProjectID, SourceType: string(model.SourceTypeTikTok)}, output: errNameRequired},
		"source_type_required":     {input: createReq{ProjectID: testProjectID, Name: "Source"}, output: errSourceTypeRequired},
		"invalid_source_type":      {input: createReq{ProjectID: testProjectID, Name: "Source", SourceType: "BAD"}, output: errInvalidSourceType},
		"invalid_category":         {input: createReq{ProjectID: testProjectID, Name: "Source", SourceType: string(model.SourceTypeTikTok), SourceCategory: "BAD"}, output: errInvalidCategory},
		"invalid_crawl_mode":       {input: createReq{ProjectID: testProjectID, Name: "Source", SourceType: string(model.SourceTypeTikTok), CrawlMode: "BAD", CrawlIntervalMinutes: 10}, output: errInvalidCrawlMode},
		"invalid_crawl_interval":   {input: createReq{ProjectID: testProjectID, Name: "Source", SourceType: string(model.SourceTypeWebhook), CrawlIntervalMinutes: -1}, output: errInvalidCrawlInterval},
		"crawl_config_required":    {input: createReq{ProjectID: testProjectID, Name: "Source", SourceType: string(model.SourceTypeTikTok)}, output: errCrawlConfigRequired},
		"success_crawl_explicit":   {input: createReq{ProjectID: testProjectID, Name: "Source", SourceType: string(model.SourceTypeTikTok), SourceCategory: string(model.SourceCategoryCrawl), CrawlMode: string(model.CrawlModeNormal), CrawlIntervalMinutes: 10}, output: nil},
		"success_passive_explicit": {input: createReq{ProjectID: testProjectID, Name: "Source", SourceType: string(model.SourceTypeFileUpload), SourceCategory: string(model.SourceCategoryPassive)}, output: nil},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := tc.input.validate()

			require.ErrorIs(t, err, tc.output)
		})
	}
}

func TestListReqValidate(t *testing.T) {
	tcs := map[string]struct {
		input  listReq
		mock   struct{}
		output error
		err    error
	}{
		"success":                 {input: listReq{ProjectID: testProjectID, SourceType: string(model.SourceTypeTikTok), SourceCategory: string(model.SourceCategoryCrawl), CrawlMode: string(model.CrawlModeNormal)}, output: nil},
		"wrong_project_id":        {input: listReq{ProjectID: "bad"}, output: errWrongBody},
		"invalid_source_type":     {input: listReq{SourceType: "BAD"}, output: errInvalidSourceType},
		"invalid_source_category": {input: listReq{SourceCategory: "BAD"}, output: errInvalidCategory},
		"invalid_crawl_mode":      {input: listReq{CrawlMode: "BAD"}, output: errInvalidCrawlMode},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := tc.input.validate()

			require.ErrorIs(t, err, tc.output)
		})
	}
}

func TestProjectLifecycleReqValidate(t *testing.T) {
	tcs := map[string]struct {
		input  projectLifecycleReq
		mock   struct{}
		output error
		err    error
	}{
		"success":             {input: projectLifecycleReq{ProjectID: testProjectID}, output: nil},
		"project_id_required": {input: projectLifecycleReq{}, output: errProjectIDRequired},
		"wrong_project_id":    {input: projectLifecycleReq{ProjectID: "bad"}, output: errWrongBody},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := tc.input.validate()

			require.ErrorIs(t, err, tc.output)
		})
	}
}

func TestUpdateProjectCrawlModeReqValidate(t *testing.T) {
	tcs := map[string]struct {
		input  updateProjectCrawlModeReq
		mock   struct{}
		output error
		err    error
	}{
		"success":              {input: updateProjectCrawlModeReq{ProjectID: testProjectID, CrawlMode: string(model.CrawlModeCrisis), TriggerType: string(model.TriggerTypeCrisisEvent)}, output: nil},
		"project_id_required":  {input: updateProjectCrawlModeReq{CrawlMode: string(model.CrawlModeCrisis), TriggerType: string(model.TriggerTypeCrisisEvent)}, output: errProjectIDRequired},
		"wrong_project_id":     {input: updateProjectCrawlModeReq{ProjectID: "bad", CrawlMode: string(model.CrawlModeCrisis), TriggerType: string(model.TriggerTypeCrisisEvent)}, output: errWrongBody},
		"invalid_crawl_mode":   {input: updateProjectCrawlModeReq{ProjectID: testProjectID, CrawlMode: "BAD", TriggerType: string(model.TriggerTypeCrisisEvent)}, output: errInvalidCrawlMode},
		"invalid_trigger_type": {input: updateProjectCrawlModeReq{ProjectID: testProjectID, CrawlMode: string(model.CrawlModeCrisis), TriggerType: "BAD"}, output: errCrawlModeNotAllowed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := tc.input.validate()

			require.ErrorIs(t, err, tc.output)
		})
	}
}

func TestUpdateProjectCrawlModeReqToInput(t *testing.T) {
	tcs := map[string]struct {
		input  updateProjectCrawlModeReq
		mock   struct{}
		output datasource.UpdateProjectCrawlModeInput
		err    error
	}{
		"success": {
			input:  updateProjectCrawlModeReq{ProjectID: " " + testProjectID + " ", CrawlMode: " CRISIS ", TriggerType: " CRISIS_EVENT ", Reason: " reason ", EventRef: " event-1 "},
			output: datasource.UpdateProjectCrawlModeInput{ProjectID: testProjectID, CrawlMode: string(model.CrawlModeCrisis), TriggerType: string(model.TriggerTypeCrisisEvent), Reason: "reason", EventRef: "event-1"},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.output, tc.input.toInput())
		})
	}
}

func TestActivationReadinessReqValidate(t *testing.T) {
	tcs := map[string]struct {
		input  activationReadinessReq
		mock   struct{}
		output error
		err    error
	}{
		"success_empty_command": {input: activationReadinessReq{ProjectID: testProjectID}, output: nil},
		"success_activate":      {input: activationReadinessReq{ProjectID: testProjectID, Command: string(datasource.ActivationReadinessCommandActivate)}, output: nil},
		"success_resume":        {input: activationReadinessReq{ProjectID: testProjectID, Command: string(datasource.ActivationReadinessCommandResume)}, output: nil},
		"project_id_required":   {input: activationReadinessReq{}, output: errProjectIDRequired},
		"wrong_project_id":      {input: activationReadinessReq{ProjectID: "bad"}, output: errWrongBody},
		"invalid_command":       {input: activationReadinessReq{ProjectID: testProjectID, Command: "bad"}, output: errInvalidReadinessCommand},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := tc.input.validate()

			require.ErrorIs(t, err, tc.output)
		})
	}
}

func TestUpdateCrawlModeReqValidate(t *testing.T) {
	tcs := map[string]struct {
		input  updateCrawlModeReq
		mock   struct{}
		output error
		err    error
	}{
		"success":              {input: updateCrawlModeReq{ID: testSourceID, CrawlMode: string(model.CrawlModeNormal), TriggerType: string(model.TriggerTypeManual)}, output: nil},
		"wrong_id":             {input: updateCrawlModeReq{ID: "bad", CrawlMode: string(model.CrawlModeNormal), TriggerType: string(model.TriggerTypeManual)}, output: errWrongBody},
		"invalid_crawl_mode":   {input: updateCrawlModeReq{ID: testSourceID, CrawlMode: "BAD", TriggerType: string(model.TriggerTypeManual)}, output: errInvalidCrawlMode},
		"invalid_trigger_type": {input: updateCrawlModeReq{ID: testSourceID, CrawlMode: string(model.CrawlModeNormal), TriggerType: "BAD"}, output: errCrawlModeNotAllowed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := tc.input.validate()

			require.ErrorIs(t, err, tc.output)
		})
	}
}

func TestNewActivationReadinessResp(t *testing.T) {
	tcs := map[string]struct {
		input  datasource.ActivationReadinessOutput
		mock   struct{}
		output activationReadinessResp
		err    error
	}{
		"with_errors": {
			input: datasource.ActivationReadinessOutput{
				ProjectID: testProjectID,
				Command:   datasource.ActivationReadinessCommandActivate,
				Errors:    []datasource.ActivationReadinessError{{Code: datasource.ActivationReadinessCodeTargetDryrunMiss, Message: datasource.ActivationReadinessMessageTargetDryrunMissing, DataSourceID: testSourceID, TargetID: testTargetID}},
			},
			output: activationReadinessResp{ProjectID: testProjectID, Command: string(datasource.ActivationReadinessCommandActivate), Errors: []activationReadinessErrorResp{{Code: string(datasource.ActivationReadinessCodeTargetDryrunMiss), Message: datasource.ActivationReadinessMessageTargetDryrunMissing, DataSourceID: testSourceID, TargetID: testTargetID}}},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, _ := newTestHandler(t)

			got := h.newActivationReadinessResp(tc.input)

			require.Equal(t, tc.output, got)
		})
	}
}

func TestFormatTimePtr(t *testing.T) {
	tcs := map[string]struct {
		input  bool
		mock   struct{}
		output bool
		err    error
	}{
		"nil":     {input: false, output: false},
		"present": {input: true, output: true},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			var got *string
			if tc.input {
				now := testDataSource().CreatedAt
				got = formatTimePtr(&now)
			} else {
				got = formatTimePtr(nil)
			}

			require.Equal(t, tc.output, got != nil)
		})
	}
}

func TestDatasourceValidators(t *testing.T) {
	tcs := map[string]struct {
		input  string
		mock   struct{}
		output bool
		err    error
	}{
		"source_category_valid":   {input: string(model.SourceCategoryCrawl), output: true},
		"source_category_invalid": {input: "BAD", output: false},
		"infer_passive_file":      {input: string(model.SourceTypeFileUpload), output: true},
		"infer_passive_webhook":   {input: string(model.SourceTypeWebhook), output: true},
		"infer_crawl_default":     {input: string(model.SourceTypeTikTok), output: false},
		"trigger_scheduled_valid": {input: string(model.TriggerTypeScheduled), output: true},
		"trigger_invalid":         {input: "BAD", output: false},
		"url_valid":               {input: "https://example.com/a", output: true},
		"url_invalid":             {input: "not-url", output: false},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			switch name {
			case "source_category_valid", "source_category_invalid":
				require.Equal(t, tc.output, isValidSourceCategory(tc.input))
			case "infer_passive_file", "infer_passive_webhook":
				require.Equal(t, string(model.SourceCategoryPassive), inferSourceCategory(tc.input))
			case "infer_crawl_default":
				require.Equal(t, string(model.SourceCategoryCrawl), inferSourceCategory(tc.input))
			case "trigger_scheduled_valid", "trigger_invalid":
				require.Equal(t, tc.output, isValidTriggerType(tc.input))
			case "url_valid":
				require.NoError(t, validateRequestURLValues([]string{tc.input}))
			case "url_invalid":
				require.ErrorIs(t, validateRequestURLValues([]string{tc.input}), errTargetValuesMustBeURLs)
			}
		})
	}
}

func TestCreateTargetGroupReqValidate(t *testing.T) {
	tcs := map[string]struct {
		input  createTargetGroupReq
		mock   model.TargetType
		output error
		err    error
	}{
		"success_keyword":     {input: createTargetGroupReq{DataSourceID: testSourceID, Values: []string{"a"}, CrawlIntervalMinutes: 10}, mock: model.TargetTypeKeyword, output: nil},
		"success_profile_url": {input: createTargetGroupReq{DataSourceID: testSourceID, Values: []string{"https://example.com/u"}, CrawlIntervalMinutes: 10}, mock: model.TargetTypeProfile, output: nil},
		"success_post_url":    {input: createTargetGroupReq{DataSourceID: testSourceID, Values: []string{"https://example.com/p"}, CrawlIntervalMinutes: 10}, mock: model.TargetTypePostURL, output: nil},
		"wrong_id":            {input: createTargetGroupReq{DataSourceID: "bad", Values: []string{"a"}, CrawlIntervalMinutes: 10}, mock: model.TargetTypeKeyword, output: errWrongBody},
		"values_required":     {input: createTargetGroupReq{DataSourceID: testSourceID, Values: []string{" "}, CrawlIntervalMinutes: 10}, mock: model.TargetTypeKeyword, output: errTargetValuesRequired},
		"values_must_be_url":  {input: createTargetGroupReq{DataSourceID: testSourceID, Values: []string{"not-url"}, CrawlIntervalMinutes: 10}, mock: model.TargetTypeProfile, output: errTargetValuesMustBeURLs},
		"invalid_interval":    {input: createTargetGroupReq{DataSourceID: testSourceID, Values: []string{"a"}}, mock: model.TargetTypeKeyword, output: errInvalidTargetInterval},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := tc.input.validate(tc.mock)

			require.ErrorIs(t, err, tc.output)
		})
	}
}

func TestListTargetsReqValidate(t *testing.T) {
	tcs := map[string]struct {
		input  listTargetsReq
		mock   struct{}
		output error
		err    error
	}{
		"success_empty_type":  {input: listTargetsReq{DataSourceID: testSourceID}, output: nil},
		"success_type":        {input: listTargetsReq{DataSourceID: testSourceID, TargetType: string(model.TargetTypePostURL)}, output: nil},
		"wrong_id":            {input: listTargetsReq{DataSourceID: "bad"}, output: errWrongBody},
		"invalid_target_type": {input: listTargetsReq{DataSourceID: testSourceID, TargetType: "BAD"}, output: errInvalidTargetType},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := tc.input.validate()

			require.ErrorIs(t, err, tc.output)
		})
	}
}

func TestUpdateTargetReqValidate(t *testing.T) {
	zero := 0
	one := 1
	tcs := map[string]struct {
		input  updateTargetReq
		mock   struct{}
		output error
		err    error
	}{
		"success":          {input: updateTargetReq{DataSourceID: testSourceID, ID: testTargetID, Values: []string{"a"}, CrawlIntervalMinutes: &one}, output: nil},
		"wrong_id":         {input: updateTargetReq{DataSourceID: "bad", ID: testTargetID}, output: errWrongBody},
		"values_required":  {input: updateTargetReq{DataSourceID: testSourceID, ID: testTargetID, Values: []string{" "}}, output: errTargetValuesRequired},
		"invalid_interval": {input: updateTargetReq{DataSourceID: testSourceID, ID: testTargetID, CrawlIntervalMinutes: &zero}, output: errInvalidTargetInterval},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := tc.input.validate()

			require.ErrorIs(t, err, tc.output)
		})
	}
}
