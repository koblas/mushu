package policy

import (
	"context"
	"fmt"
	"strings"

	"github.com/koblas/mushu/internal/rules"
	"github.com/koblas/mushu/internal/teams"
	"go.starlark.net/starlark"
)

// PolicyEngine handles Starlark policy evaluation
type PolicyEngine struct {
	teamService teams.TeamService
	ruleLoader  *rules.RuleLoader
	matcher     *rules.RuleMatcher
}

// NewPolicyEngine creates a new policy engine
func NewPolicyEngine(teamService teams.TeamService, rulesFile string) *PolicyEngine {
	return &PolicyEngine{
		teamService: teamService,
		ruleLoader:  rules.NewRuleLoader(rulesFile),
		matcher:     rules.NewRuleMatcher(),
	}
}

// EvaluationResult represents the result of policy evaluation
type EvaluationResult struct {
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

// EvaluatePR evaluates a pull request against policies
func (pe *PolicyEngine) EvaluatePR(ctx context.Context, prData *PRData) (*EvaluationResult, error) {
	// Get user teams
	userTeams, err := pe.teamService.GetUserTeams(ctx, prData.Author)
	if err != nil {
		return nil, fmt.Errorf("failed to get user teams: %w", err)
	}

	// Load rules for the repository
	allRules, err := pe.ruleLoader.LoadRules(".")
	if err != nil {
		return nil, fmt.Errorf("failed to load rules: %w", err)
	}

	// Find matching rules for changed files
	var violations []string
	approvalRequirements := make(map[string]int)

	for _, file := range prData.Files {
		matchedRules := pe.matcher.MatchRules(allRules, file.Filename)

		for _, rule := range matchedRules {
			ruleViolations, ruleApprovals := pe.matcher.MatchConditions(rule.Conditions, []rules.PRFile{file})
			violations = append(violations, ruleViolations...)

			// Merge approval requirements
			for team, count := range ruleApprovals {
				if existing, exists := approvalRequirements[team]; !exists || count > existing {
					approvalRequirements[team] = count
				}
			}
		}
	}

	// Check approval requirements
	for team, requiredCount := range approvalRequirements {
		teamApprovals := 0
		for _, review := range prData.Reviews {
			if review.State == "APPROVED" {
				for _, reviewerTeam := range review.ReviewerTeams {
					if reviewerTeam == team {
						teamApprovals++
						break
					}
				}
			}
		}

		if teamApprovals < requiredCount {
			violations = append(violations, fmt.Sprintf("Requires %d approval(s) from %s", requiredCount, team))
		}
	}

	// Generate Starlark policy and evaluate
	policyCode := pe.generateStarlarkPolicy(allRules, userTeams, prData)
	result, err := pe.evaluateStarlarkPolicy(policyCode, prData, userTeams)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate Starlark policy: %w", err)
	}

	// Merge violations
	if len(violations) > 0 {
		result.Violations = append(result.Violations, violations...)
		result.Decision = "deny"
		result.Reason = strings.Join(violations, "; ")
	}

	// Add approval requirements to result
	if len(approvalRequirements) > 0 {
		result.ApprovalRequirements = approvalRequirements
	}

	return result, nil
}

// generateStarlarkPolicy generates Starlark code from rules
func (pe *PolicyEngine) generateStarlarkPolicy(rules []rules.Rule, userTeams []string, prData *PRData) string {
	var code strings.Builder

	code.WriteString("def evaluate(context):\n")
	code.WriteString("    principal = context[\"principal\"]\n")
	code.WriteString("    resource = context[\"resource\"]\n")
	code.WriteString("    reviews = context[\"reviews\"]\n")
	code.WriteString("    \n")
	code.WriteString("    violations = []\n")
	code.WriteString("    approval_requirements = {}\n")
	code.WriteString("    \n")

	for _, rule := range rules {
		code.WriteString(fmt.Sprintf("    # Rule: %s\n", rule.Name))

		for _, condition := range rule.Conditions {
			switch condition.Type {
			case "sensitive-file":
				code.WriteString("    sensitive_files = [f for f in resource[\"files\"] \n")
				code.WriteString("                      if any(pattern in f[\"filename\"] \n")
				code.WriteString("                            for pattern in [")
				for i, pattern := range condition.Patterns {
					if i > 0 {
						code.WriteString(", ")
					}
					code.WriteString(fmt.Sprintf("\"%s\"", pattern))
				}
				code.WriteString("])]\n")
				code.WriteString("    \n")
				code.WriteString("    if sensitive_files:\n")
				code.WriteString("        if \"security-team\" not in principal[\"teams\"]:\n")
				code.WriteString("            violations.append(\"Sensitive files require security team membership\")\n")
				code.WriteString("        else:\n")
				code.WriteString("            approval_requirements[\"security-team\"] = 1\n")
				code.WriteString("    \n")

			case "file-change":
				if condition.MaxChanges > 0 {
					code.WriteString("    large_files = [f for f in resource[\"files\"] \n")
					code.WriteString(fmt.Sprintf("                   if f[\"changes\"] > %d]\n", condition.MaxChanges))
					code.WriteString("    \n")
					code.WriteString("    if large_files:\n")
					code.WriteString("        if \"senior-backend\" not in principal[\"teams\"]:\n")
					code.WriteString("            violations.append(\"Large changes require senior-backend team membership\")\n")
					code.WriteString("        else:\n")
					code.WriteString("            approval_requirements[\"senior-backend\"] = 1\n")
					code.WriteString("    \n")
				}
			}
		}
	}

	code.WriteString("    \n")
	code.WriteString("    # Check approval requirements\n")
	code.WriteString("    for team, required_count in approval_requirements.items():\n")
	code.WriteString("        team_approvals = [r for r in reviews \n")
	code.WriteString("                         if r[\"state\"] == \"APPROVED\" and team in r[\"reviewer_teams\"]]\n")
	code.WriteString("        if len(team_approvals) < required_count:\n")
	code.WriteString("            violations.append(f\"Requires {required_count} approval(s) from {team}\")\n")
	code.WriteString("    \n")
	code.WriteString("    if violations:\n")
	code.WriteString("        return {\n")
	code.WriteString("            \"decision\": \"deny\",\n")
	code.WriteString("            \"reason\": \"; \".join(violations),\n")
	code.WriteString("            \"approval_requirements\": approval_requirements\n")
	code.WriteString("        }\n")
	code.WriteString("    \n")
	code.WriteString("    return {\"decision\": \"allow\"}\n")

	return code.String()
}

// evaluateStarlarkPolicy evaluates the generated Starlark policy
func (pe *PolicyEngine) evaluateStarlarkPolicy(policyCode string, prData *PRData, userTeams []string) (*EvaluationResult, error) {
	// Create Starlark context
	context := starlark.NewDict(4)

	// Principal
	principal := starlark.NewDict(3)
	if err := principal.SetKey(starlark.String("username"), starlark.String(prData.Author)); err != nil {
		return nil, fmt.Errorf("failed to set principal username: %w", err)
	}
	if err := principal.SetKey(starlark.String("teams"), pe.stringSliceToStarlarkList(userTeams)); err != nil {
		return nil, fmt.Errorf("failed to set principal teams: %w", err)
	}
	if err := principal.SetKey(starlark.String("permissions"), pe.stringSliceToStarlarkList([]string{"merge", "review", "approve"})); err != nil {
		return nil, fmt.Errorf("failed to set principal permissions: %w", err)
	}
	if err := context.SetKey(starlark.String("principal"), principal); err != nil {
		return nil, fmt.Errorf("failed to set context principal: %w", err)
	}

	// Resource
	resource := starlark.NewDict(4)
	if err := resource.SetKey(starlark.String("number"), starlark.MakeInt(prData.Number)); err != nil {
		return nil, fmt.Errorf("failed to set resource number: %w", err)
	}
	if err := resource.SetKey(starlark.String("title"), starlark.String(prData.Title)); err != nil {
		return nil, fmt.Errorf("failed to set resource title: %w", err)
	}
	if err := resource.SetKey(starlark.String("files"), pe.filesToStarlarkList(prData.Files)); err != nil {
		return nil, fmt.Errorf("failed to set resource files: %w", err)
	}
	if err := resource.SetKey(starlark.String("labels"), pe.stringSliceToStarlarkList(prData.Labels)); err != nil {
		return nil, fmt.Errorf("failed to set resource labels: %w", err)
	}
	if err := context.SetKey(starlark.String("resource"), resource); err != nil {
		return nil, fmt.Errorf("failed to set context resource: %w", err)
	}

	// Reviews
	if err := context.SetKey(starlark.String("reviews"), pe.reviewsToStarlarkList(prData.Reviews)); err != nil {
		return nil, fmt.Errorf("failed to set context reviews: %w", err)
	}

	// Execute Starlark code
	thread := &starlark.Thread{Name: "policy"}
	globals, err := starlark.ExecFileOptions(nil, thread, "policy.star", policyCode, make(starlark.StringDict))
	if err != nil {
		return nil, fmt.Errorf("failed to execute Starlark policy: %w", err)
	}

	// Call evaluate function
	evaluateFunc, ok := globals["evaluate"]
	if !ok {
		return nil, fmt.Errorf("evaluate function not found in policy")
	}

	result, err := starlark.Call(thread, evaluateFunc, starlark.Tuple{context}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call evaluate function: %w", err)
	}

	// Convert result to Go struct
	return pe.starlarkDictToEvaluationResult(result.(*starlark.Dict))
}

// Helper functions for Starlark conversion
func (pe *PolicyEngine) stringSliceToStarlarkList(slice []string) *starlark.List {
	list := starlark.NewList(nil)
	for _, s := range slice {
		if err := list.Append(starlark.String(s)); err != nil {
			// Log error but continue - this shouldn't fail in normal operation
			continue
		}
	}
	return list
}

func (pe *PolicyEngine) filesToStarlarkList(files []rules.PRFile) *starlark.List {
	list := starlark.NewList(nil)
	for _, file := range files {
		fileDict := starlark.NewDict(5)
		if err := fileDict.SetKey(starlark.String("filename"), starlark.String(file.Filename)); err != nil {
			continue
		}
		if err := fileDict.SetKey(starlark.String("status"), starlark.String(file.Status)); err != nil {
			continue
		}
		if err := fileDict.SetKey(starlark.String("additions"), starlark.MakeInt(file.Additions)); err != nil {
			continue
		}
		if err := fileDict.SetKey(starlark.String("deletions"), starlark.MakeInt(file.Deletions)); err != nil {
			continue
		}
		if err := fileDict.SetKey(starlark.String("changes"), starlark.MakeInt(file.Changes)); err != nil {
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
		reviewDict := starlark.NewDict(4)
		if err := reviewDict.SetKey(starlark.String("reviewer"), starlark.String(review.Reviewer)); err != nil {
			continue
		}
		if err := reviewDict.SetKey(starlark.String("reviewer_teams"), pe.stringSliceToStarlarkList(review.ReviewerTeams)); err != nil {
			continue
		}
		if err := reviewDict.SetKey(starlark.String("state"), starlark.String(review.State)); err != nil {
			continue
		}
		if err := reviewDict.SetKey(starlark.String("submitted_at"), starlark.String(review.SubmittedAt)); err != nil {
			continue
		}
		if err := list.Append(reviewDict); err != nil {
			continue
		}
	}
	return list
}

func (pe *PolicyEngine) starlarkDictToEvaluationResult(dict *starlark.Dict) (*EvaluationResult, error) {
	result := &EvaluationResult{}

	if decision, ok, _ := dict.Get(starlark.String("decision")); ok {
		result.Decision = string(decision.(starlark.String))
	}

	if reason, ok, _ := dict.Get(starlark.String("reason")); ok {
		result.Reason = string(reason.(starlark.String))
	}

	if approvals, ok, _ := dict.Get(starlark.String("approval_requirements")); ok {
		if approvalDict, ok := approvals.(*starlark.Dict); ok {
			result.ApprovalRequirements = make(map[string]int)
			for _, item := range approvalDict.Items() {
				key := string(item[0].(starlark.String))
				if intVal, ok := item[1].(starlark.Int); ok {
					if val, ok := intVal.Int64(); ok {
						result.ApprovalRequirements[key] = int(val)
					}
				}
			}
		}
	}

	return result, nil
}
