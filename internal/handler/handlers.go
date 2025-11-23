package handler

import (
	"github.com/zalhui/pull-request-service/internal/service"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *service.Service
	log     *zap.SugaredLogger
}

func NewHandler(service *service.Service, log *zap.SugaredLogger) *Handler {
	return &Handler{
		service: service,
		log:     log,
	}
}

func (h *Handler) SetupRouter() *gin.Engine {
	router := gin.Default()

	// Health check
	router.GET("/health", h.HealthCheck)

	// Teams
	router.POST("/team/add", h.CreateTeam)
	router.GET("/team/get", h.GetTeam)

	// Users
	router.POST("/users/setIsActive", h.SetUserActive)
	router.GET("/users/getReview", h.GetUserReviewPRs)

	// Pull Requests
	router.POST("/pullRequest/create", h.CreatePullRequest)
	router.POST("/pullRequest/merge", h.MergePullRequest)
	router.POST("/pullRequest/reassign", h.ReassignReviewer)

	return router
}

// Регистрация роутов (альтернативный метод)
func (h *Handler) RegisterRoutes(router *gin.Engine) {
	h.SetupRouter()
}
