# CLAUDE.md

Agent guidance for the devloop project. This is the single canonical source — all agents (Claude, Copilot, etc.) should treat this as truth.

## Project Overview

devloop is a Go-based agent-driven development workflow system. It replaces bash scripts with structured, queryable, crash-safe task automation.

**Tech Stack:** Go 1.21+, Cobra CLI framework, JSONL storage

## Before Starting Work

**Read these first:**

1. **`docs/DESIGN.md`** — Complete architecture and design decisions
2. **`docs/diagrams/`** — Visual flows for task execution pipeline, session recovery, and data flow

Understand the design before implementing. This prevents over-engineering and ensures consistency.

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

**Tasks** are tracked in **beads** (`bd`). IDs use the `DEV-NNN` format. Status progression: `pending → in_progress → completed`

**devloop runtime files** (not task storage — that's beads):

- `.devloop/config.json` — Project configuration
- `.devloop/logs/` — Execution logs

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

**Task Execution:** (see `docs/diagrams/task-execution.md`)

beads (bd ready) → Query pending → Generate prompt → Run agent →
Verify → Commit (if success) → bd close → Checkpoint

**Session Recovery:** (see `docs/diagrams/session-recovery.md`)

Load session → Find checkpoint → Query remaining tasks (beads) → Resume execution

## Beads (Issue Tracking)

This project uses **bd** (beads) for issue tracking.

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```

## Task Implementation Workflow

Use `bd ready` to find work, `bd show <id>` to read task details.

**When implementing:**

1. `bd ready` — find available tasks
2. `bd show <id>` — read full task description and dependencies
3. `bd update <id> --status in_progress` — claim the task
4. Implement according to requirements
5. Verify acceptance criteria
6. **Record learnings** via `devloop knowledge` commands (see Knowledge Layer section)
7. Commit with: `task <ID>: <title>` (lowercase, imperative)
8. `bd close <id>` — mark complete

**Model annotations guide complexity:**

- **haiku**: Simple changes (< 50 lines, single file)
- **sonnet**: Moderate complexity (multi-file, state management)
- **opus**: Complex logic (algorithms, workflows)

## Knowledge Layer

The devloop knowledge layer captures expertise and learnings from task execution to improve future work.

### Querying Knowledge

```bash
devloop knowledge query mypackage        # Query expertise for a domain
devloop knowledge search "topic"         # Search for a specific topic
devloop knowledge status                 # Show knowledge status
devloop knowledge compact                # Compact the knowledge base
devloop knowledge doctor                 # Run diagnostics
devloop knowledge diff HEAD~1            # View knowledge diff from a git ref
devloop knowledge prime mypackage other  # Prime knowledge for specified domains
```

The knowledge backend is configured in `.devloop/config.json` under `knowledge.backend` (default: `"mulch"`).

### Automatic Prompt Injection

Knowledge is automatically injected into agent prompts during task execution via the `Prime()` function. Agents receive priming data that enhances prompt awareness of related work.

### Recording Learnings

At task completion, record domain knowledge and insights:

```bash
devloop knowledge query <domain>          # Review existing knowledge first
mulch record <domain> <type> <content>    # Store new knowledge
```

**Expectations:**
- Record learnings for the packages/domains modified
- Capture architectural decisions, patterns, and gotchas
- Include examples or snippets for recurring problems
- Document integration points or dependencies discovered

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

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- `github.com/charmbracelet/lipgloss` — Terminal styling
- `github.com/charmbracelet/bubbles` — Terminal UI components
- `github.com/charmbracelet/bubbletea` — TUI framework for interactive features
- `github.com/charmbracelet/glamour` — Markdown rendering
- `github.com/olekukonko/tablewriter` — Table formatting
- `github.com/google/uuid` — Session IDs

## References

- **`docs/DESIGN.md`** — Full architecture documentation
- **`docs/GETTING_STARTED.md`** — Setup and usage guide
- **`docs/diagrams/`** — Visual flow diagrams (Mermaid)

## Claude-Specific Notes

- Session context automatically included in your workspace
- Use tools efficiently: parallel calls, batch edits, suppress verbose output
- Trust your analysis: if a sub-agent or tool completes successfully, verify with spot-checks rather than re-running
