package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zalhui/pull-request-service/internal/service"
)

func (h *Handler) handleError(c *gin.Context, err error) {
	switch err {
	case service.ErrTeamExists:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "TEAM_EXISTS",
				"message": "team_name already exists",
			},
		})
	case service.ErrPRExists:
		c.JSON(http.StatusConflict, gin.H{
			"error": gin.H{
				"code":    "PR_EXISTS",
				"message": "PR id already exists",
			},
		})
	case service.ErrPRMerged:
		c.JSON(http.StatusConflict, gin.H{
			"error": gin.H{
				"code":    "PR_MERGED",
				"message": "cannot reassign on merged PR",
			},
		})
	case service.ErrNotAssigned:
		c.JSON(http.StatusConflict, gin.H{
			"error": gin.H{
				"code":    "NOT_ASSIGNED",
				"message": "reviewer is not assigned to this PR",
			},
		})
	case service.ErrNoCandidate:
		c.JSON(http.StatusConflict, gin.H{
			"error": gin.H{
				"code":    "NO_CANDIDATE",
				"message": "no active replacement candidate in team",
			},
		})
	case service.ErrNotFound:
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "resource not found",
			},
		})
	default:
		h.log.Errorw("Internal server error", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "internal server error",
			},
		})
	}
}
