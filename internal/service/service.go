package service

import (
	"context"

	"github.com/zalhui/pull-request-service/internal/models"
	"github.com/zalhui/pull-request-service/internal/repository"
	"go.uber.org/zap"
)

type Service struct {
	repo repository.Repository
	log  *zap.SugaredLogger
}

func (s *Service) CreateTeam(ctx context.Context, team *models.Team) error {
	panic("not implemented") // TODO: Implement
}

func (s *Service) GetTeam(ctx context.Context, teamName string) (*models.Team, error) {
	panic("not implemented") // TODO: Implement
}

func (s *Service) UpdateUserActivity(ctx context.Context, userID string, isActive bool) (*models.User, error) {
	panic("not implemented") // TODO: Implement
}

// Уникальные методы сервиса - их НЕТ в репозитории
func (s *Service) CreatePullRequest(ctx context.Context, prCreate *models.PullRequest) (*models.PullRequest, error) {
	panic("not implemented") // TODO: Implement
}

func (s *Service) MergePullRequest(ctx context.Context, prID string) (*models.PullRequest, error) {
	panic("not implemented") // TODO: Implement
}

func (s *Service) ReassignReviewer(ctx context.Context, prID string, oldReviewerID string) (*models.PullRequest, string, error) {
	panic("not implemented") // TODO: Implement
}

func (s *Service) GetUserReviewPRs(ctx context.Context, userID string) ([]models.PullRequestShort, error) {
	panic("not implemented") // TODO: Implement
}

// Приватные бизнес-методы
func (s *Service) autoAssignReviewers(ctx context.Context, teamName string, authorID string) ([]string, error) {
	panic("not implemented") // TODO: Implement
}
