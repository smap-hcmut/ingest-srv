package http

import (
	"ingest-srv/internal/model"

	"github.com/gin-gonic/gin"
)

// processCreateReq binds, validates request for create.
func (h *handler) processCreateReq(c *gin.Context) (createReq, error) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processCreateReq.ShouldBindJSON: %v", err)
		return req, errWrongBody
	}
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processCreateReq.validate: %v", err)
		return req, err
	}
	return req, nil
}

// processDetailReq extracts path param for detail.
func (h *handler) processDetailReq(c *gin.Context) (detailReq, error) {
	req := detailReq{ID: c.Param("id")}
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processDetailReq.validate: %v", err)
		return req, err
	}
	return req, nil
}

// processListReq binds query params for listing.
func (h *handler) processListReq(c *gin.Context) (listReq, error) {
	var req listReq
	if err := c.ShouldBindQuery(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processListReq.ShouldBindQuery: %v", err)
		return req, errWrongBody
	}
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processListReq.validate: %v", err)
		return req, err
	}
	return req, nil
}

// processUpdateReq binds and extracts path param for update.
func (h *handler) processUpdateReq(c *gin.Context) (updateReq, error) {
	var req updateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processUpdateReq.ShouldBindJSON: %v", err)
		return req, errWrongBody
	}
	req.ID = c.Param("id")
	if detailErr := (detailReq{ID: req.ID}).validate(); detailErr != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processUpdateReq.validatePath: %v", detailErr)
		return req, detailErr
	}
	return req, nil
}

// processArchiveReq extracts path param for archive.
func (h *handler) processArchiveReq(c *gin.Context) (archiveReq, error) {
	req := archiveReq{ID: c.Param("id")}
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processArchiveReq.validate: %v", err)
		return req, err
	}
	return req, nil
}

// processUpdateCrawlModeReq binds internal crawl-mode update requests.
func (h *handler) processUpdateCrawlModeReq(c *gin.Context) (updateCrawlModeReq, error) {
	var req updateCrawlModeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processUpdateCrawlModeReq.ShouldBindJSON: %v", err)
		return req, errWrongBody
	}
	req.ID = c.Param("id")
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processUpdateCrawlModeReq.validate: %v", err)
		return req, err
	}
	return req, nil
}

// processProjectLifecycleReq extracts project_id from internal project lifecycle routes.
func (h *handler) processProjectLifecycleReq(c *gin.Context) (projectLifecycleReq, error) {
	req := projectLifecycleReq{ProjectID: c.Param("project_id")}
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processProjectLifecycleReq.validate: %v", err)
		return req, err
	}
	return req, nil
}

// --- CrawlTarget Request Processors ---

// processCreateTargetGroupReq binds JSON + path param for creating a grouped target.
func (h *handler) processCreateTargetGroupReq(c *gin.Context, targetType string) (createTargetGroupReq, error) {
	var req createTargetGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processCreateTargetGroupReq.ShouldBindJSON: %v", err)
		return req, errWrongBody
	}
	req.DataSourceID = c.Param("id")
	if err := req.validate(model.TargetType(targetType)); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processCreateTargetGroupReq.validate: %v", err)
		return req, err
	}
	return req, nil
}

// processListTargetsReq binds query params + path param for listing targets.
func (h *handler) processListTargetsReq(c *gin.Context) (listTargetsReq, error) {
	var req listTargetsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processListTargetsReq.ShouldBindQuery: %v", err)
		return req, errWrongBody
	}
	req.DataSourceID = c.Param("id")
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processListTargetsReq.validate: %v", err)
		return req, err
	}
	return req, nil
}

// processDetailTargetReq extracts datasource + target ids from path params.
func (h *handler) processDetailTargetReq(c *gin.Context) (detailTargetReq, error) {
	req := detailTargetReq{
		DataSourceID: c.Param("id"),
		ID:           c.Param("target_id"),
	}
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processDetailTargetReq.validate: %v", err)
		return req, err
	}
	return req, nil
}

// processUpdateTargetReq binds JSON + path param for updating a target.
func (h *handler) processUpdateTargetReq(c *gin.Context) (updateTargetReq, error) {
	var req updateTargetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processUpdateTargetReq.ShouldBindJSON: %v", err)
		return req, errWrongBody
	}
	req.DataSourceID = c.Param("id")
	req.ID = c.Param("target_id")
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processUpdateTargetReq.validate: %v", err)
		return req, err
	}
	return req, nil
}

// processDeleteTargetReq extracts target_id from path param.
func (h *handler) processDeleteTargetReq(c *gin.Context) (deleteTargetReq, error) {
	req := deleteTargetReq{
		DataSourceID: c.Param("id"),
		ID:           c.Param("target_id"),
	}
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processDeleteTargetReq.validate: %v", err)
		return req, err
	}
	return req, nil
}
