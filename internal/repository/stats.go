package repository

import (
	"context"
	"fmt"

	"github.com/zalhui/pull-request-service/internal/models"
)

func (r *PostgresRepository) GetStats(ctx context.Context) (*models.Stats, error) {
	var stats models.Stats

	//статистика по пользователям
	err := r.db.QueryRow(ctx, `
        SELECT 
            COUNT(*) as total_users,
            COUNT(*) FILTER (WHERE is_active = true) as active_users,
            COUNT(DISTINCT team_name) as total_teams
        FROM users
    `).Scan(&stats.TotalUsers, &stats.ActiveUsers, &stats.TotalTeams)
	if err != nil {
		return nil, fmt.Errorf("get user stats: %w", err)
	}

	//статистика по пулреквестам
	err = r.db.QueryRow(ctx, `
        SELECT 
            COUNT(*) as total_prs,
            COUNT(*) FILTER (WHERE status = 'OPEN') as open_prs,
            COUNT(*) FILTER (WHERE status = 'MERGED') as merged_prs
        FROM pull_requests
    `).Scan(&stats.TotalPRs, &stats.OpenPRs, &stats.MergedPRs)
	if err != nil {
		return nil, fmt.Errorf("get pr stats: %w", err)
	}

	// топ 5 ревьюверов
	rows, err := r.db.Query(ctx, `
        SELECT u.user_id, u.username, COUNT(prr.pull_request_id) as review_count
        FROM users u
        LEFT JOIN pull_request_reviewers prr ON u.user_id = prr.user_id
        GROUP BY u.user_id, u.username
        ORDER BY review_count DESC
        LIMIT 5
    `)
	if err != nil {
		return nil, fmt.Errorf("get top reviewers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var reviewer models.TopReviewer
		if err := rows.Scan(&reviewer.UserID, &reviewer.Username, &reviewer.ReviewCount); err != nil {
			return nil, fmt.Errorf("scan top reviewer: %w", err)
		}
		stats.TopReviewers = append(stats.TopReviewers, reviewer)
	}

	return &stats, nil
}
