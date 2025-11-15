package github

import "github.com/koblas/mushu/internal/rules"

// PRData represents pull request data
type PRData struct {
	Number            int
	Title             string
	State             string
	Author            string
	SourceBranch      string
	TargetBranch      string
	Approvals         int
	Reviewers         []string
	Labels            []string
	RequiredReviewers []string
	Additions         int
	Deletions         int
	ChangedFiles      int
	Files             []rules.PRFile
	Reviews           []Review
}

// Review represents a PR review
type Review struct {
	Reviewer      string   `json:"reviewer"`
	ReviewerTeams []string `json:"reviewer_teams"`
	State         string   `json:"state"`
	SubmittedAt   string   `json:"submitted_at"`
}

// GetAuthor returns the PR author
func (p *PRData) GetAuthor() string {
	return p.Author
}

// GetSourceBranch returns the source branch
func (p *PRData) GetSourceBranch() string {
	return p.SourceBranch
}

// GetTargetBranch returns the target branch
func (p *PRData) GetTargetBranch() string {
	return p.TargetBranch
}

// GetFiles returns the PR files
func (p *PRData) GetFiles() []rules.PRFile {
	return p.Files
}
