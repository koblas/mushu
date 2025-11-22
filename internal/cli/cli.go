package cli

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/actions-go/toolkit/core"
	githubtool "github.com/actions-go/toolkit/github"
	"github.com/koblas/mushu/internal/config"
	"github.com/koblas/mushu/internal/engine"
	"github.com/koblas/mushu/internal/github"
	"github.com/koblas/mushu/internal/logging"
	"github.com/koblas/mushu/internal/teams"
	"github.com/koblas/mushu/internal/version"
	"github.com/urfave/cli/v3"
)

type exitCode int

type ErrCmdUsage struct {
	err error
	cmd *cli.Command
}

func (e *ErrCmdUsage) Error() string {
	return fmt.Sprintf("command usage error: %s", e.err.Error())
}

const (
	exitOK    exitCode = 0
	exitError exitCode = 1
	// exitCancel  exitCode = 2
	exitAuth exitCode = 4
	// exitPending exitCode = 8
)

func Main() exitCode {
	ctx := context.Background()

	// Load initial config for logging setup
	cfg, err := config.Load(ctx, "config.yaml", nil)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create logger and add to context
	logger := logging.New(cfg.Logging.Level, cfg.Logging.Format)
	slog.SetDefault(logger)

	// Get version information once
	versionInfo := version.Get()
	slog.DebugContext(ctx, "Starting mushu",
		"version", versionInfo.Version,
		"commit", versionInfo.Commit,
		"build_time", versionInfo.BuildTime,
		"go_version", versionInfo.GoVersion,
		"log_level", cfg.Logging.Level,
		"log_format", cfg.Logging.Format)

	if err := Execute(ctx, os.Args); err != nil {
		if uerr := new(ErrCmdUsage); errors.As(err, &uerr) {
			fmt.Printf("Error: %s\n\n", uerr.err.Error())
			_ = cli.ShowSubcommandHelp(uerr.cmd)

			return exitError
		}

		slog.ErrorContext(ctx, "Application error", "error", err)

		if errors.Is(err, github.ErrNoAuth) {
			return exitAuth
		}

		return exitError
	}

	slog.DebugContext(ctx, "Application completed successfully")

	return exitOK
}

// Execute runs the CLI application using urfave/cli
func Execute(ctx context.Context, args []string) error {
	app := &cli.Command{
		Name:        "mushu",
		Usage:       "Pull request constraint system",
		Description: "Mushu applies Starlark policy rules to determine if pull requests can be approved",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Usage: "Path to configuration file",
			},
			&cli.StringFlag{
				Name:  "github-token",
				Usage: "GitHub API token",
			},
			&cli.StringFlag{
				Name:  "github-owner",
				Usage: "GitHub repository owner",
			},
			&cli.StringFlag{
				Name:  "github-repo",
				Usage: "GitHub repository name",
			},
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "Log level (debug, info, warn, error)",
			},
			&cli.StringFlag{
				Name:  "log-format",
				Usage: "Log format (json, text)",
			},
		},
		Commands: []*cli.Command{
			validateCommand(),
			listCommand(),
			generateCommand(),
			versionCommand(),
		},
	}

	return app.Run(ctx, args)
}

func usageErrorHandler(ctx context.Context, cmd *cli.Command, err error, isSubcommand bool) error {
	return &ErrCmdUsage{err: err, cmd: cmd}
}

// loadConfigWithFlags loads config with values from CLI flags
func loadConfigWithFlags(ctx context.Context, cmd *cli.Command) (context.Context, *config.Config, error) {
	// Create a simple flag set to pass to config.Load
	// Since urfave/cli doesn't use flag.FlagSet, we need to create one
	// and populate it with values from the cli context

	// For now, load the basic config and override with CLI flags
	cfg, err := config.Load(ctx, cmd.String("config"), nil)
	if err != nil {
		return ctx, nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override with CLI flags if provided
	if token := cmd.String("github-token"); token != "" {
		cfg.GitHub.Token = token
	}
	if owner := cmd.String("github-owner"); owner != "" {
		cfg.GitHub.Owner = owner
	}
	if repo := cmd.String("github-repo"); repo != "" {
		cfg.GitHub.Repo = repo
	}
	if logLevel := cmd.String("log-level"); logLevel != "" {
		cfg.Logging.Level = logLevel
	}
	if logFormat := cmd.String("log-format"); logFormat != "" {
		cfg.Logging.Format = logFormat
	}

	// Re-initialize logging with final config values
	newLogger := logging.New(cfg.Logging.Level, cfg.Logging.Format)
	slog.SetDefault(newLogger)

	return ctx, cfg, nil
}

// validateCommand returns the validate command
func validateCommand() *cli.Command {
	return &cli.Command{
		Name:        "validate",
		Usage:       "Validate a pull request against configured policies and rules",
		Description: "Validates a pull request number against the configured policies",
		ArgsUsage:   "<pr-number|pr-url>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "dry-run",
				Usage:   "Perform a dry run without making any changes (default: false)",
				Aliases: []string{"n"},
			},
		},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:      "pr",
				UsageText: "Either the pull request number or full URL to the PR",
			},
		},
		OnUsageError: usageErrorHandler,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// args := cmd.Args()
			// if args.Len() < 1 {
			// 	return fmt.Errorf("validate command requires a PR number")
			// }

			// Load config with CLI flags
			ctx, cfg, err := loadConfigWithFlags(ctx, cmd)
			if err != nil {
				return err
			}

			prValue := cmd.StringArg("pr")
			if prValue == "" {
				act := githubtool.ParseActionEnv()

				hook := act.Payload.PullRequest
				if hook == nil || hook.Number == nil {
					return fmt.Errorf("validate command requires a PR argument")
				}

				prValue = fmt.Sprintf("%d", *hook.Number)
			}

			baseRepo := github.NewWithHost(cfg.GitHub.Owner, cfg.GitHub.Repo, cfg.GitHub.Host)

			prNumber, repo, err := github.ParsePRValue(prValue)
			if err != nil {
				return fmt.Errorf("invalid PR value: %w", err)
			}
			if repo == nil {
				repo = baseRepo
			}

			ctx = logging.ContextWith(ctx, slog.Int("pr", prNumber))

			client, err := github.NewClient(ctx, &github.Config{
				Token: cfg.GitHub.Token,
				Repo:  repo,
			})
			if err != nil {
				return fmt.Errorf("failed to create GitHub client: %w", err)
			}

			// Create team service
			yamlService := teams.NewYAMLTeamService(os.DirFS(cfg.Teams.TeamsDir), cfg.Teams.TeamsFile)
			// Load teams (optional - only fail on real errors, not missing files)
			if err := yamlService.Load(ctx); err != nil && !errors.Is(err, os.ErrNotExist) {
				slog.ErrorContext(ctx, "Failed to load teams from YAML", "error", err)
				return fmt.Errorf("failed to load teams: %w", err)
			}

			var teamService teams.TeamService = yamlService

			// Create policy engine
			policyEngine := engine.NewPolicyEngine(teamService, cfg.Policy.RulesFile, os.DirFS(cfg.Policy.PolicyDir))
			slog.DebugContext(ctx, "Created policy engine", "rules_file", cfg.Policy.RulesFile)

			tstart := time.Now()
			data, err := client.GetPRData(ctx, prNumber, teamService)
			if err != nil {
				return fmt.Errorf("failed to get PR data: %w", err)
			}
			slog.DebugContext(ctx, "Fetched PR data", "took", time.Since(tstart))

			// Validate PR
			slog.InfoContext(ctx, "Validating PR")
			result, err := policyEngine.EvaluatePR(ctx, data)
			if err != nil {
				slog.ErrorContext(ctx, "PR validation failed", logging.Err(err))

				return fmt.Errorf("failed to validate PR: %w", err)
			}

			summary := strings.Builder{}
			for _, result := range result {
				slog.InfoContext(ctx,
					"evaluation result",
					slog.String("rule_name", result.Rule.Name),
					slog.String("decision", result.Decision),
					slog.String("reason", result.Reason),
					slog.Int("violations_count", len(result.Violations)),
					slog.Int("approval_requirements_count", len(result.ApprovalRequirements)),
				)

				summary.WriteString("## ")
				if result.Decision == "approve" {
					summary.WriteString("✅ ")
				} else {
					summary.WriteString("❌ ")
				}

				summary.WriteString(result.Rule.Name)
				summary.WriteString("\n\n")
				summary.WriteString(result.Reason)
				summary.WriteString("\n\n")
			}

			if !cmd.Bool("dry-run") {
				core.AddStepSummary(summary.String())
			}

			// // Log validation result
			// slog.InfoContext(ctx, "PR validation completed",
			// 	"decision", result.Decision,
			// 	"violations_count", len(result.Violations),
			// 	"approval_requirements_count", len(result.ApprovalRequirements))

			// // Print result to stdout for user
			// if result.Reason != "" {
			// 	fmt.Printf("Reason: %s\n", result.Reason)
			// }
			// if len(result.Violations) > 0 {
			// 	fmt.Println("Violations:")
			// 	for _, violation := range result.Violations {
			// 		fmt.Printf("  - %s\n", violation)
			// 		slog.InfoContext(ctx, "Constraint violation", slog.String("violation", violation))
			// 	}
			// }
			// if len(result.ApprovalRequirements) > 0 {
			// 	fmt.Println("Approval Requirements:")
			// 	for team, count := range result.ApprovalRequirements {
			// 		fmt.Printf("  - %s: %d approval(s)\n", team, count)
			// 		slog.InfoContext(ctx, "Approval requirement",
			// 			slog.String("team", team),
			// 			slog.Int("count", count),
			// 		)
			// 	}
			// }

			// // Exit with error code if validation failed
			// if result.Decision == "deny" {
			// 	slog.InfoContext(ctx, "PR validation denied")
			// 	os.Exit(1)
			// }

			// slog.InfoContext(ctx, "PR validation approved")
			return nil
		},
	}
}

// listCommand returns the list command
func listCommand() *cli.Command {
	return &cli.Command{
		Name:        "list",
		Usage:       "List available policies or teams",
		Description: "Lists available resources (policies or teams)",
		Commands: []*cli.Command{
			{
				Name:  "policies",
				Usage: "List available policies",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					ctx, cfg, err := loadConfigWithFlags(ctx, cmd)
					if err != nil {
						return err
					}

					slog.InfoContext(ctx, "Listing available policies")

					fmt.Println("Available policies:")

					// List Starlark policy files
					if len(cfg.Policy.PolicyFiles) > 0 {
						fmt.Println("Starlark policies:")
						for _, file := range cfg.Policy.PolicyFiles {
							fmt.Printf("  - %s\n", file)
							slog.DebugContext(ctx, "Found policy file", "file", file)
						}
					}

					// List rules files
					fmt.Println("Rules files:")
					fmt.Printf("  - %s\n", cfg.Policy.RulesFile)
					slog.DebugContext(ctx, "Found rules file", "file", cfg.Policy.RulesFile)

					return nil
				},
			},
			{
				Name:  "teams",
				Usage: "List available teams",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					ctx, cfg, err := loadConfigWithFlags(ctx, cmd)
					if err != nil {
						return err
					}

					slog.InfoContext(ctx, "Listing available teams")

					// Load teams from YAML (optional - only fail on real errors, not missing files)
					yamlService := teams.NewYAMLTeamService(os.DirFS("."), cfg.Teams.TeamsFile)
					if err := yamlService.Load(ctx); err != nil {
						slog.ErrorContext(ctx, "Failed to load teams from YAML", "error", err)
						return fmt.Errorf("failed to load teams: %w", err)
					}

					// List teams (this would need to be implemented in the team service)
					slog.DebugContext(ctx, "Teams loaded successfully", "teams_file", cfg.Teams.TeamsFile, "teams_dir", cfg.Teams.TeamsDir)

					fmt.Println("Available teams:")

					return nil
				},
			},
		},
	}
}

// generateCommand returns the generate command
func generateCommand() *cli.Command {
	return &cli.Command{
		Name:        "generate",
		Usage:       "Generate Starlark policy from YAML rules",
		Description: "Generates Starlark policy code from YAML rules configuration",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "rules",
				Usage: "Rules file to process",
			},
			&cli.StringFlag{
				Name:  "output",
				Usage: "Output file (default: stdout)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			ctx, cfg, err := loadConfigWithFlags(ctx, cmd)
			if err != nil {
				return err
			}

			rulesFile := cmd.String("rules")
			if rulesFile == "" {
				rulesFile = cfg.Policy.RulesFile
			}
			outputFile := cmd.String("output")

			slog.InfoContext(ctx, "Generate command executed", "rules_file", rulesFile, "output_file", outputFile)

			// For now, just print the parameters
			fmt.Printf("Generate policy from rules file: %s\n", rulesFile)
			if outputFile != "" {
				fmt.Printf("Output to file: %s\n", outputFile)
			} else {
				fmt.Println("Output to stdout")
			}

			slog.InfoContext(ctx, "Policy generation requested")

			fmt.Println("Policy generation not yet implemented")
			fmt.Println("This would generate Starlark code from mushu.yaml rules")

			slog.WarnContext(ctx, "Policy generation not implemented", "rules_file", cfg.Policy.RulesFile)
			return nil
		},
	}
}

// versionCommand returns the version command
func versionCommand() *cli.Command {
	return &cli.Command{
		Name:        "version",
		Usage:       "Show version information",
		Description: "Displays version, commit, build time, and Go version information",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			slog.InfoContext(ctx, "Version command executed")

			info := version.Get()

			// Print version information
			fmt.Printf("mushu version %s\n", info.Version)
			fmt.Printf("  commit: %s\n", info.Commit)
			fmt.Printf("  built: %s\n", info.BuildTime)
			fmt.Printf("  go: %s\n", info.GoVersion)

			return nil
		},
	}
}
