package action

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	gogithub "github.com/google/go-github/v79/github"
)

type ActionRepo struct {
	Owner string
	Repo  string
}

type ActionIssue struct {
	ActionRepo
	Number int
}

// ActionContext contains details on the workflow execution
type ActionContext struct {
	Payload    any
	EventName  string
	SHA        string
	Ref        string
	Workflow   string
	Action     string
	Actor      string
	RunAttempt int
	RunNumber  int
	RunId      int
	ApiUrl     string
	ServerUrl  string
	GraphqlUrl string
	Issue      ActionIssue
	Repo       ActionRepo

	// Added for convenience
	OutputFilePath    string
	StateFilePath     string
	ExportEnvFilePath string
}

var errNoRepoInEvent = errors.New("no repository information in event payload")

func repoFromEvent(event any) (ActionRepo, error) {
	var r *gogithub.Repository
	// try to get repo from payload
	switch e := event.(type) {
	case *gogithub.BranchProtectionConfigurationEvent:
		r = e.GetRepo()
	case *gogithub.BranchProtectionRuleEvent:
		r = e.GetRepo()
	case *gogithub.CheckRunEvent:
		r = e.GetRepo()
	case *gogithub.CheckSuiteEvent:
		r = e.GetRepo()
	case *gogithub.CodeScanningAlertEvent:
		r = e.GetRepo()
	case *gogithub.CommitCommentEvent:
		r = e.GetRepo()
	case *gogithub.ContentReferenceEvent:
		r = e.GetRepo()
	case *gogithub.CreateEvent:
		r = e.GetRepo()
	case *gogithub.CustomPropertyEvent:
		return ActionRepo{}, fmt.Errorf("%T: %w", e, errNoRepoInEvent)
	case *gogithub.CustomPropertyValuesEvent:
		r = e.GetRepo()
	case *gogithub.DeleteEvent:
		r = e.GetRepo()
	case *gogithub.DependabotAlertEvent:
		r = e.GetRepo()
	case *gogithub.DeployKeyEvent:
		r = e.GetRepo()
	case *gogithub.DeploymentEvent:
		r = e.GetRepo()
	case *gogithub.DeploymentReviewEvent:
		r = e.GetRepo()
	case *gogithub.DeploymentStatusEvent:
		r = e.GetRepo()
	case *gogithub.DeploymentProtectionRuleEvent:
		r = e.GetRepo()
	case *gogithub.DiscussionEvent:
		r = e.GetRepo()
	case *gogithub.DiscussionCommentEvent:
		r = e.GetRepo()
	case *gogithub.ForkEvent:
		r = e.GetRepo()
	case *gogithub.GitHubAppAuthorizationEvent:
		return ActionRepo{}, fmt.Errorf("%T: %w", e, errNoRepoInEvent)
	case *gogithub.GollumEvent:
		r = e.GetRepo()
	case *gogithub.InstallationEvent:
		return ActionRepo{}, fmt.Errorf("%T: %w", e, errNoRepoInEvent)
	case *gogithub.InstallationRepositoriesEvent:
		return ActionRepo{}, fmt.Errorf("%T: %w", e, errNoRepoInEvent)
	case *gogithub.InstallationTargetEvent:
		r = e.GetRepository()
	case *gogithub.IssueCommentEvent:
		r = e.GetRepo()
	case *gogithub.IssuesEvent:
		r = e.GetRepo()
	case *gogithub.LabelEvent:
		r = e.GetRepo()
	case *gogithub.MarketplacePurchaseEvent:
		return ActionRepo{}, fmt.Errorf("%T: %w", e, errNoRepoInEvent)
	case *gogithub.MemberEvent:
		r = e.GetRepo()
	case *gogithub.MembershipEvent:
		return ActionRepo{}, fmt.Errorf("%T: %w", e, errNoRepoInEvent)
	case *gogithub.MergeGroupEvent:
		r = e.GetRepo()
	case *gogithub.MetaEvent:
		r = e.GetRepo()
	case *gogithub.MilestoneEvent:
		r = e.GetRepo()
	case *gogithub.OrganizationEvent:
		return ActionRepo{}, fmt.Errorf("%T: %w", e, errNoRepoInEvent)
	case *gogithub.OrgBlockEvent:
		return ActionRepo{}, fmt.Errorf("%T: %w", e, errNoRepoInEvent)
	case *gogithub.PackageEvent:
		r = e.GetRepo()
	case *gogithub.PageBuildEvent:
		r = e.GetRepo()
	case *gogithub.PersonalAccessTokenRequestEvent:
		return ActionRepo{}, fmt.Errorf("%T: %w", e, errNoRepoInEvent)
	case *gogithub.PingEvent:
		r = e.GetRepo()
	case *gogithub.ProjectV2Event:
		return ActionRepo{}, fmt.Errorf("%T: %w", e, errNoRepoInEvent)
	case *gogithub.ProjectV2ItemEvent:
		return ActionRepo{}, fmt.Errorf("%T: %w", e, errNoRepoInEvent)
	case *gogithub.PublicEvent:
		r = e.GetRepo()
	case *gogithub.PullRequestEvent:
		r = e.GetRepo()
	case *gogithub.PullRequestReviewEvent:
		r = e.GetRepo()
	case *gogithub.PullRequestReviewCommentEvent:
		r = e.GetRepo()
	case *gogithub.PullRequestReviewThreadEvent:
		r = e.GetRepo()
	case *gogithub.PullRequestTargetEvent:
		r = e.GetRepo()
	case *gogithub.PushEvent:
		r = &gogithub.Repository{
			Owner: e.GetRepo().GetOwner(),
			Name:  e.GetRepo().Name,
		}
	case *gogithub.RegistryPackageEvent:
		return ActionRepo{}, fmt.Errorf("%T: %w", e, errNoRepoInEvent)
	case *gogithub.RepositoryEvent:
		r = e.GetRepo()
	case *gogithub.RepositoryDispatchEvent:
		r = e.GetRepo()
	case *gogithub.RepositoryImportEvent:
		r = e.GetRepo()
	case *gogithub.RepositoryRulesetEvent:
		r = e.GetRepository()
	case *gogithub.RepositoryVulnerabilityAlertEvent:
		r = e.GetRepository()
	case *gogithub.ReleaseEvent:
		r = e.GetRepo()
	case *gogithub.SecretScanningAlertEvent:
		r = e.GetRepo()
	case *gogithub.SecretScanningAlertLocationEvent:
		r = e.GetRepo()
	case *gogithub.SecurityAdvisoryEvent:
		r = e.GetRepository()
	case *gogithub.SecurityAndAnalysisEvent:
		r = e.GetRepository()
	case *gogithub.SponsorshipEvent:
		r = e.GetRepository()
	case *gogithub.StatusEvent:
		r = e.GetRepo()
	case *gogithub.TeamEvent:
		r = e.GetRepo()
	case *gogithub.TeamAddEvent:
		r = e.GetRepo()
	case *gogithub.UserEvent:
		return ActionRepo{}, fmt.Errorf("%T: %w", e, errNoRepoInEvent)
	case *gogithub.WatchEvent:
		r = e.GetRepo()
	case *gogithub.WorkflowDispatchEvent:
		r = e.GetRepo()
	case *gogithub.WorkflowJobEvent:
		r = e.GetRepo()
	case *gogithub.WorkflowRunEvent:
		r = e.GetRepo()
	default:
		return ActionRepo{}, fmt.Errorf("unsupported event type %T", e)
	}

	return ActionRepo{
		Owner: r.GetOwner().GetLogin(),
		Repo:  r.GetName(),
	}, nil
}

func issueFromEvent(event any, repo ActionRepo) ActionIssue {
	var number int

	switch e := event.(type) {
	case *gogithub.MilestoneEvent:
		number = e.GetMilestone().GetNumber()
	case *gogithub.IssueEvent:
		number = e.GetIssue().GetNumber()
	case *gogithub.PullRequestEvent:
		number = e.GetNumber()
	}

	return ActionIssue{
		ActionRepo: repo,
		Number:     number,
	}
}

// ParseActionEnv parses the environemnt and extracts the ActionContext on demand. For example in tests
func ParseActionEnv() (ActionContext, error) {
	repo := ActionRepo{}
	repoSet := false

	if val, ok := os.LookupEnv("GITHUB_REPOSITORY"); ok {
		r := strings.SplitN(val, "/", 2)
		if len(r) != 2 {
			return ActionContext{}, errors.New("gogithub_REPOSITORY_OWNER is malformed")
		}
		repo.Owner = r[0]
		repo.Repo = r[1]
		repoSet = true
	}

	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return ActionContext{}, errors.New("gogithub_EVENT_PATH is not set")

	}
	payload, err := os.ReadFile(eventPath)
	if err != nil {
		return ActionContext{}, fmt.Errorf("could not read gogithub_EVENT_PATH=%q file: %w", eventPath, err)
	}

	eventName := os.Getenv("GITHUB_EVENT_NAME")
	hook, err := gogithub.ParseWebHook(eventName, payload)
	if err != nil {
		return ActionContext{}, fmt.Errorf("could not parse gogithub webhook payload for event %q: %w", eventName, err)
	}

	intEnv := func(varName string) int {
		valStr := os.Getenv(varName)

		val, err := strconv.Atoi(valStr)
		if err != nil {
			return 0
		}

		return val
	}

	defEnv := func(val, def string) string {
		val, ok := os.LookupEnv(val)
		if !ok {
			return def
		}
		return val
	}

	if !repoSet {
		repo, err = repoFromEvent(hook)

		if !errors.Is(err, errNoRepoInEvent) {
			return ActionContext{}, fmt.Errorf("could not extract repository from event payload: %w", err)
		}
	}

	ctx := ActionContext{
		EventName:  eventName,
		SHA:        os.Getenv("GITHUB_SHA"),
		Ref:        os.Getenv("GITHUB_REF"),
		Workflow:   os.Getenv("GITHUB_WORKFLOW"),
		Action:     os.Getenv("GITHUB_ACTION"),
		Actor:      os.Getenv("GITHUB_ACTOR"),
		RunAttempt: intEnv("GITHUB_RUN_ATTEMPT"),
		RunNumber:  intEnv("GITHUB_RUN_NUMBER"),
		RunId:      intEnv("GITHUB_RUN_ID"),
		ApiUrl:     defEnv(os.Getenv("GITHUB_API_URL"), "https://api.gogithub.com"),
		ServerUrl:  defEnv(os.Getenv("GITHUB_SERVER_URL"), "https://gogithub.com"),
		GraphqlUrl: defEnv(os.Getenv("GITHUB_GRAPHQL_URL"), "https://api.gogithub.com/graphql"),

		OutputFilePath:    os.Getenv(GitHubOutputFilePathEnvName),
		StateFilePath:     os.Getenv(GitHubStateFilePathEnvName),
		ExportEnvFilePath: os.Getenv(GitHubExportEnvFilePathEnvName),
		Repo:              repo,
		Payload:           hook,
		Issue:             issueFromEvent(hook, repo),
	}

	return ctx, nil
}
