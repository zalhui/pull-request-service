package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zalhui/pull-request-service/internal/handler/dto"
	"github.com/zalhui/pull-request-service/internal/models"
)

func (h *Handler) CreatePullRequest(c *gin.Context) {
	var req dto.CreatePRRequest
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
	pr, err := h.service.CreatePullRequest(ctx, &models.PullRequest{
		PullRequestID:   req.PullRequestID,
		PullRequestName: req.PullRequestName,
		AuthorID:        req.AuthorID,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"pull_request": pr,
	})
}
func (h *Handler) MergePullRequest(c *gin.Context) {
	var req dto.MergePRRequest

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
	pr, err := h.service.MergePullRequest(ctx, req.PullRequestID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"pull_request": pr,
	})
}
func (h *Handler) ReassignReviewer(c *gin.Context) {
	var req dto.ReassignReviewerRequest
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
	pr, newReviewerID, err := h.service.ReassignReviewer(ctx, req.PullRequestID, req.OldUserID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ReassignReviewerResponse{
		PR:         pr,
		ReplacedBy: newReviewerID,
	})
}
