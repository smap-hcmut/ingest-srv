package http

import "github.com/gin-gonic/gin"

func (h *handler) processDispatchReq(c *gin.Context) (dispatchReq, error) {
	req := dispatchReq{
		DataSourceID: c.Param("id"),
		TargetID:     c.Param("target_id"),
	}
	if err := req.validate(); err != nil {
		return req, err
	}
	return req, nil
}
