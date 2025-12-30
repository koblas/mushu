package action

import (
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	stdout   io.Writer = os.Stdout
	replacer           = strings.NewReplacer(
		"%", "%25",
		"\r", "%0D",
		"\n", "%0A",
		":", "%3A",
		",", "%2C",
	)
)

func SetStdout(w io.Writer) {
	stdout = w
}

// Issue displays a plain typed message following github actions interface
func Issue(kind string, message ...string) {
	IssueCommand(kind, nil, strings.Join(message, ""))
}

// IssueCommand displays a typed message with properties following github actions interface.
// see https://github.com/actions/toolkit/blob/e69833ed16500afaa7d137a9cf6da76fb8fb54da/packages/core/src/command.ts#L19
func IssueCommand(kind string, properties map[string]string, message string) {
	c := &command{kind, properties, message}
	_, _ = stdout.Write([]byte(c.String()))
	_, _ = stdout.Write([]byte(EOL))
}

// issueFileCommand implements stores the command in a file
// see https://github.com/actions/toolkit/pull/571/files#diff-9ce6eb99f5fb5529e795254801e03ae56d67d3d5fcbec635f91e9a8a61ad8b64R10
func issueFileCommandWithPerm(command string, message string, flag int, perm os.FileMode) error {
	if path, ok := os.LookupEnv(command); ok {
		fd, err := os.OpenFile(path, flag, perm) // #nosec G304 -- controlled by environment
		if err != nil {
			return fmt.Errorf("opening command file %s: %w", path, err)
		}
		defer func() { _ = fd.Close() }()
		if _, err = fd.WriteString(message); err != nil {
			return fmt.Errorf("writing message: %s: %w", path, err)
		}
		if _, err = fd.WriteString("\n"); err != nil {
			return fmt.Errorf("writing newline: %w", err)
		}

		return nil
	}

	return fmt.Errorf("unable to find command file %s", command)
}

// issueFileCommand implements stores the command in a file
// see https://github.com/actions/toolkit/pull/571/files#diff-9ce6eb99f5fb5529e795254801e03ae56d67d3d5fcbec635f91e9a8a61ad8b64R10
func issueFileCommand(command string, message string) error {
	err := issueFileCommandWithPerm(command, message, os.O_APPEND|os.O_RDWR, 0)
	if err != nil {
		return err
	}

	return nil
}

type command struct {
	command    string
	properties map[string]string
	message    string
}

func (c *command) String() string {
	const cmdString = "::"

	build := strings.Builder{}

	_, _ = build.WriteString(cmdString)
	_, _ = build.WriteString(c.command)

	sep := " "
	for key, value := range c.properties {
		_, _ = build.WriteString(sep)
		sep = ","

		_, _ = build.WriteString(key)
		_, _ = build.WriteString("=")
		_, _ = build.WriteString(replacer.Replace(value))
	}

	_, _ = build.WriteString(cmdString)
	_, _ = build.WriteString(replacer.Replace(c.message))

	return build.String()
}
