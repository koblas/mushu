package github

type Repo interface {
	RepoOwner() string
	RepoName() string
	RepoHost() string
}

type GitHubRepo struct {
	owner string
	name  string
	host  string
}

func NewWithHost(owner, name, host string) *GitHubRepo {
	if host == "" {
		host = "github.com"
	}

	return &GitHubRepo{
		owner: owner,
		name:  name,
		host:  host,
	}
}

func (r *GitHubRepo) RepoOwner() string {
	return r.owner
}

func (r *GitHubRepo) RepoName() string {
	return r.name
}

func (r *GitHubRepo) RepoHost() string {
	return r.host
}
