package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zalhui/pull-request-service/internal/models"
	"go.uber.org/zap"
)

var (
	ErrNotFound = errors.New("not found")
)

type PostgresRepository struct {
	log *zap.SugaredLogger
	db  *pgxpool.Pool
}

func NewPostgresRepository(log *zap.SugaredLogger, db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		log: log,
		db:  db,
	}
}

func (r *PostgresRepository) CreateTeam(ctx context.Context, team *models.Team) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO teams (team_name) VALUES ($1)`, team.TeamName)
	if err != nil {
		return fmt.Errorf("insert team: %w", err)
	}

	//если в команде существующие юзеры->обновление существующего
	for _, member := range team.Members {
		_, err := tx.Exec(ctx, `
			INSERT INTO users (user_id, username, team_name, is_active)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id) 
			DO UPDATE SET 
				team_name = EXCLUDED.team_name,
				is_active = EXCLUDED.is_active,
				updated_at = NOW()
		`, member.UserID, member.Username, team.TeamName, member.IsActive)
		if err != nil {
			return fmt.Errorf("upsert team member %s: %w", member.UserID, err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) GetTeam(ctx context.Context, teamName string) (*models.Team, error) {
	exists, err := r.TeamExists(ctx, teamName)
	if err != nil {
		return nil, fmt.Errorf("check team existence: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	var team models.Team
	team.TeamName = teamName

	rows, err := r.db.Query(ctx,
		`SELECT user_id, username, is_active
		FROM users 
		WHERE team_name = $1
	`, teamName)

	if err != nil {
		return nil, fmt.Errorf("query team members: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var member models.TeamMember
		err := rows.Scan(&member.UserID, &member.Username, &member.IsActive)
		if err != nil {
			return nil, fmt.Errorf("scan team member: %w", err)
		}
		team.Members = append(team.Members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return &team, nil
}

func (r *PostgresRepository) TeamExists(ctx context.Context, teamName string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)
	`, teamName).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("check team exists: %w", err)
	}
	return exists, nil
}

func (r *PostgresRepository) CreateUser(ctx context.Context, user *models.User) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO users (user_id, username, team_name, is_active) 
		VALUES ($1, $2, $3, $4)
	`, user.UserID, user.Username, user.TeamName, user.IsActive)

	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetUser(ctx context.Context, userID string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(ctx, `
		SELECT user_id, username, team_name, is_active 
		FROM users 
		WHERE user_id = $1
	`, userID).Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	return &user, nil
}

func (r *PostgresRepository) UpdateUserActivity(ctx context.Context, userID string, isActive bool) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(ctx, `
		UPDATE users 
		SET is_active = $2, updated_at = NOW()
		WHERE user_id = $1
		RETURNING user_id, username, team_name, is_active
	`, userID, isActive).Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update user activity: %w", err)
	}

	return &user, nil
}

func (r *PostgresRepository) GetActiveTeamMembers(ctx context.Context, teamName string, excludeUserID string) ([]models.User, error) {
	rows, err := r.db.Query(ctx, `
		SELECT user_id, username, team_name, is_active 
		FROM users 
		WHERE team_name = $1 AND is_active = true AND user_id != $2
	`, teamName, excludeUserID)

	if err != nil {
		return nil, fmt.Errorf("get active team members: %w", err)
	}

	defer rows.Close()

	var activeMembers []models.User
	for rows.Next() {
		var member models.User
		err := rows.Scan(&member.UserID, &member.Username, &member.TeamName, &member.IsActive)
		if err != nil {
			return nil, fmt.Errorf("scan team member: %w", err)
		}
		activeMembers = append(activeMembers, member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return activeMembers, nil
}

func (r *PostgresRepository) GetRandomActiveTeamMember(ctx context.Context, teamName string, excludeUserIDs []string) (*models.User, error) {
	var user models.User

	query := `
		SELECT user_id, username, team_name, is_active
		FROM users 
		WHERE team_name = $1 
		AND is_active = true
	`
	args := []interface{}{teamName}

	if len(excludeUserIDs) > 0 {
		query += fmt.Sprintf(" AND user_id NOT IN (%s)", placeholders(2, len(excludeUserIDs)))
		for _, id := range excludeUserIDs {
			args = append(args, id)
		}
	}

	query += " ORDER BY RANDOM() LIMIT 1"

	err := r.db.QueryRow(ctx, query, args...).Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get random team member: %w", err)
	}
	return &user, nil
}

func (r *PostgresRepository) CreatePullRequest(ctx context.Context, pr *models.PullRequest) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) 
		VALUES ($1, $2, $3, $4)
	`, pr.PullRequestID, pr.PullRequestName, pr.AuthorID, pr.Status)

	if err != nil {
		return fmt.Errorf("insert pull request: %w", err)
	}

	for _, reviewerID := range pr.AssignedReviewers {
		_, err := tx.Exec(ctx, `
			INSERT INTO pull_request_reviewers (pull_request_id, user_id)
			VALUES ($1, $2)
		`, pr.PullRequestID, reviewerID)

		if err != nil {
			return fmt.Errorf("insert reviewer %s: %w", reviewerID, err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) GetPullRequest(ctx context.Context, prID string) (*models.PullRequest, error) {
	var pr models.PullRequest
	err := r.db.QueryRow(ctx, `
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests 
		WHERE pull_request_id = $1
	`, prID).Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status, &pr.CreatedAt, &pr.MergedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get pull requests: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT user_id 
		FROM pull_request_reviewers 
		WHERE pull_request_id = $1
	`, prID)

	if err != nil {
		return nil, fmt.Errorf("get pr reviewers: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var reviewerID string
		if err := rows.Scan(&reviewerID); err != nil {
			return nil, fmt.Errorf("scan pr reviewers: %w", err)
		}
		pr.AssignedReviewers = append(pr.AssignedReviewers, reviewerID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return &pr, nil
}

func (r *PostgresRepository) PullRequestExists(ctx context.Context, prID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)
	`, prID).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("check pr exists: %w", err)
	}
	return exists, nil
}

func (r *PostgresRepository) UpdatePRStatus(ctx context.Context, prID string, status string) (*models.PullRequest, error) {
	var pr models.PullRequest
	err := r.db.QueryRow(ctx, `
		UPDATE pull_requests 
		SET status = $1, 
		    merged_at = CASE WHEN $1 = 'MERGED' AND merged_at IS NULL THEN NOW() ELSE merged_at END,
		    updated_at = NOW()
		WHERE pull_request_id = $2
		RETURNING pull_request_id, pull_request_name, author_id, status, created_at, merged_at
	`, status, prID).Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID,
		&pr.Status, &pr.CreatedAt, &pr.MergedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update pr status: %w", err)
	}

	return &pr, nil
}

func (r *PostgresRepository) AssignReviewers(ctx context.Context, prID string, reviewerIDs []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		DELETE FROM pull_request_reviewers 
		WHERE pull_request_id = $1
	`, prID)

	if err != nil {
		return fmt.Errorf("delete reviewers: %w", err)
	}

	for _, reviewerID := range reviewerIDs {
		_, err := tx.Exec(ctx, `
			INSERT INTO pull_request_reviewers (pull_request_id, user_id)
			VALUES ($1, $2)
		`, prID, reviewerID)

		if err != nil {
			return fmt.Errorf("insert reviewer %s: %w", reviewerID, err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) ReplaceReviewer(ctx context.Context, prID string, oldReviewerID string, newReviewerID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE pull_request_reviewers
		SET user_id = $1
		WHERE pull_request_id = $2 AND user_id = $3
	`, newReviewerID, prID, oldReviewerID)
	if err != nil {
		return fmt.Errorf("replace reviewer: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetUserReviewPRs(ctx context.Context, userID string) ([]models.PullRequestShort, error) {
	rows, err := r.db.Query(ctx, `
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
		FROM pull_requests pr
		JOIN pull_request_reviewers prr ON pr.pull_request_id = prr.pull_request_id
		WHERE prr.user_id = $1
		ORDER BY pr.created_at DESC
	`, userID)

	if err != nil {
		return nil, fmt.Errorf("get user review prs: %w", err)
	}

	defer rows.Close()

	var prs []models.PullRequestShort

	for rows.Next() {
		var pr models.PullRequestShort
		if err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status); err != nil {
			return nil, fmt.Errorf("scan pr: %w", err)
		}
		prs = append(prs, pr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return prs, nil
}

func placeholders(start int, count int) string {
	if count <= 0 {
		return ""
	}
	result := fmt.Sprintf("$%d", start)
	for i := 1; i < count; i++ {
		result += fmt.Sprintf(", $%d", start+i)
	}
	return result
}
