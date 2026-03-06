package http

import (
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
	return detailReq{ID: c.Param("id")}, nil
}

// processListReq binds query params for listing.
func (h *handler) processListReq(c *gin.Context) (listReq, error) {
	var req listReq
	if err := c.ShouldBindQuery(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processListReq.ShouldBindQuery: %v", err)
		return req, errWrongBody
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
	return req, nil
}

// processArchiveReq extracts path param for archive.
func (h *handler) processArchiveReq(c *gin.Context) (archiveReq, error) {
	return archiveReq{ID: c.Param("id")}, nil
}

// --- CrawlTarget Request Processors ---

// processCreateTargetReq binds JSON + path param for creating a target.
func (h *handler) processCreateTargetReq(c *gin.Context) (createTargetReq, error) {
	var req createTargetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processCreateTargetReq.ShouldBindJSON: %v", err)
		return req, errWrongBody
	}
	req.DataSourceID = c.Param("id")
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
	return req, nil
}

// processUpdateTargetReq binds JSON + path param for updating a target.
func (h *handler) processUpdateTargetReq(c *gin.Context) (updateTargetReq, error) {
	var req updateTargetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "datasource.delivery.processUpdateTargetReq.ShouldBindJSON: %v", err)
		return req, errWrongBody
	}
	req.ID = c.Param("target_id")
	return req, nil
}

// processDeleteTargetReq extracts target_id from path param.
func (h *handler) processDeleteTargetReq(c *gin.Context) (deleteTargetReq, error) {
	return deleteTargetReq{ID: c.Param("target_id")}, nil
}
