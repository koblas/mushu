# Mushu Design Document

## Overview

Mushu is a pull request constraint system that applies Starlark policy rules to determine if pull requests can be approved. The system combines team membership data from GitHub organizations or YAML files with Starlark policies to enforce organizational standards and security requirements.

## Core Concepts

### 1. Teams as Roles

- Teams serve as roles directly - no separate role concept
- Teams from GitHub organizations or YAML files map directly to permissions
- Simplifies the mental model and reduces configuration complexity

### 2. Policy Evaluation

- Starlark policies define what actions are allowed/forbidden
- Policies can reference team membership, file changes, and PR metadata
- Both general policies and directory-specific rules are supported

### 3. Multi-Source Team Data

- GitHub GraphQL API for organization teams and user memberships
- YAML files for local team definitions with external user memberships
- Fallback mechanism: YAML first, then GitHub API
- Users are external identifiers (GitHub usernames) - no local user definitions

## Architecture

### Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   GitHub API    │    │   YAML Files    │    │Starlark Policies│
│   (Teams)       │    │   (Teams)       │    │   (Rules)       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │  Policy Engine  │
                    │  (Starlark)     │
                    └─────────────────┘
                                 │
                    ┌─────────────────┐
                    │   Validator     │
                    │  (Constraints)  │
                    └─────────────────┘
                                 │
                    ┌─────────────────┐
                    │   CLI Tool      │
                    │   (mushu)       │
                    └─────────────────┘
```

### Data Flow

1. **PR Data Collection**: Fetch PR metadata, reviews, labels, and file changes
2. **User Resolution**: Resolve PR author as external user (GitHub username)
3. **Team Resolution**: Resolve user's team memberships from YAML or GitHub API
4. **Policy Loading**: Load Starlark policies from files and generated rules
5. **Policy Evaluation**: Apply policies against PR data and team membership
6. **Decision**: Return allow/deny decision with reasons

## Data Structures

### Team Membership

```yaml
# teams.yaml - Only team definitions with external user memberships
teams:
  senior-backend:
    description: "Senior backend developers with merge permissions"
    members:
      - "alice" # GitHub username
      - "eve" # GitHub username

  security-team:
    description: "Security and compliance team"
    members:
      - "bob" # GitHub username
      - "eve" # GitHub username

  platform-team:
    description: "Platform and infrastructure team"
    members:
      - "alice" # GitHub username
      - "bob" # GitHub username
      - "charlie" # GitHub username
```

### Pull Request Data

```go
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
    Files             []PRFile
}

type PRFile struct {
    Filename  string
    Status    string // added, modified, removed, renamed
    Additions int
    Deletions int
    Changes   int
}
```

### Starlark Context

```python
# Context object passed to Starlark policies
context = {
    "principal": {
        "username": "alice",  # External GitHub username
        "teams": ["senior-backend", "platform-team"],  # Resolved from teams.yaml or GitHub API
        "permissions": ["merge", "review", "approve"]  # Derived from team memberships
    },
    "action": "merge",
    "resource": {
        "number": 123,
        "title": "Fix security vulnerability",
        "files": [...],
        "labels": ["security", "urgent"]
    },
    "reviews": [  # PR reviews and approvals
        {
            "reviewer": "bob",
            "reviewer_teams": ["security-team"],
            "state": "APPROVED",
            "submitted_at": "2024-01-15T10:30:00Z"
        },
        {
            "reviewer": "charlie",
            "reviewer_teams": ["senior-backend"],
            "state": "CHANGES_REQUESTED",
            "submitted_at": "2024-01-15T11:00:00Z"
        }
    ]
}
```

## Policy System

### General Starlark Policies

Policies are written in Starlark syntax and can reference:

- **Principal**: The PR author with their team memberships
- **Action**: The action being performed (merge, approve, etc.)
- **Resource**: The pull request being evaluated
- **Context**: PR metadata including files, labels, approvals

Example policy:

```python
# Require security review for sensitive file changes
def evaluate(context):
    principal = context["principal"]
    resource = context["resource"]

    # Check for sensitive files
    sensitive_files = [f for f in resource["files"]
                      if any(pattern in f["filename"]
                            for pattern in [".env", "secrets", "password"])]

    if sensitive_files and "security-team" not in principal["teams"]:
        return {
            "decision": "deny",
            "reason": "Sensitive files require security team review"
        }

    return {"decision": "allow"}
```

### Directory-Specific Rules

YAML configuration files can define rules that are automatically converted to Starlark policies. Rules support inheritance and can be defined at multiple levels:

```yaml
# mushu.yaml - Project root rules
rules:
  - name: "security-review"
    path: "**"
    conditions:
      - type: "sensitive-file"
        patterns: [".env", "secrets", "password", "key"]
        require_teams: ["security-team"]
        approvers:
          security-team: 1

      - type: "auth-file"
        patterns: ["auth", "login", "permission", "rbac"]
        require_teams: ["security-team"]
        approvers:
          security-team: 1

  - name: "large-changes"
    path: "**"
    conditions:
      - type: "file-change"
        max_changes: 500
        require_teams: ["senior-backend", "platform-team"]
        approvers:
          senior-backend: 1
          platform-team: 1

  - name: "infrastructure"
    path: "infra/**"
    conditions:
      - type: "file-change"
        patterns: ["*.tf", "*.yaml", "*.yml"]
        require_teams: ["platform-team"]
        approvers:
          platform-team: 2

  - name: "binary-files"
    path: "**"
    conditions:
      - type: "binary-file"
        extensions: [".exe", ".dll", ".so", ".bin"]
        action: "forbid"

# Project-specific rules inherit from parent directories
# src/backend/mushu.yaml
rules:
  - name: "backend-security"
    path: "**"  # Relative to src/backend/
    inherit: true  # Inherit from parent rules
    conditions:
      - type: "file-change"
        pattern: "*.go"
        max_changes: 200
        require_teams: ["senior-backend"]
        approvers:
          senior-backend: 1
      - type: "database-schema"
        patterns: ["migrations/**", "schema/**"]
        require_teams: ["senior-backend", "database-team"]
        approvers:
          senior-backend: 1
          database-team: 1

# src/frontend/mushu.yaml
rules:
  - name: "frontend-quality"
    path: "**"  # Relative to src/frontend/
    inherit: true
    conditions:
      - type: "file-change"
        pattern: "*.tsx"
        max_changes: 100
        require_teams: ["frontend-team"]
        approvers:
          frontend-team: 1
      - type: "dependency-update"
        patterns: ["package.json", "yarn.lock", "package-lock.json"]
        require_teams: ["frontend-team", "security-team"]
        approvers:
          frontend-team: 1
          security-team: 1
```

### Policy Generation

Directory-specific rules are converted to Starlark policies with inheritance and approval requirements:

```python
# Generated from inherited rules (root + project-specific)
def evaluate(context):
    principal = context["principal"]
    resource = context["resource"]
    reviews = context["reviews"]  # PR reviews and approvals

    violations = []
    approval_requirements = {}

    # Security review rule (inherited from root)
    sensitive_files = [f for f in resource["files"]
                      if any(pattern in f["filename"]
                            for pattern in [".env", "secrets", "password", "key"])]

    if sensitive_files:
        if "security-team" not in principal["teams"]:
            violations.append("Sensitive files require security team membership")
        else:
            # Check approval requirements
            approval_requirements["security-team"] = 1

    # Backend security rule (project-specific)
    backend_files = [f for f in resource["files"]
                    if f["filename"].startswith("src/backend/") and
                       f["filename"].endswith(".go") and
                       f["changes"] > 200]

    if backend_files:
        if "senior-backend" not in principal["teams"]:
            violations.append("Backend changes require senior-backend team membership")
        else:
            approval_requirements["senior-backend"] = 1

    # Database schema rule (project-specific)
    schema_files = [f for f in resource["files"]
                   if any(pattern in f["filename"]
                         for pattern in ["migrations/", "schema/"])]

    if schema_files:
        required_teams = ["senior-backend", "database-team"]
        missing_teams = [team for team in required_teams if team not in principal["teams"]]
        if missing_teams:
            violations.append(f"Database schema changes require: {', '.join(missing_teams)}")
        else:
            approval_requirements["senior-backend"] = 1
            approval_requirements["database-team"] = 1

    # Check approval requirements
    for team, required_count in approval_requirements.items():
        team_approvals = [r for r in reviews
                         if r["state"] == "APPROVED" and team in r["reviewer_teams"]]
        if len(team_approvals) < required_count:
            violations.append(f"Requires {required_count} approval(s) from {team}")

    if violations:
        return {
            "decision": "deny",
            "reason": "; ".join(violations),
            "approval_requirements": approval_requirements
        }

    return {"decision": "allow"}
```

## Configuration

### Main Configuration

```yaml
# config.yaml
github:
  token: "${GITHUB_TOKEN}"
  owner: "myorg"
  repo: "myrepo"
  base_url: "https://api.github.com"

teams:
  use_github_api: true
  teams_file: "teams.yaml"
  teams_dir: "teams/"

policy:
  policy_dir: "policies/"
  policy_files:
    - "policies/approval-policy.star"
    - "policies/file-policy.star"
  rules_file: "mushu.yaml" # Root rules file

logging:
  level: "info"
  format: "json"
```

### Directory Structure

```
project/
├── mushu.yaml              # Root-level rules (inherited by all subdirectories)
├── src/
│   ├── backend/
│   │   └── mushu.yaml      # Backend-specific rules (inherits from root)
│   ├── frontend/
│   │   └── mushu.yaml      # Frontend-specific rules (inherits from root)
│   └── infra/
│       └── mushu.yaml      # Infrastructure-specific rules (inherits from root)
├── teams.yaml              # Team definitions
├── teams/
│   └── security-teams.yaml # Additional team files
├── policies/
│   ├── approval-policy.star
│   ├── file-policy.star
│   └── size-policy.star
├── entities/
│   ├── users.json
│   └── actions.json
└── config.yaml
```

## API Design

### CLI Interface

```bash
# Validate a pull request
mushu validate 123

# List available policies
mushu list policies

# Validate team configuration
mushu validate teams

# Generate Starlark policy from YAML rules
mushu generate policy --rules mushu.yaml
```

### Programmatic API

```go
// Create validator
validator, err := constraints.NewValidator(cfg)
if err != nil {
    return err
}

// Validate PR
result, err := validator.Validate(ctx, validationData)
if err != nil {
    return err
}

if !result.IsValid() {
    for _, violation := range result.GetViolations() {
        log.Error("Constraint violation", "violation", violation)
    }
}
```

## Policy Rules

### Built-in Rule Types

1. **File Change Rules**

   - Maximum changes per file
   - File size limits
   - File type restrictions

2. **Team Requirements**

   - Required team membership for specific changes
   - Role-based approvals
   - Team-specific permissions

3. **Security Rules**

   - Sensitive file detection
   - Binary file restrictions
   - Authentication file changes

4. **Quality Rules**
   - Code review requirements
   - Label requirements
   - Approval thresholds

## Implementation Details

### Team Service

```go
type TeamService interface {
    // GetUserTeams returns team memberships for an external user (GitHub username)
    GetUserTeams(ctx context.Context, username string) ([]string, error)
    // GetTeamMembers returns all members of a team (external usernames)
    GetTeamMembers(ctx context.Context, teamSlug string) ([]string, error)
    // GetUserRoles is an alias for GetUserTeams for backward compatibility
    GetUserRoles(ctx context.Context, username string) ([]string, error)
}
```

### Policy Engine

```go
type PolicyEngine struct {
    policies    []string
    teamService TeamService
}

func (pe *PolicyEngine) EvaluatePR(ctx context.Context, prData *PRData) (*EvaluationResult, error)
```

### GitHub Integration

- GraphQL API for team membership
- REST API for PR data and file changes
- Rate limiting and error handling
- Authentication via GitHub tokens

## Security Considerations

### Access Control

- GitHub token scopes: `read:org`, `repo`
- Team membership verification
- Policy file integrity

### Data Protection

- No sensitive data logging
- Secure token handling
- Audit trail for policy decisions

## Performance

### Optimization Strategies

- Team membership caching
- Policy compilation caching
- Parallel file processing
- Incremental policy evaluation

### Scalability

- Support for large repositories
- Efficient file change processing
- Memory-efficient policy evaluation

## Testing Strategy

### Unit Tests

- Policy evaluation logic
- Team service functionality
- GitHub API integration
- Configuration parsing

### Integration Tests

- End-to-end PR validation
- Policy rule application
- Team resolution scenarios

### Policy Tests

- Starlark policy validation
- Rule generation testing
- Edge case handling

## Deployment

### Docker Support

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o mushu ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/mushu .
CMD ["./mushu"]
```

### CI/CD Integration

```yaml
# GitHub Actions example
- name: Validate PR
  uses: myorg/mushu-action@v1
  with:
    pr-number: ${{ github.event.pull_request.number }}
    config-file: ".github/mushu.yaml"
```

## Future Enhancements

### Planned Features

1. **Policy Templates**: Reusable policy templates for common scenarios
2. **Metrics Collection**: Policy evaluation metrics and dashboards
3. **Policy Testing**: Test framework for policy development
4. **Webhook Integration**: Real-time policy evaluation
5. **Policy Marketplace**: Community-shared policies

### Extensibility

- Plugin system for custom rule types
- Custom policy functions
- Integration with external systems
- Custom team providers

## Conclusion

Mushu provides a comprehensive, flexible system for enforcing organizational standards through Starlark policies. The combination of GitHub team integration, YAML-based configuration, and powerful Starlark policies creates a robust foundation for pull request governance while maintaining simplicity and extensibility.

The design emphasizes:

- **Simplicity**: Teams as roles, clear configuration
- **Flexibility**: Multiple policy sources, directory-specific rules
- **Security**: Comprehensive file-based policies
- **Performance**: Efficient evaluation and caching
- **Extensibility**: Plugin architecture and custom rules
