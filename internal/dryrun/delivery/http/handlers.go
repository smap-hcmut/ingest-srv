package http

import (
	"github.com/smap-hcmut/shared-libs/go/response"

	"github.com/gin-gonic/gin"
)

// @Summary Trigger dryrun
// @Description Trigger one async dryrun for one datasource; response returns a RUNNING result accepted by ingest-srv
// @Tags Dryrun
// @Accept json
// @Produce json
// @Param id path string true "Data Source ID"
// @Param body body triggerReq true "Dryrun trigger request"
// @Success 200 {object} triggerResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /datasources/{id}/dryrun [post]
func (h *handler) Trigger(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processTriggerReq(c)
	if err != nil {
		h.l.Warnf(ctx, "dryrun.delivery.Trigger.processTriggerReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.Trigger(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "dryrun.delivery.Trigger.uc.Trigger: source=%s err=%v", req.SourceID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newTriggerResp(o))
}

// @Summary Get latest dryrun result
// @Description Return latest dryrun result for datasource or datasource-target pair
// @Tags Dryrun
// @Produce json
// @Param id path string true "Data Source ID"
// @Param target_id query string false "Target ID"
// @Success 200 {object} latestResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /datasources/{id}/dryrun/latest [get]
func (h *handler) GetLatest(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processLatestReq(c)
	if err != nil {
		h.l.Warnf(ctx, "dryrun.delivery.GetLatest.processLatestReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.GetLatest(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "dryrun.delivery.GetLatest.uc.GetLatest: source=%s err=%v", req.SourceID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newLatestResp(o))
}

// @Summary Get dryrun history
// @Description Return paginated dryrun history for datasource or datasource-target pair
// @Tags Dryrun
// @Produce json
// @Param id path string true "Data Source ID"
// @Param target_id query string false "Target ID"
// @Param page query int false "Page number"
// @Param limit query int false "Page size"
// @Success 200 {object} historyResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /datasources/{id}/dryrun/history [get]
func (h *handler) ListHistory(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processHistoryReq(c)
	if err != nil {
		h.l.Warnf(ctx, "dryrun.delivery.ListHistory.processHistoryReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.ListHistory(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "dryrun.delivery.ListHistory.uc.ListHistory: source=%s err=%v", req.SourceID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newHistoryResp(o))
}
