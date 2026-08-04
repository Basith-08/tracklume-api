package dashboard

import (
	"context"
	"github.com/Basith-08/tracklume-api/internal/project"
	"github.com/google/uuid"
)

type Service struct {
	repo     *Repository
	projects *project.Service
}

func NewService(repo *Repository, projects *project.Service) *Service {
	return &Service{repo: repo, projects: projects}
}
func (s *Service) Get(ctx context.Context, userID, projectID uuid.UUID) (Summary, error) {
	if _, _, err := s.projects.Require(ctx, projectID, userID, "read"); err != nil {
		return Summary{}, err
	}
	result, err := s.repo.Build(ctx, projectID)
	if err != nil {
		return Summary{}, err
	}
	if result.ByStatus == nil {
		result.ByStatus = map[string]int{}
	}
	return result, nil
}

func CalculateProgress(done, totalNonCancelled int) float64 {
	if totalNonCancelled <= 0 {
		return 0
	}
	return float64(done) * 100 / float64(totalNonCancelled)
}
