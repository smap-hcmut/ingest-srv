package http

import (
	"ingest-srv/pkg/response"

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
// @Router /sources [post]
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
// @Router /sources/{id} [get]
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
// @Router /sources [get]
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
// @Router /sources/{id} [put]
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
// @Description Soft-delete a data source by ID
// @Tags DataSource
// @Produce json
// @Param id path string true "Data Source ID"
// @Success 200 {object} response.Resp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /sources/{id} [delete]
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

// --- CrawlTarget Handlers ---

// @Summary Create a crawl target
// @Description Create a new crawl target under a data source
// @Tags CrawlTarget
// @Accept json
// @Produce json
// @Param id path string true "Data Source ID"
// @Param body body createTargetReq true "Create target request"
// @Success 200 {object} createTargetResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /datasources/{id}/targets [post]
func (h *handler) CreateTarget(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processCreateTargetReq(c)
	if err != nil {
		h.l.Warnf(ctx, "datasource.delivery.CreateTarget.processCreateTargetReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.CreateTarget(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "datasource.delivery.CreateTarget.uc.CreateTarget: %v", err)
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

	if err := h.uc.DeleteTarget(ctx, req.ID); err != nil {
		h.l.Errorf(ctx, "datasource.delivery.DeleteTarget.uc.DeleteTarget: id=%s err=%v", req.ID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, nil)
}
