# Copilot Instructions for devloop

**See `docs/AGENT_INSTRUCTIONS.md` for complete architecture, conventions, and task workflow.**

This file is a quick reference; all other instructions are in the shared guide above.

## Quick Reference

```bash
go build ./cmd/devloop        # Build binary
go test ./...                 # Run all tests
go test ./internal/config/... # Run specific package tests
just build                    # Build to bin/devloop
just test                     # Run all tests
```

## Key Points

- **Tech Stack**: Go 1.21+, Cobra CLI framework, JSONL storage
- **Storage**: Append-only JSONL in `.devloop/tasks.jsonl`, config in `.devloop/config.json`
- **Packages**: Commands, executor, storage, processor (TODO), config, agent, archiver, prompts, UI
- **Pattern**: Error-wrapping with `fmt.Errorf()`, colocated tests (`foo_test.go`), table-driven tests
- **Architecture**: See `docs/DESIGN.md` for full design; tasks organized into 7 waves in `docs/TASKS.md`

## Testing

Coverage target >80% for critical packages (config, storage, executor).

Before committing:
```bash
go test ./... && go build ./cmd/devloop
```

## Commits

Format: `<type>[(scope)]: <description>` (lowercase, imperative)
```
task 1.2: implement feature
fix(storage): handle edge case
test(config): add validation tests
```
