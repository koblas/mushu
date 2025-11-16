# Mushu

### A pull request policy tool you can trust.

Mushu automates pull request validation by enforcing customizable rules around titles, approvals, and file permissions. Built with transparency and developer trust at its core.

### Why Mushu?

Your CI/CD pipeline is critical infrastructure. The tools that gate your deployments should be:

- Transparent - Every line of code is open for inspection
- Privacy-respecting - Your PR data stays in your infrastructure
- Community-driven - Governed by engineers, for engineers
- Simple - Clear configuration, predictable behavior

Mushu validates pull requests against policies you define, ensuring consistency across your organization without compromising on trust or transparency.

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
  format: "console"  # or "json"
```

## Team Configuration

Define teams in `teams.yaml`:

```yaml
teams:
  senior-backend:
    description: "Senior backend developers with merge permissions"
    members:
      - "alice"
      - "eve"

  security-team:
    description: "Security and compliance team"
    members:
      - "bob"
      - "eve"

  platform-team:
    description: "Platform and infrastructure team"
    members:
      - "alice"
      - "bob"
      - "charlie"

  database-team:
    description: "Database and schema management team"
    members:
      - "alice"
      - "frank"
```

## Rules Configuration

Create `mushu.yaml` files in your project to define policy rules. Each rule specifies a Starlark policy to execute with optional conditions:

```yaml
rules:
  # Example 1: Semantic PR Title Validation
  - name: "semantic-title"
    description: "Enforce conventional commit format for PR titles"
    args:
      pattern: "^(feat|fix|docs|style|refactor|perf|test|chore)(\\(.+\\))?: .{10,50}$"
      message: "PR title must follow conventional commits format"

  # Example 2: Conditional Rule with Branch and File Filters
  - name: "print"
    description: "Debug policy that prints context"
    when:
      branch:
        target: ["main", "develop"]
        source: ["feature/*", "bugfix/*"]
      file:
        include: ["src/**", "lib/**"]
        exclude: ["src/experimental/**"]
      contributor:
        users: ["alice", "bob"]
        teams: ["dev-team", "qa-team"]
```

### Rule Structure

- **name** (required): Name of the Starlark policy to execute (e.g., "semantic-title", "print")
- **description** (optional): Human-readable description of the rule
- **when** (optional): Conditions that must be met for the rule to execute:
  - **branch**: Filter by source and target branches (supports glob patterns)
  - **file**: Filter by included/excluded file patterns (supports glob patterns)
  - **contributor**: Filter by specific users or teams
- **args** (optional): Key-value arguments passed to the Starlark policy
- **files** (optional): File patterns this rule applies to
- **exclude** (optional): File patterns to exclude from this rule
- **verbose** (optional): Enable detailed logging

## Usage

### Validate a Pull Request

```bash
# Set GitHub token
export GITHUB_TOKEN="your_token_here"

# Validate PR #123 using config.yaml
./mushu -config=config.yaml validate 123

# Or specify options directly
./mushu -github-token=$GITHUB_TOKEN -github-owner=myorg -github-repo=myrepo validate 123
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
./mushu generate policy
```

### Global Flags

```bash
-config string          Path to configuration file
-github-owner string    GitHub repository owner
-github-repo string     GitHub repository name
-github-token string    GitHub API token
-log-format string      Log format (json, text)
-log-level string       Log level (debug, info, warn, error)
```

## Built-in Policies

Mushu includes built-in Starlark policies you can reference by name:

### semantic-title

Validates PR titles follow conventional commit format.

**Default Pattern**: `^(feat|fix|docs|style|refactor|perf|test|chore)(\(.+\))?[!:] .{10,70}$`

**Arguments**:
- `pattern` (optional): Custom regex pattern for title validation
- `message` (optional): Custom error message when validation fails

**Example**:
```yaml
rules:
  - name: "semantic-title"
    args:
      pattern: "^(feat|fix|docs):\\s.{10,50}$"
      message: "Title must start with feat:, fix:, or docs: and be 10-50 chars"
```

### print

Debug policy that prints the PR context during evaluation. Useful for testing and understanding the data available to policies.

**Example**:
```yaml
rules:
  - name: "print"
    description: "Debug: print PR context"
```

## Writing Custom Starlark Policies

Create `.star` files in your `policy_dir` to define custom policies:

```python
# policies/custom-policy.star
load("re.star", "re")

def evaluate(context, **kwargs):
    """
    Evaluate custom policy.

    Args:
        context (dict): Contains 'principal', 'resource', and 'reviews'
        **kwargs: Custom arguments from rule configuration

    Returns:
        dict: {'decision': 'allow'|'deny', 'reason': str, 'approval_requirements': dict}
    """
    pr = context["resource"]
    author = context["principal"]
    reviews = context["reviews"]

    # Your policy logic here
    if pr.get("draft", False):
        return deny(message = "Draft PRs cannot be merged")

    return allow()
```

Use built-in helper functions:
- `allow()`: Returns an allow decision
- `deny(message = "reason")`: Returns a deny decision with a reason

## Complete Example

Here's a complete example showing how all components work together:

**Directory structure:**
```
myproject/
├── config.yaml         # Main configuration
├── teams.yaml          # Team definitions
├── mushu.yaml          # Root rules
└── policies/
    └── approval.star   # Custom policy
```

**config.yaml:**
```yaml
github:
  token: "${GITHUB_TOKEN}"
  owner: "myorg"
  repo: "myrepo"
  base_url: "https://api.github.com"

teams:
  use_github_api: false
  teams_file: "teams.yaml"

policy:
  rules_file: "mushu.yaml"
  policy_dir: "policies/"

logging:
  level: "info"
  format: "console"
```

**teams.yaml:**
```yaml
teams:
  backend-team:
    description: "Backend developers"
    members:
      - "alice"
      - "bob"

  security-team:
    description: "Security team"
    members:
      - "eve"
```

**mushu.yaml:**
```yaml
rules:
  # All PRs must have semantic titles
  - name: "semantic-title"

  # Backend changes require team review
  - name: "approval"
    when:
      file:
        include: ["src/backend/**", "api/**"]
      branch:
        target: ["main"]
    args:
      required_teams: ["backend-team"]
      min_approvals: 2
```

**policies/approval.star:**
```python
def evaluate(context, required_teams = [], min_approvals = 1, **kwargs):
    reviews = context["reviews"]
    pr = context["resource"]

    # Count approvals from required teams
    approved_count = 0
    for review in reviews:
        if review.get("state") == "approved":
            approved_count += 1

    if approved_count >= min_approvals:
        return allow()

    return deny(
        message = "Requires {} approvals, got {}".format(min_approvals, approved_count)
    )
```

**Run validation:**
```bash
export GITHUB_TOKEN="ghp_xxx"
./mushu -config=config.yaml validate 123
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
