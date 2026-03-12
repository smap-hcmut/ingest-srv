package http

import (
	"github.com/smap-hcmut/shared-libs/go/response"

	"github.com/gin-gonic/gin"
)

// @Summary Dispatch one crawl target manually
// @Description Internal API to create one scheduled_job, fan out one-or-many external_tasks, and publish RabbitMQ task(s)
// @Tags Execution
// @Produce json
// @Param id path string true "Data Source ID"
// @Param target_id path string true "Crawl Target ID"
// @Success 200 {object} dispatchResp
// @Failure 401 {object} response.Resp
// @Failure 404 {object} response.Resp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /ingest/datasources/{id}/targets/{target_id}/dispatch [post]
func (h *handler) DispatchTarget(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processDispatchReq(c)
	if err != nil {
		h.l.Warnf(ctx, "execution.delivery.DispatchTarget.processDispatchReq: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.DispatchTargetManually(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "execution.delivery.DispatchTarget.uc.DispatchTargetManually: datasource_id=%s target_id=%s err=%v", req.DataSourceID, req.TargetID, err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newDispatchResp(o))
}
