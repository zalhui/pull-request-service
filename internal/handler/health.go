package handler

import "github.com/gin-gonic/gin"

func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "ok",
	})
}
