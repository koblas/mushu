# Project Overview

This repository contains `mushu`, a small Go command line tool that provides constraints on pull request.

## Code Quality Rules

1. **Production-Ready Only**

   - All code must be **production-grade** and ready for deployment.
   - No additional edits should be necessary.

2. **No Placeholders**

   - Do **not** use placeholders without explicit confirmation.

4. **Control Flow Restrictions**

   - Avoid **recursion** unless absolutely necessary.
   - Use **simple and bounded** control structures only.
   - All loops must have a **fixed upper bound**.

6. **Scope Discipline**

   - Declare all data objects at the **smallest possible scope**.

7. **Return and Parameter Validation**

   - Calling functions must **check all return values** from non-void functions.
   - All parameters must be **validated** in the called function.

8. **Self-Review**

   - Always **evaluate the quality** of your response.
   - Rate your solution on a **scale of 1 to 10**.

9. **Ask Questions**

   - If anything is unclear, **ask questions until 100% confident**.
   - **Break down** complex tasks into small, clear steps.
   - Point out **any contradictions** in the prompt.

10. **Post-Code Analysis**
    - After delivering code, **explain its limits and strengths**.
    - Include notes on **scalability**, e.g., for handling **1M users**.


## Coding Style

- Use Go 1.24 or newer.
- Format all Go code with `gofumpt -w` before committing.
- Check code with `golangci-lint` to ensure code quality.
- Stick to the standard Go style and keep the code cross-platform.
- Keep indentation with tabs for Go code (per `.editorconfig`). Use two spaces for YAML/JSON/Markdown.
- Follow semantic commit message convention (`type: description`).

## Testing Style

- Ensure `go test ./...` runs successfully before submitting changes.
- Add or update tests whenever modifying code.
- Use the standard Go testing framework and `testify/suite` where appropriate.

## Final Advice

- Build small, composable pieces.
- Always profile for performance.
- Write tests and document edge cases.
- Treat warnings and errors seriously.

