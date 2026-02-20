# Agent Instructions for devloop

> **Deprecated:** This file is superseded by **`CLAUDE.md`** at the project root. All agent guidance, conventions, and workflow are now maintained there. This file is kept for historical reference only.

---

Shared guidance for AI agents (Claude, Copilot, etc.) working on the devloop project.

## Before Starting Work

**Read these first:**

1. **`docs/DESIGN.md`** - Complete architecture and design decisions
2. **`docs/diagrams/`** - Visual flows for:
   - Task execution pipeline
   - Session recovery mechanism
   - Data flow through the system

Understand the design and architecture before implementing any changes. This prevents over-engineering and ensures consistency with the system design.

## Project Overview

devloop is a Go-based agent-driven development workflow system. It replaces bash scripts with structured, queryable, crash-safe task automation.

**Tech Stack:** Go 1.21+, Cobra CLI framework, JSONL storage

## AI CLI Tools

- **Claude**: `claude --model MODEL --dangerously-skip-permissions -p PROMPT`
- **Copilot**: `copilot --model MODEL --allow-all -p PROMPT`

## Build, Test, and Commands

```bash
# Build
go build ./cmd/devloop        # Build binary to ./devloop
just build                    # Build to bin/devloop

# Test
go test ./...                 # Run all tests
go test ./internal/config/... # Run specific package tests
just test                     # Run all tests

# Run without building
go run ./cmd/devloop [command]

# Dependencies
go mod tidy                   # Clean dependencies
```

## Architecture

See `docs/DESIGN.md` for full architecture documentation.

### Core Concepts

**Tasks** are atomic units with IDs (e.g., "DEV-123", "1.1") and status progression: `pending → in_progress → completed/failed/archived`

**Storage** uses append-only JSONL format:

- `.devloop/tasks.jsonl` - One task per line, rewritten on updates
- `.devloop/config.json` - Project configuration
- `.devloop/logs/` - Execution logs
- `.devloop/archive/` - Completed tasks

### Package Structure

```text
cmd/devloop/              # CLI entry point
internal/
  agent/                  # AI agent runner (executes bash/gh commands)
  archiver/               # Archive completed tasks
  commands/               # CLI command implementations (Cobra)
  config/                 # Configuration schema & validation
  executor/               # Task execution engine with verification
  processor/              # TODO → Task conversion
  prompts/                # AI prompt templates
  storage/                # JSONL operations & in-memory task index
  ui/                     # Terminal output helpers
```

### Data Flow

**TODO Processing:** (see `docs/diagrams/todo-processing.md`)

TODO.md → Parse → AI Agent → JSON tasks → Validate → tasks.jsonl

**Task Execution:** (see `docs/diagrams/task-execution.md`)

tasks.jsonl → Query pending → Generate prompt → Run agent →
Verify → Commit (if success) → Update task → Checkpoint

**Session Recovery:** (see `docs/diagrams/session-recovery.md`)

Load session → Find checkpoint → Query remaining tasks → Resume execution

## Code Conventions

### Go Standards

- **Naming**: PascalCase for exports, camelCase for private
- **Errors**: Always return errors, wrap with context: `fmt.Errorf("context: %w", err)`
- **No panics**: Use error returns instead
- **No globals**: Pass config/state explicitly via function parameters

### Testing

- **Colocated tests**: `foo.go` → `foo_test.go`
- **Table-driven tests** for multiple cases
- **Coverage target**: >80% for critical packages (config, storage, executor)
- **Mock external dependencies**: AI agents, git commands, filesystem ops

### Documentation

- **Godoc on all exports**: Explain "why" not "what"
- **Struct tags**: Use `json:"field_name"` for all config/storage structs

### Common Patterns

**Error handling:**

```go
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}
```

**JSONL operations:**

```go
encoder := json.NewEncoder(file)
for _, item := range items {
    if err := encoder.Encode(item); err != nil {
        return fmt.Errorf("encode failed: %w", err)
    }
}
```

**Struct validation:**

```go
func (c *Config) Validate() error {
    if c.Project.Path == "" {
        return errors.New("project.path is required")
    }
    return nil
}
```

**Cobra commands:**

```go
var myCmd = &cobra.Command{
    Use:   "mycommand",
    Short: "Brief description",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
        return nil
    },
}
```

## Design Principles

1. **Simple over clever**: JSONL over database, explicit over magical
2. **Type-safe**: Leverage Go's type system, avoid `interface{}`
3. **Testable**: Pure functions, dependency injection
4. **Minimal dependencies**: Prefer stdlib when sufficient

### Key Design Decisions

**JSONL vs SQLite:**

- Debuggable with `cat`/`grep`
- Git-friendly
- No query complexity needed (<1000 tasks)
- In-memory indexing provides fast lookups

**Project-agnostic design:**

- Configuration-driven (no hardcoded paths)
- Can be distributed as standalone tool
- Works with any tech stack

## Knowledge Layer

The devloop knowledge layer captures expertise and learnings from task execution to improve future work.

### Querying Knowledge

Query the knowledge base to learn from past task implementations:

```bash
# Query expertise for a specific domain
devloop knowledge query mypackage

# Search for a specific topic
devloop knowledge search "authentication flow"

# Show knowledge status
devloop knowledge status

# Compact the knowledge base
devloop knowledge compact

# Run diagnostics
devloop knowledge doctor

# View knowledge diff from a git ref
devloop knowledge diff HEAD~1

# Prime knowledge for specified domains
devloop knowledge prime mypackage otherpackage
```

The knowledge backend is configured in `.devloop/config.json` under `knowledge.backend` (default: `"mulch"`).

### Automatic Prompt Injection

Knowledge is automatically injected into agent prompts during task execution via the `Prime()` function. This provides relevant context without explicit querying—agents receive priming data that enhances prompt awareness of related work.

The priming mechanism:
- Retrieves cached domain expertise via `devloop knowledge prime`
- Formats results as markdown for integration into prompts
- Returns empty string gracefully if mulch is unavailable (non-fatal)
- Respects configured token budgets to keep prompts manageable

### Recording Learnings

At task completion, agents are expected to record domain knowledge and insights:

```bash
# From within task execution
devloop knowledge query <domain>  # Review existing knowledge before recording
mulch record <domain> <type> <content>  # Store new knowledge
```

**Expectations:**
- Record learnings for the packages/domains modified in the task
- Capture architectural decisions, patterns, and gotchas
- Include examples or snippets for recurring problems
- Document integration points or dependencies discovered

**Example:**
After implementing an authentication feature, record learnings:
```bash
mulch record auth implementation "JWT token validation for REST endpoints..."
mulch record auth pattern "Use middleware for auth checks..."
```

The knowledge base is version-controlled alongside code, allowing team members to benefit from collective insights.

## Task Implementation Workflow

Tasks are tracked in `docs/TASKS.md`. Work sequentially—later tasks may depend on earlier ones.

**When implementing:**

1. Read full task description in `docs/TASKS.md`
2. Check dependencies (blocked_by field)
3. Implement according to requirements
4. Verify acceptance criteria
5. **Record learnings** via `devloop knowledge` commands (see Knowledge Layer section)
6. Commit with: `task <ID>: <title>` (lowercase, imperative)

**Model annotations guide complexity:**

- **haiku**: Simple changes (< 50 lines, single file)
- **sonnet**: Moderate complexity (multi-file, state management)
- **opus**: Complex logic (algorithms, workflows)

## Commit Guidelines

Use conventional commits:

```text
task 1.2: implement configuration data structures
fix(storage): handle missing JSONL file gracefully
test(config): add validation tests
docs: update architecture diagram
```

**Format:** `<type>[(scope)]: <description>`

- Types: `task`, `feat`, `fix`, `test`, `docs`, `refactor`, `chore`
- Scope: package name or area
- Description: lowercase, imperative, no period

**Before committing:**

```bash
go test ./... && go build ./cmd/devloop
```

## Dependencies

**Current:**

- `github.com/spf13/cobra` - CLI framework
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `github.com/charmbracelet/bubbles` - Terminal UI components
- `github.com/charmbracelet/bubbletea` - TUI framework for interactive features
- `github.com/charmbracelet/glamour` - Markdown rendering
- `github.com/olekukonko/tablewriter` - Table formatting
- `github.com/google/uuid` - Session IDs

## References

- **`docs/DESIGN.md`** - Full architecture documentation
- **`docs/TASKS.md`** - Implementation roadmap with task details
- **`docs/GETTING_STARTED.md`** - Setup and usage guide
- **`docs/diagrams/`** - Visual flow diagrams (Mermaid)
