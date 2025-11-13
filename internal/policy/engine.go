package policy

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"time"

	"github.com/koblas/mushu/internal/loader"
	"github.com/koblas/mushu/internal/rules"
	"github.com/koblas/mushu/internal/teams"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// PolicyEngine handles Starlark policy evaluation
type PolicyEngine struct {
	teamService teams.TeamService

	programs map[string]*starlark.Program
}

// NewPolicyEngine creates a new policy engine
func NewPolicyEngine(teamService teams.TeamService, rulesFile string) *PolicyEngine {
	if teamService == nil {
		teamService = teams.NewNoOpTeamService()
	}

	return &PolicyEngine{
		teamService: teamService,
	}
}

// EvaluationResult represents the result of policy evaluation
type EvaluationResult struct {
	Rule *rules.Rule `json:"-"`

	Decision             string         `json:"decision"`
	Reason               string         `json:"reason,omitempty"`
	ApprovalRequirements map[string]int `json:"approval_requirements,omitempty"`
	Violations           []string       `json:"violations,omitempty"`
}

// PRData represents pull request data
type PRData struct {
	Number            int
	Title             string
	State             string
	Author            string
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

//go:embed policy.star
var starlarkPolicy string

// EvaluatePR evaluates a pull request against policies
func (pe *PolicyEngine) EvaluatePR(ctx context.Context, prData *PRData) ([]*EvaluationResult, error) {
	// Get user teams
	userTeams, err := pe.teamService.GetUserTeams(ctx, prData.Author)
	if err != nil {
		return nil, fmt.Errorf("failed to get user teams: %w", err)
	}

	loader := rules.NewPrRuleLoader(prData.Files)

	// Load rules for the repository
	allRules, err := loader.LoadRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load rules: %w", err)
	}

	var files []string
	for _, f := range prData.Files {
		files = append(files, f.Filename)
	}

	var results []*EvaluationResult

	for _, rule := range allRules {
		slog.DebugContext(ctx, "evaluating rule", "rule_name", rule.Name)

		match, err := rule.MatchFiles(ctx, files)
		if err != nil {
			slog.ErrorContext(ctx, "error matching rule files", "rule", rule.Name, "error", err)
		}
		if !match {
			continue
		}

		tstart := time.Now()

		result, err := pe.evaluateStarlarkPolicy(ctx, rule, starlarkPolicy, prData, userTeams)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate Starlark policy: %w", err)
		}

		slog.DebugContext(ctx, "evaluating rule", "rule_name", rule.Name, "tooks", time.Since(tstart))

		results = append(results, result)
	}

	// // Find matching rules for changed files
	// var violations []string
	// approvalRequirements := make(map[string]int)

	// for _, file := range prData.Files {
	// 	matchedRules := pe.matcher.MatchRules(allRules, file.Filename)

	// 	for _, rule := range matchedRules {
	// 		ruleViolations, ruleApprovals := pe.matcher.MatchConditions(rule.Conditions, []rules.PRFile{file})
	// 		violations = append(violations, ruleViolations...)

	// 		// Merge approval requirements
	// 		for team, count := range ruleApprovals {
	// 			if existing, exists := approvalRequirements[team]; !exists || count > existing {
	// 				approvalRequirements[team] = count
	// 			}
	// 		}
	// 	}
	// }

	// // Check approval requirements
	// for team, requiredCount := range approvalRequirements {
	// 	teamApprovals := 0
	// 	for _, review := range prData.Reviews {
	// 		if review.State == "APPROVED" {
	// 			for _, reviewerTeam := range review.ReviewerTeams {
	// 				if reviewerTeam == team {
	// 					teamApprovals++
	// 					break
	// 				}
	// 			}
	// 		}
	// 	}

	// 	if teamApprovals < requiredCount {
	// 		violations = append(violations, fmt.Sprintf("Requires %d approval(s) from %s", requiredCount, team))
	// 	}
	// }

	// // Generate Starlark policy and evaluate
	// result, err := pe.evaluateStarlarkPolicy(starlarkPolicy, prData, userTeams)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to evaluate Starlark policy: %w", err)
	// }

	// // Merge violations
	// if len(violations) > 0 {
	// 	result.Violations = append(result.Violations, violations...)
	// 	result.Decision = "deny"
	// 	result.Reason = strings.Join(violations, "; ")
	// }

	// // Add approval requirements to result
	// if len(approvalRequirements) > 0 {
	// 	result.ApprovalRequirements = approvalRequirements
	// }

	return results, nil
}

var (
	keyPrincipal = starlark.String("principal")
	keyResource  = starlark.String("resource")
	keyReviews   = starlark.String("reviews")
)

// evaluateStarlarkPolicy evaluates the generated Starlark policy
func (pe *PolicyEngine) evaluateStarlarkPolicy(ctx context.Context, rule *rules.Rule, policyCode string, prData *PRData, userTeams []string) (*EvaluationResult, error) {
	// Create Starlark context
	context := starlark.NewDict(4)

	// Principal
	principal, err := convertValueToStarlark(map[string]any{
		"username":    prData.Author,
		"teams":       userTeams,
		"permissions": []string{"merge", "review", "approve"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to convert principal to Starlark: %w", err)
	}
	if err := context.SetKey(keyPrincipal, principal); err != nil {
		return nil, fmt.Errorf("failed to set context principal: %w", err)
	}

	// Resource
	resource, err := convertMapToDict(map[string]any{
		"number": prData.Number,
		"title":  prData.Title,
		"files":  pe.filesToStarlarkList(prData.Files),
		"labels": prData.Labels,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to convert resource to Starlark: %w", err)
	}
	if err := context.SetKey(keyResource, resource); err != nil {
		return nil, fmt.Errorf("failed to set context resource: %w", err)
	}

	// Reviews
	if err := context.SetKey(keyReviews, pe.reviewsToStarlarkList(prData.Reviews)); err != nil {
		return nil, fmt.Errorf("failed to set context reviews: %w", err)
	}

	prog, err := pe.loadProgram(rule.Name)
	if err != nil {
		slog.ErrorContext(ctx, "error loading starlark program", "rule", rule.Name, "error", err)

		return nil, fmt.Errorf("failed to load Starlark program: %w", err)
	}

	// Execute Starlark code
	thread := &starlark.Thread{
		Name: rule.Name,
		Print: func(thread *starlark.Thread, msg string) {
			slog.InfoContext(ctx, "starlark", slog.String("thread", thread.Name), slog.String("msg", msg))
		},
		Load: loader.Load,
	}

	globals, err := prog.Init(thread, make(starlark.StringDict))
	if err != nil {
		return nil, fmt.Errorf("failed to execute Starlark policy: %w", err)
	}
	globals.Freeze()

	// Call evaluate function
	evaluateFunc, ok := globals["evaluate"]
	if !ok {
		return nil, fmt.Errorf("evaluate function not found in policy")
	}

	var kwargs []starlark.Tuple
	for k, v := range rule.Args {
		val, err := convertValueToStarlark(v)
		if err != nil {
			return nil, fmt.Errorf("failed to convert rule arg %q to Starlark: %w", k, err)
		}

		kwargs = append(kwargs, starlark.Tuple{starlark.String(k), val})
	}

	result, err := starlark.Call(thread, evaluateFunc, starlark.Tuple{context}, kwargs)
	if err != nil {
		return nil, fmt.Errorf("failed to call evaluate function: %w", err)
	}

	// Convert result to Go struct
	return pe.starlarkDictToEvaluationResult(rule, result.(*starlark.Dict))
}

func (pe *PolicyEngine) loadProgram(ruleName string) (*starlark.Program, error) {
	if pe.programs == nil {
		pe.programs = make(map[string]*starlark.Program)
	}

	if prog, exists := pe.programs[ruleName]; exists {
		return prog, nil
	}

	name, code, err := loader.Fetch(ruleName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch built-in policy %q: %w", ruleName, err)
	}

	decls := starlark.StringDict{}

	_, prog, err := starlark.SourceProgramOptions(
		&syntax.FileOptions{
			Set:               false,
			While:             true,
			TopLevelControl:   false,
			GlobalReassign:    false,
			Recursion:         true,
			LoadBindsGlobally: false,
		},
		name,
		code,
		decls.Has,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to compile %q: %w", ruleName, err)
	}

	pe.programs[ruleName] = prog
	return prog, nil
}

func (pe *PolicyEngine) filesToStarlarkList(files []rules.PRFile) *starlark.List {
	list := starlark.NewList(nil)
	for _, file := range files {
		fileDict, err := convertValueToStarlark(map[string]any{
			"filename":  file.Filename,
			"status":    file.Status,
			"additions": file.Additions,
			"deletions": file.Deletions,
			"changes":   file.Changes,
		})
		if err != nil {
			continue
		}

		if err := list.Append(fileDict); err != nil {
			continue
		}
	}
	return list
}

func (pe *PolicyEngine) reviewsToStarlarkList(reviews []Review) *starlark.List {
	list := starlark.NewList(nil)
	for _, review := range reviews {
		reviewDict, err := convertValueToStarlark(map[string]any{
			"reviewer":       review.Reviewer,
			"reviewer_teams": review.ReviewerTeams,
			"state":          review.State,
			"submitted_at":   review.SubmittedAt,
		})
		if err != nil {
			continue
		}
		if err := list.Append(reviewDict); err != nil {
			continue
		}
	}
	return list
}

func (pe *PolicyEngine) starlarkDictToEvaluationResult(rule *rules.Rule, dict *starlark.Dict) (*EvaluationResult, error) {
	clean, err := convertDictToMap(dict)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Starlark dict to map: %w", err)
	}

	result := &EvaluationResult{
		Rule: rule,
	}

	if value, found := clean["decision"]; found {
		if decision, ok := value.(string); ok {
			result.Decision = decision
		}
	}

	if value, found := clean["reason"]; found {
		if reason, ok := value.(string); ok {
			result.Reason = reason
		}
	}

	if value, found := clean["approval_requirements"]; found {
		result.ApprovalRequirements = make(map[string]int)
		if approvalsMap, ok := value.(map[string]any); ok {
			for key, val := range approvalsMap {
				if intVal, ok := val.(int); ok {
					result.ApprovalRequirements[key] = intVal
				}
			}
		}
	}

	return result, nil
}
