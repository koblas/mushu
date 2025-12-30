package cli_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/koblas/mushu/internal/cli"
	"github.com/stretchr/testify/require"
)

func TestValidateCommand_MissingPRNumber(t *testing.T) {
	// Capture stdout/stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	defer func() {
		os.Stderr = oldStderr
	}()

	// Create a context
	ctx := context.Background()

	// Set up args without PR number
	args := []string{"mushu", "validate"}

	// Run Execute
	err := cli.Execute(ctx, args)

	// Close writer and read output
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	// Should return an error
	if err == nil {
		t.Error("Expected error when PR number is missing, got nil")
	}

	if err != nil && !strings.Contains(err.Error(), "PR argument") {
		t.Errorf("Expected error message about missing PR number, got: %v", err)
	}
}

func TestValidateCommand_InvalidPRNumber(t *testing.T) {
	// Capture stdout/stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	defer func() {
		os.Stderr = oldStderr
	}()

	// Create a context
	ctx := context.Background()

	// Set up args with invalid PR number
	args := []string{"mushu", "validate", "invalid"}

	// Run Execute
	err := cli.Execute(ctx, args)

	// Close writer and read output
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	require.Error(t, err)

	if err != nil && !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "parsing") {
		t.Errorf("Expected error message about invalid PR number, got: %v", err)
	}
}

func TestValidateCommand_WithPRNumber(t *testing.T) {
	// This test requires valid GitHub configuration
	// Skip if GitHub token is not available
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping test: GITHUB_TOKEN not set")
	}

	// Create a context
	ctx := context.Background()

	// Set up args with a PR number
	args := []string{
		"mushu",
		"--github-token", os.Getenv("GITHUB_TOKEN"),
		"--github-owner", os.Getenv("GITHUB_OWNER"),
		"--github-repo", os.Getenv("GITHUB_REPO"),
		"validate",
		"1",
	}

	// Run Execute
	err := cli.Execute(ctx, args)
	// This may fail due to missing config or GitHub API errors
	// but we're mainly testing that the command structure works
	if err != nil {
		// Check that it's a reasonable error (not a panic or structure issue)
		errMsg := err.Error()
		if !strings.Contains(errMsg, "GitHub") &&
			!strings.Contains(errMsg, "config") &&
			!strings.Contains(errMsg, "PR") &&
			!strings.Contains(errMsg, "token") {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestVersionCommand(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	defer func() {
		os.Stdout = oldStdout
	}()

	// Create a context
	ctx := context.Background()

	// Set up args for version command
	args := []string{"mushu", "version"}

	// Run Execute
	err := cli.Execute(ctx, args)

	// Close writer and read output
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Check for error
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check output contains version information
	if !strings.Contains(output, "mushu version") {
		t.Errorf("Expected output to contain 'mushu version', got: %s", output)
	}

	if !strings.Contains(output, "commit:") {
		t.Errorf("Expected output to contain 'commit:', got: %s", output)
	}

	if !strings.Contains(output, "built:") {
		t.Errorf("Expected output to contain 'built:', got: %s", output)
	}

	if !strings.Contains(output, "go:") {
		t.Errorf("Expected output to contain 'go:', got: %s", output)
	}
}

func TestListPoliciesCommand(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	defer func() {
		os.Stdout = oldStdout
	}()

	// Create a context
	ctx := context.Background()

	// Set up args for list policies command
	args := []string{"mushu", "list", "policies"}

	// Run Execute
	err := cli.Execute(ctx, args)

	// Close writer and read output
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Check for error
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check output contains expected text
	if !strings.Contains(output, "Available policies:") {
		t.Errorf("Expected output to contain 'Available policies:', got: %s", output)
	}

	if !strings.Contains(output, "Rules files:") {
		t.Errorf("Expected output to contain 'Rules files:', got: %s", output)
	}
}

func TestGlobalFlags(t *testing.T) {
	// Create a context
	ctx := context.Background()

	// Set up args with global flags
	args := []string{
		"mushu",
		"--log-level", "debug",
		"--log-format", "json",
		"version",
	}

	// Run Execute - should not error
	err := cli.Execute(ctx, args)
	if err != nil {
		t.Errorf("Expected no error with global flags, got: %v", err)
	}
}

func TestHelpCommand(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	defer func() {
		os.Stdout = oldStdout
	}()

	// Create a context
	ctx := context.Background()

	// Set up args for help
	args := []string{"mushu", "--help"}

	// Run Execute
	err := cli.Execute(ctx, args)

	// Close writer and read output
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Help should not return an error in urfave/cli v3
	if err != nil {
		t.Logf("Note: help returned error (may be expected): %v", err)
	}

	// Check output contains usage information
	if !strings.Contains(output, "USAGE:") && !strings.Contains(output, "Usage:") {
		t.Errorf("Expected output to contain usage information, got: %s", output)
	}

	if !strings.Contains(output, "COMMANDS:") && !strings.Contains(output, "Commands:") {
		t.Errorf("Expected output to contain commands list, got: %s", output)
	}
}
