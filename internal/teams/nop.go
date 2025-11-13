package teams

import "context"

type NoOpTeamService struct{}

func NewNoOpTeamService() *NoOpTeamService {
	return &NoOpTeamService{}
}

func (s *NoOpTeamService) GetUserRoles(ctx context.Context, username string) ([]string, error) {
	return []string{}, nil
}

func (s *NoOpTeamService) GetTeamMembers(ctx context.Context, username string) ([]string, error) {
	return []string{}, nil
}

func (s *NoOpTeamService) GetUserTeams(ctx context.Context, username string) ([]string, error) {
	return []string{}, nil
}
