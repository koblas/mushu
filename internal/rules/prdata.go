package rules

import "context"

// PRData represents pull request data for rule evaluation
type PRData interface {
	GetAuthor() string
	GetSourceBranch() string
	GetTargetBranch() string
	GetFiles() []PRFile
}

// TeamLookup provides team membership information
type TeamLookup interface {
	GetUserTeams(ctx context.Context, username string) ([]string, error)
}
