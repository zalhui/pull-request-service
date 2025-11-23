package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zalhui/pull-request-service/internal/handler/dto"
)

func (h *Handler) SetUserActive(c *gin.Context) {
	var req dto.SetUserActiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "invalid request body",
			},
		})
		return
	}

	ctx := c.Request.Context()
	user, err := h.service.UpdateUserActivity(ctx, req.UserID, req.IsActive)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

func (h *Handler) GetUserReviewPRs(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "user_id query parameter is required",
			},
		})
		return
	}

	ctx := c.Request.Context()
	prs, err := h.service.GetUserReviewPRs(ctx, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user_id":       userID,
		"pull_requests": prs,
	})
}
