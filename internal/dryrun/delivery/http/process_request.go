package http

import "github.com/gin-gonic/gin"

func (h *handler) processTriggerReq(c *gin.Context) (triggerReq, error) {
	var req triggerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "dryrun.delivery.processTriggerReq.ShouldBindJSON: %v", err)
		return req, errWrongBody
	}
	req.SourceID = c.Param("id")
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "dryrun.delivery.processTriggerReq.validate: %v", err)
		return req, err
	}
	return req, nil
}

func (h *handler) processLatestReq(c *gin.Context) (latestReq, error) {
	var req latestReq
	if err := c.ShouldBindQuery(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "dryrun.delivery.processLatestReq.ShouldBindQuery: %v", err)
		return req, errWrongBody
	}
	req.SourceID = c.Param("id")
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "dryrun.delivery.processLatestReq.validate: %v", err)
		return req, err
	}
	return req, nil
}

func (h *handler) processHistoryReq(c *gin.Context) (historyReq, error) {
	var req historyReq
	if err := c.ShouldBindQuery(&req); err != nil {
		h.l.Warnf(c.Request.Context(), "dryrun.delivery.processHistoryReq.ShouldBindQuery: %v", err)
		return req, errWrongBody
	}
	req.SourceID = c.Param("id")
	if err := req.validate(); err != nil {
		h.l.Warnf(c.Request.Context(), "dryrun.delivery.processHistoryReq.validate: %v", err)
		return req, err
	}
	return req, nil
}
