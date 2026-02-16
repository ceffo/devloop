# Agent Instructions for devloop

Shared guidance for AI agents (Claude, Copilot, etc.) working on the devloop project.

## Project Overview

devloop is a Go-based agent-driven development workflow system. It replaces bash scripts with structured, queryable, crash-safe task automation.

**Tech Stack:** Go 1.21+, Cobra CLI framework, JSONL storage

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

**Tasks** are atomic units with hierarchical IDs (e.g., "1.1", "2.3") and status progression: `pending → in_progress → completed/failed/archived`

**Waves** group related tasks by implementation phase. Tasks within a wave can execute in parallel unless blocked by dependencies.

**Storage** uses append-only JSONL format:
- `.devloop/tasks.jsonl` - One task per line, rewritten on updates
- `.devloop/config.json` - Project configuration
- `.devloop/logs/` - Execution logs
- `.devloop/archive/` - Completed waves

### Package Structure

```
cmd/devloop/              # CLI entry point
internal/
  agent/                  # AI agent runner (executes bash/gh commands)
  archiver/               # Archive completed waves
  commands/               # CLI command implementations (Cobra)
  config/                 # Configuration schema & validation
  executor/               # Task execution engine with verification
  processor/              # TODO → Task conversion
  prompts/                # AI prompt templates
  storage/                # JSONL operations & in-memory task index
  ui/                     # Terminal output helpers
```

### Data Flow

**TODO Processing:**
```
TODO.md → Parse → AI Agent → JSON tasks → Validate → tasks.jsonl
```

**Task Execution:**
```
tasks.jsonl → Query pending → Generate prompt → Run agent → 
Verify → Commit (if success) → Update task → Checkpoint
```

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

## Task Implementation Workflow

Tasks are tracked in `docs/TASKS.md` organized into 7 waves. Work sequentially—later tasks depend on earlier ones.

**When implementing:**
1. Read full task description in `docs/TASKS.md`
2. Check dependencies (earlier tasks in same wave)
3. Implement according to requirements
4. Verify acceptance criteria
5. Commit with: `task X.Y: <title>` (lowercase, imperative)

**Model annotations guide complexity:**
- **haiku**: Simple changes (< 50 lines, single file)
- **sonnet**: Moderate complexity (multi-file, state management)
- **opus**: Complex logic (algorithms, workflows)

## Commit Guidelines

Use conventional commits:
```
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
- `github.com/fatih/color` - Terminal colors
- `github.com/olekukonko/tablewriter` - Table formatting
- `github.com/google/uuid` - Session IDs

## References

- **`docs/DESIGN.md`** - Full architecture documentation
- **`docs/TASKS.md`** - Implementation roadmap with task details
- **`docs/GETTING_STARTED.md`** - Setup and usage guide
