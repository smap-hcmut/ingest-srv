package http

import (
	"ingest-srv/internal/model"

	"github.com/smap-hcmut/shared-libs/go/response"

	"github.com/gin-gonic/gin"
)

// @Summary Create a data source
// @Description Create a new data source under a project
// @Tags DataSource
// @Accept json
// @Produce json
// @Param body body createReq true "Create data source request"
// @Success 200 {object} createResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /datasources [post]
func (h *handler) Create(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processCreateReq(c)
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.Create.processCreateReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.Create(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "datasource.delivery.Create.uc.Create: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newCreateResp(o))
}

// @Summary Get data source detail
// @Description Return data source info by ID
// @Tags DataSource
// @Produce json
// @Param id path string true "Data Source ID"
// @Success 200 {object} detailResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /datasources/{id} [get]
func (h *handler) Detail(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processDetailReq(c)
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.Detail.processDetailReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.Detail(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "datasource.delivery.Detail.uc.Detail: id=%s err=%v", req.ID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newDetailResp(o))
}

// @Summary List data sources
// @Description Paginate data sources with filters
// @Tags DataSource
// @Produce json
// @Param project_id query string false "Filter by project ID"
// @Param status query string false "Filter by status"
// @Param source_type query string false "Filter by source type"
// @Param source_category query string false "Filter by source category"
// @Param crawl_mode query string false "Filter by crawl mode"
// @Param name query string false "Filter by name (ILIKE)"
// @Param page query int false "Page number (default 1)"
// @Param limit query int false "Number of records per page (default 15)"
// @Success 200 {object} listResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /datasources [get]
func (h *handler) List(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processListReq(c)
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.List.processListReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.List(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "datasource.delivery.List.uc.List: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newListResp(o))
}

// @Summary Update a data source
// @Description Update data source fields by ID
// @Tags DataSource
// @Accept json
// @Produce json
// @Param id path string true "Data Source ID"
// @Param body body updateReq true "Update data source request"
// @Success 200 {object} updateResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /datasources/{id} [put]
func (h *handler) Update(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processUpdateReq(c)
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.Update.processUpdateReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.Update(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "datasource.delivery.Update.uc.Update: id=%s err=%v", req.ID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newUpdateResp(o))
}

// @Summary Archive a data source
// @Description Transition a data source into ARCHIVED status
// @Tags DataSource
// @Produce json
// @Param id path string true "Data Source ID"
// @Success 200 {object} response.Resp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /datasources/{id}/archive [post]
func (h *handler) Archive(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processArchiveReq(c)
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.Archive.processArchiveReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	if err := h.uc.Archive(ctx, req.toInput()); err != nil {
		h.l.Errorf(ctx, "datasource.delivery.Archive.uc.Archive: id=%s err=%v", req.ID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, nil)
}

// @Summary Delete a data source
// @Description Soft-delete a data source after it has been archived
// @Tags DataSource
// @Produce json
// @Param id path string true "Data Source ID"
// @Success 200 {object} response.Resp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /datasources/{id} [delete]
func (h *handler) Delete(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processArchiveReq(c)
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.Delete.processArchiveReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	if err := h.uc.Delete(ctx, req.toInput()); err != nil {
		h.l.Errorf(ctx, "datasource.delivery.Delete.uc.Delete: id=%s err=%v", req.ID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, nil)
}

// @Summary Update datasource crawl mode
// @Description Internal API to update crawl mode for a datasource
// @Tags DataSource
// @Accept json
// @Produce json
// @Param id path string true "Data Source ID"
// @Param body body updateCrawlModeReq true "Update crawl mode request"
// @Success 200 {object} updateCrawlModeResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /internal/datasources/{id}/crawl-mode [put]
func (h *handler) UpdateCrawlMode(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processUpdateCrawlModeReq(c)
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.UpdateCrawlMode.processUpdateCrawlModeReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.UpdateCrawlMode(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "datasource.delivery.UpdateCrawlMode.uc.UpdateCrawlMode: id=%s err=%v", req.ID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newUpdateCrawlModeResp(o))
}

// @Summary Get project activation readiness
// @Description Internal API to evaluate project readiness from datasource/target dryrun state
// @Tags DataSource
// @Produce json
// @Param project_id path string true "Project ID"
// @Param command query string false "Lifecycle command to evaluate" Enums(activate,resume) default(activate)
// @Success 200 {object} activationReadinessResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /internal/projects/{project_id}/activation-readiness [get]
func (h *handler) GetActivationReadiness(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processActivationReadinessReq(c)
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.GetActivationReadiness.processActivationReadinessReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.GetActivationReadiness(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "datasource.delivery.GetActivationReadiness.uc.GetActivationReadiness: project_id=%s command=%s err=%v", req.ProjectID, req.Command, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newActivationReadinessResp(o))
}

// @Summary Activate all datasources in a project
// @Description Internal API to activate project-level datasource runtime with fail-fast semantics
// @Tags DataSource
// @Produce json
// @Param project_id path string true "Project ID"
// @Success 200 {object} projectLifecycleResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /internal/projects/{project_id}/activate [post]
func (h *handler) Activate(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processProjectLifecycleReq(c)
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.Activate.processProjectLifecycleReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.Activate(ctx, req.toProjectID())
	if err != nil {
		h.l.Errorf(ctx, "datasource.delivery.Activate.uc.Activate: project_id=%s err=%v", req.ProjectID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newProjectLifecycleResp(o))
}

// @Summary Pause all active datasources in a project
// @Description Internal API to pause project-level datasource runtime
// @Tags DataSource
// @Produce json
// @Param project_id path string true "Project ID"
// @Success 200 {object} projectLifecycleResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /internal/projects/{project_id}/pause [post]
func (h *handler) Pause(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processProjectLifecycleReq(c)
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.Pause.processProjectLifecycleReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.Pause(ctx, req.toProjectID())
	if err != nil {
		h.l.Errorf(ctx, "datasource.delivery.Pause.uc.Pause: project_id=%s err=%v", req.ProjectID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newProjectLifecycleResp(o))
}

// @Summary Resume all paused datasources in a project
// @Description Internal API to resume project-level datasource runtime after readiness passes
// @Tags DataSource
// @Produce json
// @Param project_id path string true "Project ID"
// @Success 200 {object} projectLifecycleResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /internal/projects/{project_id}/resume [post]
func (h *handler) Resume(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processProjectLifecycleReq(c)
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.Resume.processProjectLifecycleReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.Resume(ctx, req.toProjectID())
	if err != nil {
		h.l.Errorf(ctx, "datasource.delivery.Resume.uc.Resume: project_id=%s err=%v", req.ProjectID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newProjectLifecycleResp(o))
}

// --- CrawlTarget Handlers ---

// @Summary Create grouped keyword target
// @Description Create a new grouped keyword target under a data source
// @Tags CrawlTarget
// @Accept json
// @Produce json
// @Param id path string true "Data Source ID"
// @Param body body createTargetGroupReq true "Create target request"
// @Success 200 {object} createTargetResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /datasources/{id}/targets/keywords [post]
func (h *handler) CreateKeywordTarget(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processCreateTargetGroupReq(c, string(model.TargetTypeKeyword))
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.CreateKeywordTarget.processCreateTargetGroupReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.CreateKeywordTarget(ctx, req.toInput(model.TargetTypeKeyword))
	if err != nil {
		h.l.Errorf(ctx, "datasource.delivery.CreateKeywordTarget.uc.CreateKeywordTarget: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newCreateTargetResp(o))
}

// @Summary Create grouped profile target
// @Description Create a new grouped profile target under a data source
// @Tags CrawlTarget
// @Accept json
// @Produce json
// @Param id path string true "Data Source ID"
// @Param body body createTargetGroupReq true "Create target request"
// @Success 200 {object} createTargetResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /datasources/{id}/targets/profiles [post]
func (h *handler) CreateProfileTarget(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processCreateTargetGroupReq(c, string(model.TargetTypeProfile))
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.CreateProfileTarget.processCreateTargetGroupReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.CreateProfileTarget(ctx, req.toInput(model.TargetTypeProfile))
	if err != nil {
		h.l.Errorf(ctx, "datasource.delivery.CreateProfileTarget.uc.CreateProfileTarget: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newCreateTargetResp(o))
}

// @Summary Create grouped post target
// @Description Create a new grouped post target under a data source
// @Tags CrawlTarget
// @Accept json
// @Produce json
// @Param id path string true "Data Source ID"
// @Param body body createTargetGroupReq true "Create target request"
// @Success 200 {object} createTargetResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /datasources/{id}/targets/posts [post]
func (h *handler) CreatePostTarget(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processCreateTargetGroupReq(c, string(model.TargetTypePostURL))
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.CreatePostTarget.processCreateTargetGroupReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.CreatePostTarget(ctx, req.toInput(model.TargetTypePostURL))
	if err != nil {
		h.l.Errorf(ctx, "datasource.delivery.CreatePostTarget.uc.CreatePostTarget: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newCreateTargetResp(o))
}

// @Summary List crawl targets
// @Description List crawl targets of a data source
// @Tags CrawlTarget
// @Produce json
// @Param id path string true "Data Source ID"
// @Param target_type query string false "Filter by target type" enums(KEYWORD,PROFILE,POST_URL)
// @Param is_active query bool false "Filter by active status"
// @Success 200 {object} listTargetsResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /datasources/{id}/targets [get]
func (h *handler) ListTargets(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processListTargetsReq(c)
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.ListTargets.processListTargetsReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.ListTargets(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "datasource.delivery.ListTargets.uc.ListTargets: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newListTargetsResp(o))
}

// @Summary Get crawl target detail
// @Description Return crawl target info by ID under a data source
// @Tags CrawlTarget
// @Produce json
// @Param id path string true "Data Source ID"
// @Param target_id path string true "Target ID"
// @Success 200 {object} detailTargetResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /datasources/{id}/targets/{target_id} [get]
func (h *handler) DetailTarget(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processDetailTargetReq(c)
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.DetailTarget.processDetailTargetReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.DetailTarget(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "datasource.delivery.DetailTarget.uc.DetailTarget: id=%s err=%v", req.ID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newDetailTargetResp(o))
}

// @Summary Update a crawl target
// @Description Update crawl target fields by ID
// @Tags CrawlTarget
// @Accept json
// @Produce json
// @Param id path string true "Data Source ID"
// @Param target_id path string true "Target ID"
// @Param body body updateTargetReq true "Update target request"
// @Success 200 {object} updateTargetResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /datasources/{id}/targets/{target_id} [put]
func (h *handler) UpdateTarget(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processUpdateTargetReq(c)
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.UpdateTarget.processUpdateTargetReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.UpdateTarget(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "datasource.delivery.UpdateTarget.uc.UpdateTarget: id=%s err=%v", req.ID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newUpdateTargetResp(o))
}

// @Summary Activate a crawl target
// @Description Activate one crawl target after dry-run evidence is available
// @Tags CrawlTarget
// @Produce json
// @Param id path string true "Data Source ID"
// @Param target_id path string true "Target ID"
// @Success 200 {object} updateTargetResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /datasources/{id}/targets/{target_id}/activate [post]
func (h *handler) ActivateTarget(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processDetailTargetReq(c)
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.ActivateTarget.processDetailTargetReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.ActivateTarget(ctx, req.toActivateInput())
	if err != nil {
		h.l.Errorf(ctx, "datasource.delivery.ActivateTarget.uc.ActivateTarget: id=%s err=%v", req.ID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, updateTargetResp{Target: toCrawlTargetResp(o.Target)})
}

// @Summary Deactivate a crawl target
// @Description Deactivate one crawl target manually
// @Tags CrawlTarget
// @Produce json
// @Param id path string true "Data Source ID"
// @Param target_id path string true "Target ID"
// @Success 200 {object} updateTargetResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /datasources/{id}/targets/{target_id}/deactivate [post]
func (h *handler) DeactivateTarget(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processDetailTargetReq(c)
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.DeactivateTarget.processDetailTargetReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.DeactivateTarget(ctx, req.toDeactivateInput())
	if err != nil {
		h.l.Errorf(ctx, "datasource.delivery.DeactivateTarget.uc.DeactivateTarget: id=%s err=%v", req.ID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, updateTargetResp{Target: toCrawlTargetResp(o.Target)})
}

// @Summary Delete a crawl target
// @Description Hard-delete a crawl target by ID
// @Tags CrawlTarget
// @Produce json
// @Param id path string true "Data Source ID"
// @Param target_id path string true "Target ID"
// @Success 200 {object} response.Resp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /datasources/{id}/targets/{target_id} [delete]
func (h *handler) DeleteTarget(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processDeleteTargetReq(c)
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.DeleteTarget.processDeleteTargetReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	if err := h.uc.DeleteTarget(ctx, req.toInput()); err != nil {
		h.l.Errorf(ctx, "datasource.delivery.DeleteTarget.uc.DeleteTarget: id=%s err=%v", req.ID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, nil)
}
