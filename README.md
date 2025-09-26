# Mushu

Mushu is a pull request constraint system that applies Starlark policy rules to determine if pull requests can be approved. The system combines team membership data from GitHub organizations or YAML files with Starlark policies to enforce organizational standards and security requirements.

## Features

- **Starlark Policy Engine**: Uses Starlark for flexible policy evaluation
- **Rule Inheritance**: Hierarchical rule system with `mushu.yaml` files
- **Team Management**: Supports both YAML and GitHub API team sources
- **Approval Requirements**: Team-specific approval counts
- **GitHub Integration**: Full GitHub API integration for PR data
- **CLI Interface**: Easy-to-use command-line interface

## Installation

```bash
go build -o mushu ./cmd/main.go
```

## Configuration

Create a `config.yaml` file:

```yaml
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
  rules_file: "mushu.yaml"

logging:
  level: "info"
  format: "json"
```

## Team Configuration

Define teams in `teams.yaml`:

```yaml
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
```

## Rules Configuration

Create `mushu.yaml` files in your project directories:

```yaml
rules:
  - name: "security-review"
    path: "**"
    conditions:
      - type: "sensitive-file"
        patterns: [".env", "secrets", "password", "key"]
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
```

## Usage

### Validate a Pull Request

```bash
# Set GitHub token
export GITHUB_TOKEN="your_token_here"

# Validate PR #123
./mushu validate 123
```

### List Available Policies

```bash
./mushu list policies
```

### List Available Teams

```bash
./mushu list teams
```

### Generate Starlark Policy

```bash
./mushu generate policy --rules mushu.yaml
```

## Rule Types

### File Change Rules

- **sensitive-file**: Detects sensitive files (secrets, passwords, etc.)
- **file-change**: Enforces change limits and team requirements
- **binary-file**: Restricts binary file uploads
- **auth-file**: Requires security review for authentication changes

### Approval Requirements

Each rule can specify approval requirements:

```yaml
approvers:
  security-team: 1 # Requires 1 approval from security-team
  platform-team: 2 # Requires 2 approvals from platform-team
```

## Rule Inheritance

Rules inherit from parent directories:

```
project/
├── mushu.yaml              # Root rules
├── src/
│   ├── backend/
│   │   └── mushu.yaml      # Inherits from root + backend-specific
│   └── frontend/
│       └── mushu.yaml      # Inherits from root + frontend-specific
```

## Architecture

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

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o mushu ./cmd/main.go
```

### Dependencies

- `github.com/google/go-github/v60` - GitHub API client
- `github.com/starlark-go/starlark` - Starlark interpreter
- `github.com/spf13/cobra` - CLI framework
- `gopkg.in/yaml.v3` - YAML parsing

## License

MIT License
