package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetStats(c *gin.Context) {
	ctx := c.Request.Context()
	stats, err := h.service.GetStats(ctx)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}
