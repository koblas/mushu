package action

import (
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strings"
)

const (

	// StatusFailed is returned by Status() in case this action has been marked as failed
	StatusFailed = 1
	// StatusSuccess is returned by Status() in case this action has not been marked as failed. By default an action is claimed as successful
	StatusSuccess = 0

	prefix = "_goact_"
)

var (
	// this has a random component add at init
	delimiter = prefix + "file_command_" + randomString(16)
	stopToken = prefix + "stop_commands_" + randomString(16)

	logger = slog.New(NewSlogHandler())
)

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))] // #nosec G404
	}

	return string(b)
}

func multilineOutput(name, value string) string {
	builder := strings.Builder{}

	_, _ = builder.WriteString(name)
	_, _ = builder.WriteString("<<")
	_, _ = builder.WriteString(delimiter)
	_, _ = builder.WriteString(EOL)
	_, _ = builder.WriteString(value)
	_, _ = builder.WriteString(EOL)
	_, _ = builder.WriteString(delimiter)
	_, _ = builder.WriteString(EOL)

	return builder.String()
}

// ExportVariable sets the environment variable name (for this action and future actions)
func ExportVariable(name, value string) {
	if err := issueFileCommand(GitHubExportEnvFilePathEnvName, multilineOutput(name, value)); err != nil {
		IssueCommand("set-env", map[string]string{"name": name}, value)
	}

	_ = os.Setenv(name, value)
}

// SetSecret registers a secret which will get masked from logs
func SetSecret(secret string) {
	Issue("add-mask", secret)
}

// AddPath prepends inputPath to the PATH (for this action and future actions)
func AddPath(path string) {
	if err := issueFileCommand(GitHubPathFilePathEnvName, path); err != nil {
		Issue("add-path", path)
	}
	// TODO js: process.env['PATH'] = `${inputPath}${path.delimiter}${process.env['PATH']}`
}

// GetBoolInput gets the value of an input and returns whether it equals "true".
// In any other case, whether it does not equal, or the input is not set, false is returned
func GetBoolInput(name string) bool {
	return strings.ToLower(GetInputOrDefault(name, "false")) == "true"
}

// GetInput gets the value of an input.  The value is also trimmed.
func GetInput(name string) (string, bool) {
	if val, ok := os.LookupEnv(strings.ToUpper("INPUT_" + strings.ReplaceAll(name, " ", "_"))); ok {
		return strings.TrimSpace(val), true
	}

	logger.Debug("Did not find the input name=" + name)

	return "", false
}

// GetInputOrDefault gets the value of an input. If value is not found, a default value is used
func GetInputOrDefault(name, dflt string) string {
	if val, ok := GetInput(name); ok {
		return val
	}

	return dflt
}

// SetOutput sets the value of an output for future actions
func SetOutput(name, value string) {
	if err := issueFileCommand(GitHubOutputFilePathEnvName, multilineOutput(name, value)); err != nil {
		logger.Info(fmt.Sprintf("did not find output file from environment variable %s, falling back to the deprecated command implementation", GitHubOutputFilePathEnvName))

		IssueCommand("set-output", map[string]string{"name": name}, value)
	}
}

// SetFailedf sets the action status to failed and sets an error message
func SetFailedf(format string, args ...any) {
	SetFailed(fmt.Sprintf(format, args...))
}

// SetFailed sets the action status to failed and sets an error message
func SetFailed(message string) {
	logger.Error(message)
}

// StartGroup begin an output group. Output until the next `GroupEnd` will be foldable in this group
func StartGroup(name string) {
	Issue("group", name)
}

// EndGroup end an output group and folds it
func EndGroup() {
	Issue("endgroup")
}

// Group wrap an asynchronous function call in a group, all logs of the function will be collapsed after completion
func Group(name string, f func()) {
	StartGroup(name)
	defer EndGroup()
	f()
}

// StopCommands Stops processing any workflow commands.
// Commands will be resumed when calling StartCommands(endToken)
// This special command allows you to log anything without accidentally running a workflow command.
// For example, you could stop logging to output an entire script that has comments.
func StopCommands(endToken string) {
	if endToken == "" {
		endToken = stopToken
	}

	Issue("stop-commands", endToken)
}

// StartCommands enables commands stopped until the endToken
func StartCommands(endToken string) {
	if endToken == "" {
		endToken = stopToken
	}

	Issue(endToken)
}

// WithoutCommands executes the functions ensuring it does not execute any github actions commands.
// This special command allows you to log anything without accidentally running a workflow command.
// For example, you could stop logging to output an entire script that has comments.
func WithoutCommands(f func()) {
	endToken := prefix + "_without_" + randomString(16)

	StopCommands(endToken)
	defer StartCommands(endToken)
	f()
}

// SaveState saves state for current action, the state can only be retrieved by this action's post job execution.
func SaveState(name, value string) {
	if err := issueFileCommand(GitHubStateFilePathEnvName, multilineOutput(name, value)); err != nil {
		logger.Info(fmt.Sprintf("did not find state file from environment variable %s, falling back to the deprecated command implementation", GitHubStateFilePathEnvName))
		IssueCommand("save-state", map[string]string{"name": name}, value)
	}
}

// GetState gets the value of an state set by this action's main execution.
func GetState(name string) string {
	return os.Getenv("STATE_" + name)
}

// IsDebug returns whether the github actions is currently under debug
func IsDebug() bool {
	value, found := os.LookupEnv("RUNNER_DEBUG")

	return found && value == "1"
}

// AddStepSummary adds some custom Markdown for each job so that it will be displayed on the summary
// page of a workflow run. You can use job summaries to display and group unique content, such as test
// result summaries, so that someone viewing the result of a workflow run doesn't need to go into the
// logs to see important information related to the run, such as failures.
// see: https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#adding-a-job-summary

func AddStepSummary(summary string) error {
	// os.O_CREATE: If pathname does not exist, create it as a regular file.
	err := issueFileCommandWithPerm(GitHubSummaryPathEnvName, summary, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("add step summary: %w", err)
	}

	return nil
}

// ReplaceStepSummary clear all content for the current step
// see: https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#overwriting-job-summaries
func ReplaceStepSummary(summary string) error {
	// os.O_CREATE: If pathname does not exist, create it as a regular file.
	err := issueFileCommandWithPerm(GitHubSummaryPathEnvName, summary, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("replace step summary: %w", err)
	}

	return nil
}

// DeleteStepSummary completely remove a summary for the current step
// see: https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#removing-job-summaries
func DeleteStepSummary() error {
	path, ok := os.LookupEnv(GitHubSummaryPathEnvName)
	if ok {
		err := os.Remove(path)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("delete step summary: %w", err)
		}
	} else {
		logger.Info("unable to find command file " + GitHubSummaryPathEnvName)
	}

	return nil
}
