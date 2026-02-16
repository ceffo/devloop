# CLAUDE.md

This file provides guidance to Claude Code when working on the devloop project.

## Project Overview

devloop is a Go-based agent-driven development workflow system. It replaces bash scripts with structured, queryable, crash-safe task automation.

**Tech Stack:** Go 1.21+, Cobra CLI framework, JSONL storage

## Commands

```bash
go build ./cmd/devloop        # Build binary
go test ./...                 # Run all tests
go test ./internal/config/... # Run specific package tests
go run ./cmd/devloop          # Run without building
go mod tidy                   # Clean dependencies
```

## Architecture

See `docs/DESIGN.md` for full architecture documentation.

**Key principles:**
- Simple over clever: JSONL over database, explicit over magical
- Type-safe: Leverage Go's type system
- Testable: Pure functions, dependency injection
- Minimal dependencies: Prefer stdlib when sufficient

## Code Conventions

- **Naming**: Go standard (PascalCase exports, camelCase private)
- **Errors**: Return errors, don't panic. Wrap with context: `fmt.Errorf("context: %w", err)`
- **Testing**: Colocated tests (`foo.go` → `foo_test.go`), use table-driven tests
- **Comments**: Godoc on all exports, explain "why" not "what"
- **Structs**: Use struct tags for JSON: `json:"field_name"`
- **No globals**: Pass config/state explicitly

## File Organization

```
cmd/devloop/              # CLI entry point
internal/
  config/                # Configuration schema & loading
  storage/               # Task data structures & JSONL operations
  executor/              # Task execution engine
  processor/             # TODO processing
  archiver/              # Archival system
  prompts/               # AI prompt templates
  commands/              # CLI command implementations
  ui/                    # Terminal output helpers
```

## Implementation Strategy

Tasks are tracked in `docs/TASKS.md` organized into waves:
- **Wave 1**: Core infrastructure (config, storage)
- **Wave 2**: CLI commands foundation
- **Wave 3**: TODO processing
- **Wave 4**: Execution engine
- **Wave 5**: Archival
- **Wave 6**: Session management & polish
- **Wave 7**: Testing & release

**Work sequentially** - later tasks depend on earlier ones.

## Task Guidelines

Each task has:
- **Model annotation**: Complexity-based model selection
- **Requirements**: Specific implementation details
- **Acceptance criteria**: Testable outcomes

**When implementing a task:**
1. Read the full task description in `docs/TASKS.md`
2. Understand dependencies (earlier tasks in same wave)
3. Implement according to requirements
4. Verify acceptance criteria
5. Commit with: `task X.Y: <title>` (lowercase, imperative)

## Testing Requirements

- **Unit tests required** for all packages in `internal/`
- **Coverage target**: >80% for critical packages (config, storage, executor)
- **Table-driven tests** for functions with multiple cases
- **Integration tests** in `tests/` for end-to-end workflows
- **Mock external dependencies** (AI agents, git commands)

## Common Patterns

### Error Handling
```go
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}
```

### JSONL Operations
```go
encoder := json.NewEncoder(file)
for _, item := range items {
    if err := encoder.Encode(item); err != nil {
        return fmt.Errorf("encode failed: %w", err)
    }
}
```

### Struct Validation
```go
func (c *Config) Validate() error {
    if c.Project.Path == "" {
        return errors.New("project.path is required")
    }
    // ... more checks
    return nil
}
```

### CLI Commands (Cobra)
```go
var myCmd = &cobra.Command{
    Use:   "mycommand",
    Short: "Brief description",
    Long:  "Longer description...",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
        return nil
    },
}
```

## References

- **Design Document**: `docs/DESIGN.md` - Full architecture
- **Task List**: `docs/TASKS.md` - Implementation roadmap
- **Implementation Plan**: (in Modal project) - Original design rationale

## Key Design Decisions

**Why Go?**
- Single binary distribution
- Strong typing catches errors at compile time
- Excellent standard library (json, exec, filepath)
- Fast execution, low memory

**Why JSONL over SQLite?**
- Debuggable with cat/grep
- Git-friendly
- No query complexity needed (<1000 tasks)
- In-memory indexing provides fast queries

**Why Project-Agnostic?**
- Reusable across all projects
- Configuration-driven
- No hardcoded paths or assumptions
- Can be distributed as standalone tool

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

**When to commit:**
- After completing each task
- After fixing a bug found during implementation
- Before switching to a different task/area

Run tests before committing:
```bash
go test ./... && go build ./cmd/devloop
```

## Dependencies

Current:
- `github.com/spf13/cobra` - CLI framework
- `github.com/fatih/color` - Terminal colors
- `github.com/olekukonko/tablewriter` - Table formatting

To be added as needed:
- `github.com/google/uuid` - Session IDs (Task 4.3)
- `github.com/schollz/progressbar/v3` - Progress indicators (Task 6.4)

## Model Selection for Tasks

Tasks are annotated with recommended models:
- **haiku**: Simple, straightforward changes (< 50 lines, single file)
- **sonnet**: Moderate complexity (multi-file, state management)
- **opus**: Complex logic (algorithms, intricate workflows)

These guide the dev-loop.sh automation when executing tasks.

## Notes for AI Assistants

- **Don't over-engineer**: Implement exactly what the task requires
- **Follow the design**: Architecture is already defined in docs/DESIGN.md
- **Test as you go**: Don't wait for Wave 7 to add tests
- **Use examples**: Reference similar code in the codebase
- **Ask when unclear**: Use comments or ask the user if requirements are ambiguous
