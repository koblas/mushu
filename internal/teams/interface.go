package teams

import "context"

type Lookup interface {
	// GetUserTeams returns the list of teams a user belongs to
	GetUserTeams(ctx context.Context, username string) ([]string, error)
}
