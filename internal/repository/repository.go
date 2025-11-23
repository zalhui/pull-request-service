package repository

import (
	"context"

	"github.com/zalhui/pull-request-service/internal/models"
)

type Repository interface {
	CreateTeam(ctx context.Context, team *models.Team) error
	GetTeam(ctx context.Context, teamName string) (*models.Team, error)
	TeamExists(ctx context.Context, teamName string) (bool, error)

	CreateUser(ctx context.Context, user *models.User) error
	GetUser(ctx context.Context, userID string) (*models.User, error)
	UpdateUserActivity(ctx context.Context, userID string, isActive bool) (*models.User, error)
	GetActiveTeamMembers(ctx context.Context, teamName string, excludeUserID string) ([]models.User, error)
	GetRandomActiveTeamMember(ctx context.Context, teamName string, excludeUserIDs []string) (*models.User, error)

	CreatePullRequest(ctx context.Context, pr *models.PullRequest) error
	GetPullRequest(ctx context.Context, prID string) (*models.PullRequest, error)
	PullRequestExists(ctx context.Context, prID string) (bool, error)
	UpdatePRStatus(ctx context.Context, prID string, status string) (*models.PullRequest, error)
	AssignReviewers(ctx context.Context, prID string, reviewerIDs []string) error
	ReplaceReviewer(ctx context.Context, prID string, oldReviewerID string, newReviewerID string) error
	GetUserReviewPRs(ctx context.Context, userID string) ([]models.PullRequestShort, error)

	GetStats(ctx context.Context) (*models.Stats, error)
}
