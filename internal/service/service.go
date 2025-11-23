package service

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/zalhui/pull-request-service/internal/models"
	"github.com/zalhui/pull-request-service/internal/repository"
	"go.uber.org/zap"
)

var (
	ErrTeamExists  = errors.New("team already exists")
	ErrPRExists    = errors.New("PR already exists")
	ErrPRMerged    = errors.New("PR is merged")
	ErrNotAssigned = errors.New("reviewer not assigned")
	ErrNoCandidate = errors.New("no active replacement candidate")
	ErrNotFound    = errors.New("resource not found")
)

type Service struct {
	repo repository.Repository
	log  *zap.SugaredLogger
}

func NewService(repo repository.Repository, log *zap.SugaredLogger) *Service {
	return &Service{
		repo: repo,
		log:  log,
	}
}

func (s *Service) CreateTeam(ctx context.Context, team *models.Team) error {
	s.log.Infow("Creating team", "team_name", team.TeamName)

	exists, err := s.repo.TeamExists(ctx, team.TeamName)
	if err != nil {
		s.log.Errorw("Failed to check team exists", "team_name", team.TeamName, "error", err)
		return fmt.Errorf("check team exists %s: %w", team.TeamName, err)
	}

	if exists {
		s.log.Warnw("Team already exists", "team_name", team.TeamName)
		return ErrTeamExists
	}

	if err := s.repo.CreateTeam(ctx, team); err != nil {
		s.log.Errorw("Failed to create team", "team_name", team.TeamName, "error", err)
		return fmt.Errorf("create team %s: %w", team.TeamName, err)
	}

	s.log.Infow("Team created", "team_name", team.TeamName)
	return nil
}

func (s *Service) GetTeam(ctx context.Context, teamName string) (*models.Team, error) {
	s.log.Debugw("Getting team", "team_name", teamName)

	team, err := s.repo.GetTeam(ctx, teamName)

	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}

		s.log.Errorw("Failed to get team", "team_name", teamName, "error", err)
		return nil, fmt.Errorf("get team %s: %w", teamName, err)
	}

	s.log.Debugw("Team found", "team_name", teamName)
	return team, nil
}

func (s *Service) UpdateUserActivity(ctx context.Context, userID string, isActive bool) (*models.User, error) {
	s.log.Infow("Updating user activity", "user_id", userID, "is_active", isActive)

	user, err := s.repo.UpdateUserActivity(ctx, userID, isActive)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		s.log.Errorw("Failed to update user activity", "user_id", userID, "error", err)
		return nil, fmt.Errorf("update user %s activity to %t: %w", userID, isActive, err)
	}

	s.log.Infow("User activity updated", "user_id", userID, "is_active", isActive)
	return user, nil
}

func (s *Service) CreatePullRequest(ctx context.Context, prCreate *models.PullRequest) (*models.PullRequest, error) {
	s.log.Infow("Creating pull request",
		"pr_id", prCreate.PullRequestID,
		"author_id", prCreate.AuthorID)

	exists, err := s.repo.PullRequestExists(ctx, prCreate.PullRequestID)
	if err != nil {
		s.log.Errorw("Failed to check pr exists", "pr_id", prCreate.PullRequestID, "error", err)
		return nil, fmt.Errorf("check pr %s existence: %w", prCreate.PullRequestID, err)
	}

	if exists {
		s.log.Warnw("PR already exists", "pr_id", prCreate.PullRequestID)
		return nil, ErrPRExists
	}
	author, err := s.repo.GetUser(ctx, prCreate.AuthorID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		s.log.Errorw("Failed to get author", "author_id", prCreate.AuthorID, "error", err)
		return nil, fmt.Errorf("get author %s: %w", prCreate.AuthorID, err)
	}

	reviewers, err := s.autoAssignReviewers(ctx, author.TeamName, prCreate.AuthorID)
	if err != nil {
		s.log.Errorw("Failed to auto assign reviewers", "author_id", prCreate.AuthorID, "error", err)
		return nil, fmt.Errorf("auto-assign reviewers for PR %s: %w", prCreate.PullRequestID, err)
	}

	pr := &models.PullRequest{
		PullRequestID:     prCreate.PullRequestID,
		PullRequestName:   prCreate.PullRequestName,
		AuthorID:          prCreate.AuthorID,
		Status:            "OPEN",
		AssignedReviewers: reviewers,
	}

	if err := s.repo.CreatePullRequest(ctx, pr); err != nil {
		s.log.Errorw("Failed to create pr", "pr_id", prCreate.PullRequestID, "error", err)
		return nil, fmt.Errorf("create pr %s: %w", prCreate.PullRequestID, err)
	}

	s.log.Infow("Pull request created",
		"pr_id", prCreate.PullRequestID,
		"reviewers", pr.AssignedReviewers)

	return pr, nil
}

func (s *Service) MergePullRequest(ctx context.Context, prID string) (*models.PullRequest, error) {
	s.log.Infow("Merging pull request", "pr_id", prID)

	pr, err := s.repo.GetPullRequest(ctx, prID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		s.log.Errorw("Failed to get pr", "pr_id", prID, "error", err)
		return nil, fmt.Errorf("get pr %s: %w", prID, err)
	}

	if pr.Status == "MERGED" {
		s.log.Warnw("PR already merged", "pr_id", prID)
		return pr, nil
	}

	mergedPR, err := s.repo.UpdatePRStatus(ctx, prID, "MERGED")
	if err != nil {
		s.log.Errorw("Failed to merge pr", "pr_id", prID, "error", err)
		return nil, fmt.Errorf("merge pr %s: %w", prID, err)
	}

	s.log.Infow("Pull request merged", "pr_id", prID)
	return mergedPR, nil
}

func (s *Service) ReassignReviewer(ctx context.Context, prID string, oldReviewerID string) (*models.PullRequest, string, error) {
	s.log.Infow("ReassigningReviewer",
		"pr_id", prID,
		"old_reviewer_id", oldReviewerID)

	pr, err := s.repo.GetPullRequest(ctx, prID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, "", ErrNotFound
		}
		s.log.Errorw("Failed to get pr", "pr_id", prID, "error", err)
		return nil, "", fmt.Errorf("reassign pr %s: %w", prID, err)
	}

	if pr.Status == "MERGED" {
		s.log.Warnw("PR is already merged", "pr_id", prID)
		return nil, "", ErrPRMerged
	}

	if !slices.Contains(pr.AssignedReviewers, oldReviewerID) {
		s.log.Warnw("Old reviewer not assigned", "pr_id", prID, "old_reviewer_id", oldReviewerID)
		return nil, "", ErrNotAssigned
	}

	oldReviewer, err := s.repo.GetUser(ctx, oldReviewerID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, "", ErrNotFound
		}
		s.log.Errorw("Failed to get old reviewer", "old_reviewer_id", oldReviewerID, "error", err)
		return nil, "", fmt.Errorf("reassign pr %s: %w", prID, err)
	}

	//не могут назначаться старые ревьюеры и автор пр
	excludeIDs := append(pr.AssignedReviewers, pr.AuthorID)

	newReviewer, err := s.repo.GetRandomActiveTeamMember(ctx, oldReviewer.TeamName, excludeIDs)
	if err != nil {
		s.log.Errorw("Failed to get new reviewer", "error", err)
		return nil, "", fmt.Errorf("reassign pr %s: %w", prID, err)
	}

	if newReviewer == nil {
		s.log.Warnw("No available reviewers", "pr_id", prID)
		return nil, "", ErrNoCandidate
	}

	if err := s.repo.ReplaceReviewer(ctx, prID, oldReviewerID, newReviewer.UserID); err != nil {
		s.log.Errorw("Failed to replace reviewer", "pr_id", prID, "old_reviewer_id", oldReviewerID, "new_reviewer_id", newReviewer.UserID, "error", err)
		return nil, "", fmt.Errorf("reassign pr %s: %w", prID, err)
	}

	updatedPR, err := s.repo.GetPullRequest(ctx, prID)
	if err != nil {
		s.log.Errorw("Failed to get updated pr", "pr_id", prID, "error", err)
		return nil, "", fmt.Errorf("reassign pr %s: %w", prID, err)
	}

	s.log.Infow("Reviewer reasigned",
		"pr_id", prID,
		"old_reviewer_id", oldReviewerID,
		"new_reviewer_id", newReviewer.UserID)
	return updatedPR, newReviewer.UserID, nil
}

func (s *Service) GetUserReviewPRs(ctx context.Context, userID string) ([]models.PullRequestShort, error) {
	s.log.Debugw("Getting user review prs", "user_id", userID)

	_, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		s.log.Errorw("Failed to get user", "user_id", userID, "error", err)
		return nil, fmt.Errorf("get user %s: %w", userID, err)
	}

	prs, err := s.repo.GetUserReviewPRs(ctx, userID)
	if err != nil {
		s.log.Errorw("Failed to get user review prs", "user_id", userID, "error", err)
		return nil, fmt.Errorf("get review PRs for user %s: %w", userID, err)
	}

	s.log.Debugw("User review prs retrieved", "user_id", userID, "count", len(prs))
	return prs, nil
}

func (s *Service) autoAssignReviewers(ctx context.Context, teamName string, authorID string) ([]string, error) {
	s.log.Debugw("Auto assigning reviewers", "team_name", teamName, "author_id", authorID)

	availableMembers, err := s.repo.GetActiveTeamMembers(ctx, teamName, authorID)
	if err != nil {
		s.log.Errorw("Failed to get team members", "team_name", teamName, "error", err)
		return nil, fmt.Errorf("get active team members for team %s: %w", teamName, err)
	}

	var reviewers []string
	maxReviewers := 2

	//назначается до 2 ревьюеров
	needed := min(len(availableMembers), maxReviewers)
	for i := 0; i < needed; i++ {
		candidate, err := s.repo.GetRandomActiveTeamMember(ctx, teamName, append(reviewers, authorID))
		if err != nil {
			s.log.Errorw("Failed to get random team member", "team_name", teamName, "error", err)
			return nil, fmt.Errorf("get random team member from team %s: %w", teamName, err)
		}
		if candidate != nil {
			reviewers = append(reviewers, candidate.UserID)
		} else {
			break
		}
	}

	s.log.Debugw("Reviewers assigned", "team_name", teamName, "count", len(reviewers))
	return reviewers, nil
}

func (s *Service) GetStats(ctx context.Context) (*models.Stats, error) {
	s.log.Debugw("Getting statistics")

	stats, err := s.repo.GetStats(ctx)
	if err != nil {
		s.log.Errorw("Failed to get statistics", "error", err)
		return nil, fmt.Errorf("get statistics: %w", err)
	}

	s.log.Debugw("Statistics retrieved",
		"total_users", stats.TotalUsers,
		"total_prs", stats.TotalPRs)

	return stats, nil
}
