package dto

import "github.com/zalhui/pull-request-service/internal/models"

type ReassignReviewerResponse struct {
	PR         *models.PullRequest `json:"pr"`
	ReplacedBy string              `json:"replaced_by"`
}
