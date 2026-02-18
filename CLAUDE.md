# CLAUDE.md

Guidance for Claude Code when working on the devloop project.

**See `docs/AGENT_INSTRUCTIONS.md` for shared architecture, conventions, and task workflow.**

This file contains Claude-specific guidance; all other instructions are in the shared guide above.

## Quick Reference

- Build: `go build ./cmd/devloop`
- Test: `go test ./...`
- Test one package: `go test ./internal/config/...`
- Check tests pass before committing: `go test ./... && go build ./cmd/devloop`

## Claude-Specific Notes

- **Don't over-engineer**: Implement exactly what the task requires
- **Follow the design**: Architecture is already defined in `docs/DESIGN.md`
- **Test as you go**: Add tests alongside implementation
- **Use examples**: Reference similar code in the codebase
- **Ask when unclear**: Use comments or ask the user if requirements are ambiguous
- **Session context available**: This file is automatically included in your context
