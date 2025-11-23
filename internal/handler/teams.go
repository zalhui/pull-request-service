package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zalhui/pull-request-service/internal/models"
)

func (h *Handler) CreateTeam(c *gin.Context) {
	var team models.Team

	if err := c.ShouldBindJSON(&team); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "invalid request body",
			},
		})
		return
	}

	if team.TeamName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "team name is required",
			},
		})
		return
	}
	if len(team.Members) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "team requires at least one member",
			},
		})
		return
	}

	ctx := c.Request.Context()
	err := h.service.CreateTeam(ctx, &team)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"team": team,
	})
}
func (h *Handler) GetTeam(c *gin.Context) {
	teamName := c.Query("team_name")
	if teamName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "team_name query parameter is required",
			},
		})
		return
	}

	ctx := c.Request.Context()
	team, err := h.service.GetTeam(ctx, teamName)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, team)
}
