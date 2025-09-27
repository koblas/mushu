package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/koblas/mushu/internal/config"
	"github.com/koblas/mushu/internal/github"
	"github.com/koblas/mushu/internal/logging"
	"github.com/koblas/mushu/internal/policy"
	"github.com/koblas/mushu/internal/teams"
	"github.com/koblas/mushu/internal/version"
)

// Execute runs the CLI application
func Execute(ctx context.Context) error {
	// Parse flags first to handle help and global flags
	fs := flag.NewFlagSet("mushu", flag.ContinueOnError)

	// Define global flags
	fs.String("config", "", "Path to configuration file")
	fs.String("github-token", "", "GitHub API token")
	fs.String("github-owner", "", "GitHub repository owner")
	fs.String("github-repo", "", "GitHub repository name")
	fs.String("log-level", "", "Log level (debug, info, warn, error)")
	fs.String("log-format", "", "Log format (json, text)")

	// Set custom usage function
	fs.Usage = func() {
		fmt.Println("Mushu - Pull request constraint system")
		fmt.Println("Mushu applies Starlark policy rules to determine if pull requests can be approved")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  mushu [global-flags] <command> [arguments]")
		fmt.Println()
		fmt.Println("Global Flags:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  validate <pr-number>    Validate a pull request against configured policies and rules")
		fmt.Println("  list <policies|teams>   List available policies or teams")
		fmt.Println("  generate policy         Generate Starlark policy from YAML rules")
		fmt.Println("  version                 Show version information")
		fmt.Println("  help                    Show this help message")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  mushu validate 123")
		fmt.Println("  mushu -github-token=token123 validate 123")
		fmt.Println("  mushu list policies")
		fmt.Println("  mushu list teams")
		fmt.Println("  mushu generate policy")
	}

	// Parse flags
	if err := fs.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			// Help was requested, usage was already printed
			os.Exit(0)
		}
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	// Load config with command line flags (highest priority)
	cfg, err := config.Load("", fs)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Re-initialize logging with final config values
	newLogger := logging.New(cfg.Logging.Level, cfg.Logging.Format)
	ctx = logging.WithLogger(ctx, newLogger)

	// Get remaining arguments (command and its args)
	args := fs.Args()
	if len(args) < 1 {
		return fmt.Errorf("no command specified. Use 'mushu -h' for usage information")
	}

	command := args[0]
	commandArgs := args[1:]

	switch command {
	case "validate":
		return runValidateCommand(ctx, cfg, commandArgs)
	case "list":
		return runListCommand(ctx, cfg, commandArgs)
	case "generate":
		return runGenerateCommand(ctx, cfg, commandArgs)
	case "version":
		return runVersionCommand(ctx, cfg, commandArgs)
	case "help":
		// Help is handled by the flag package's Usage function
		return fmt.Errorf("use 'mushu -h' for help")
	default:
		return fmt.Errorf("unknown command: %s. Use 'mushu -h' for usage information", command)
	}
}

// runValidateCommand handles the validate command
func runValidateCommand(ctx context.Context, cfg *config.Config, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("validate command requires a PR number")
	}

	prStr := args[0]
	return runValidate(ctx, cfg, prStr)
}

// runListCommand handles the list command
func runListCommand(ctx context.Context, cfg *config.Config, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("list command requires a resource type (policies or teams)")
	}

	resource := args[0]
	return runList(ctx, cfg, resource)
}

// runVersionCommand handles the version command
func runVersionCommand(ctx context.Context, cfg *config.Config, args []string) error {
	logging.Info(ctx, "Version command executed")

	info := version.Get()

	// Print version information
	fmt.Printf("mushu version %s\n", info.Version)
	fmt.Printf("  commit: %s\n", info.Commit)
	fmt.Printf("  built: %s\n", info.BuildTime)
	fmt.Printf("  go: %s\n", info.GoVersion)

	return nil
}
func runGenerateCommand(ctx context.Context, cfg *config.Config, args []string) error {
	// Parse flags for generate command
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	rulesFile := fs.String("rules", cfg.Policy.RulesFile, "Rules file to process")
	outputFile := fs.String("output", "", "Output file (default: stdout)")

	if err := fs.Parse(args); err != nil {
		logging.Error(ctx, "Failed to parse generate command flags", "error", err)
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	logging.Info(ctx, "Generate command executed", "rules_file", *rulesFile, "output_file", *outputFile)

	// For now, just print the parameters
	fmt.Printf("Generate policy from rules file: %s\n", *rulesFile)
	if *outputFile != "" {
		fmt.Printf("Output to file: %s\n", *outputFile)
	} else {
		fmt.Println("Output to stdout")
	}

	return runGenerate(ctx, cfg)
}

// runValidate validates a pull request
func runValidate(ctx context.Context, cfg *config.Config, prStr string) error {
	logging.Info(ctx, "Starting PR validation", "pr", prStr)

	// Parse PR number
	prNumber, err := github.ParsePRNumber(prStr)
	if err != nil {
		logging.Error(ctx, "Invalid PR number", "pr", prStr, "error", err)
		return fmt.Errorf("invalid PR number: %w", err)
	}

	logging.Debug(ctx, "Parsed PR number", "pr_number", prNumber)

	// Create GitHub client
	client, err := github.CreateGitHubClient(cfg.GitHub.Token, cfg.GitHub.BaseURL)
	if err != nil {
		logging.Error(ctx, "Failed to create GitHub client", "error", err)
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	logging.Debug(ctx, "Created GitHub client", "base_url", cfg.GitHub.BaseURL)

	// Create team service
	yamlService := teams.NewYAMLTeamService(os.DirFS("."), cfg.Teams.TeamsFile, cfg.Teams.TeamsDir)
	// Load teams (optional - only fail on real errors, not missing files)
	if err := yamlService.Load(ctx); err != nil && !errors.Is(err, os.ErrNotExist) {
		logging.Error(ctx, "Failed to load teams from YAML", "error", err)
		return fmt.Errorf("failed to load teams: %w", err)
	}

	var teamService teams.TeamService = yamlService
	if cfg.Teams.UseGitHubAPI {
		logging.Debug(ctx, "Using GitHub API for team management")
		githubTeamService := teams.NewGitHubTeamService(client, cfg.GitHub.Owner, cfg.GitHub.Repo)
		teamService = teams.NewCompositeTeamService(yamlService, githubTeamService, true)
	} else {
		logging.Debug(ctx, "Using YAML-only team management")
	}

	// Create policy engine
	policyEngine := policy.NewPolicyEngine(teamService, cfg.Policy.RulesFile)
	logging.Debug(ctx, "Created policy engine", "rules_file", cfg.Policy.RulesFile)

	// Create GitHub service
	githubService := github.NewGitHubService(client, cfg.GitHub.Owner, cfg.GitHub.Repo, teamService)
	logging.Debug(ctx, "Created GitHub service", "owner", cfg.GitHub.Owner, "repo", cfg.GitHub.Repo)

	// Validate PR
	logging.Info(ctx, "Validating PR", "pr_number", prNumber)
	result, err := githubService.ValidatePR(ctx, prNumber, policyEngine)
	if err != nil {
		logging.Error(ctx, "PR validation failed", "pr_number", prNumber, "error", err)
		return fmt.Errorf("failed to validate PR: %w", err)
	}

	// Log validation result
	logging.Info(ctx, "PR validation completed",
		"pr_number", prNumber,
		"decision", result.Decision,
		"violations_count", len(result.Violations),
		"approval_requirements_count", len(result.ApprovalRequirements))

	// Print result to stdout for user
	fmt.Printf("PR #%d validation result: %s\n", prNumber, result.Decision)
	if result.Reason != "" {
		fmt.Printf("Reason: %s\n", result.Reason)
	}
	if len(result.Violations) > 0 {
		fmt.Println("Violations:")
		for _, violation := range result.Violations {
			fmt.Printf("  - %s\n", violation)
			logging.Warn(ctx, "Constraint violation", "pr_number", prNumber, "violation", violation)
		}
	}
	if len(result.ApprovalRequirements) > 0 {
		fmt.Println("Approval Requirements:")
		for team, count := range result.ApprovalRequirements {
			fmt.Printf("  - %s: %d approval(s)\n", team, count)
			logging.Info(ctx, "Approval requirement", "pr_number", prNumber, "team", team, "count", count)
		}
	}

	// Exit with error code if validation failed
	if result.Decision == "deny" {
		logging.Warn(ctx, "PR validation denied", "pr_number", prNumber)
		os.Exit(1)
	}

	logging.Info(ctx, "PR validation approved", "pr_number", prNumber)
	return nil
}

// runList lists available policies or teams
func runList(ctx context.Context, cfg *config.Config, resource string) error {
	switch resource {
	case "policies":
		return listPolicies(ctx, cfg)
	case "teams":
		return listTeams(ctx, cfg)
	default:
		return fmt.Errorf("unknown resource: %s (supported: policies, teams)", resource)
	}
}

// listPolicies lists available policies
func listPolicies(ctx context.Context, cfg *config.Config) error {
	logging.Info(ctx, "Listing available policies")

	fmt.Println("Available policies:")

	// List Starlark policy files
	if len(cfg.Policy.PolicyFiles) > 0 {
		fmt.Println("Starlark policies:")
		for _, file := range cfg.Policy.PolicyFiles {
			fmt.Printf("  - %s\n", file)
			logging.Debug(ctx, "Found policy file", "file", file)
		}
	}

	// List rules files
	fmt.Println("Rules files:")
	fmt.Printf("  - %s\n", cfg.Policy.RulesFile)
	logging.Debug(ctx, "Found rules file", "file", cfg.Policy.RulesFile)

	return nil
}

// listTeams lists available teams
func listTeams(ctx context.Context, cfg *config.Config) error {
	logging.Info(ctx, "Listing available teams")

	fmt.Println("Available teams:")

	// Load teams from YAML (optional - only fail on real errors, not missing files)
	yamlService := teams.NewYAMLTeamService(os.DirFS("."), cfg.Teams.TeamsFile, cfg.Teams.TeamsDir)
	if err := yamlService.Load(ctx); err != nil {
		logging.Error(ctx, "Failed to load teams from YAML", "error", err)
		return fmt.Errorf("failed to load teams: %w", err)
	}

	// List teams (this would need to be implemented in the team service)
	fmt.Println("Teams loaded from YAML files")
	logging.Debug(ctx, "Teams loaded successfully", "teams_file", cfg.Teams.TeamsFile, "teams_dir", cfg.Teams.TeamsDir)

	return nil
}

// runGenerate generates Starlark policy from rules
func runGenerate(ctx context.Context, cfg *config.Config) error {
	logging.Info(ctx, "Policy generation requested")

	fmt.Println("Policy generation not yet implemented")
	fmt.Println("This would generate Starlark code from mushu.yaml rules")

	logging.Warn(ctx, "Policy generation not implemented", "rules_file", cfg.Policy.RulesFile)
	return nil
}
