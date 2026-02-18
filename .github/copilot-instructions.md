# Copilot Instructions for devloop

**See `docs/AGENT_INSTRUCTIONS.md` for complete architecture, conventions, and task workflow.**

## Quick Reference

```bash
go build ./cmd/devloop        # Build binary
go test ./...                 # Run all tests
go test ./internal/config/... # Run specific package tests
just build                    # Build to bin/devloop
just test                     # Run all tests
```

## Testing Before Commit

```bash
go test ./... && go build ./cmd/devloop
```
