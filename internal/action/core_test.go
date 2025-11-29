package action

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestSummary(t *testing.T) {
	t.Setenv("GITHUB_STEP_SUMMARY", t.TempDir()+"/summary.md")

	t.Run("integrated with Github Actions (it should appear on the run)", func(t *testing.T) {
		err := AddStepSummary("and a new line")
		require.NoError(t, err)
		err = DeleteStepSummary()
		require.NoError(t, err)
		err = AddStepSummary("this content should be overridden")
		require.NoError(t, err)
		err = ReplaceStepSummary("this is the new content")
		require.NoError(t, err)
		err = AddStepSummary("and a new line")
		require.NoError(t, err)
	})
	t.Run("with content check", func(t *testing.T) {
		fd, err := os.CreateTemp("", "summary")
		require.NoError(t, err)
		name := fd.Name()
		t.Cleanup(func() {
			_ = fd.Close()
			_ = os.Remove(name)
		})
		t.Setenv(GitHubSummaryPathEnvName, name)

		err = AddStepSummary("and a new line")
		require.NoError(t, err)
		err = DeleteStepSummary()
		require.NoError(t, err)
		err = AddStepSummary("this content should be overridden")
		require.NoError(t, err)
		err = ReplaceStepSummary("this is the new content")
		require.NoError(t, err)
		err = AddStepSummary("and a new line")
		require.NoError(t, err)
		content, err := os.ReadFile(name)
		require.NoError(t, err)
		assert.Equal(t, "this is the new content\nand a new line\n", string(content))
	})
}

func TestStopCommand(t *testing.T) {
	defer func() {
		stdout = os.Stdout
	}()
	WithoutCommands(func() {
		logger.Error("this should not make the test to fail")
	})
	out := bytes.Buffer{}
	stdout = &out

	t.Run("stop command is written on stdout (test written for unix only)", func(t *testing.T) {
		logger := slog.New(NewSlogHandler(WithOutput(&out)))
		if runtime.GOOS == "windows" {
			t.Skip("This test only runs on unix with \\n line separator")
		}
		called := false
		WithoutCommands(func() {
			called = true
			logger.Error("en-error")
		})
		assert.True(t, called)

		value := out.String()

		assert.Contains(t, value, "::stop-commands::")
		assert.Contains(t, value, "::error::en-error\n")
	})
}

func TestFormatOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("This test only runs on unix with \\n line separator")
	}

	val := multilineOutput("my-name", "my-value")

	val = strings.ReplaceAll(val, delimiter, "DELIM")

	assert.Equal(t, "my-name<<DELIM\nmy-value\nDELIM\n", val)
}

func TestOutputTasks(t *testing.T) {
	if _, ok := os.LookupEnv("ACTIONS_OUTPUT_SET"); ok {
		// state is only available in pre and post actions:
		// https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#sending-values-to-the-pre-and-post-actions
		// assert.Equal(t, "my-state-value", GetState("my-state"))
		assert.Equal(t, "my-output-value", os.Getenv("my_output"))
		assert.Equal(t, "my-env-value", os.Getenv("my_env"))
	}
	SaveState("my-state", "my-state-value")
	ExportVariable("my_env", "my-env-value")
	SetOutput("my-output", "my-output-value")
}

func TestGetInput(t *testing.T) {
	t.Run("when environment variable is not net", func(t *testing.T) {
		v, ok := GetInput("some-input with-space")
		assert.False(t, ok)
		assert.Equal(t, "", v)
	})
	t.Run("when environment variable is not net", func(t *testing.T) {
		t.Setenv("INPUT_SOME-INPUT_WITH-SPACE", " some value that needs to be Trimmed \n")
		v, ok := GetInput("some-input with-space")
		assert.True(t, ok)
		assert.Equal(t, "some value that needs to be Trimmed", v)
	})
}

func TestInputDefault(t *testing.T) {
	t.Run("when environment variable is not net", func(t *testing.T) {
		v := GetInputOrDefault("some-input with-space", " default value not trimmed ")
		assert.Equal(t, " default value not trimmed ", v)
	})
	t.Run("when environment variable is not net", func(t *testing.T) {
		t.Setenv("INPUT_SOME-INPUT_WITH-SPACE", " some value that needs to be Trimmed \n")
		v := GetInputOrDefault("some-input with-space", "some default not used")
		assert.Equal(t, "some value that needs to be Trimmed", v)
	})
}

func TestBoolInput(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"", false},
		{"false", false},
		{"FALSE", false},
		{"FaLsE", false},
		{"true", true},
		{"TRUE", true},
		{"TrUe", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("run %q", tt.value), func(t *testing.T) {
			if tt.value != "" {
				t.Setenv("INPUT_BOOL-TEST", tt.value)
			}

			result := GetBoolInput("bool-test")
			assert.Equal(t, tt.expected, result)
		})
	}
}
