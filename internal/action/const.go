package action

import "strings"

const (
	GitHubOutputFilePathEnvName    = "GITHUB_OUTPUT"
	GitHubStateFilePathEnvName     = "GITHUB_STATE"
	GitHubExportEnvFilePathEnvName = "GITHUB_ENV"
	GitHubPathFilePathEnvName      = "GITHUB_PATH"
	GitHubSummaryPathEnvName       = "GITHUB_STEP_SUMMARY"

	ActionsGoJsonInputEnvName = "ACTION_GO_INPUTS"
)

var commandReplacer = strings.NewReplacer(
	"%", "%25",
	"\r", "%0D",
	"\n", "%0A",
	":", "%3A",
	",", "%2C",
)
