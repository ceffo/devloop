# CLAUDE.md

Guidance for Claude Code when working on the devloop project.

**See `docs/AGENT_INSTRUCTIONS.md` for complete architecture, conventions, and task workflow.**

## Claude-Specific Notes

- **Follow the design**: Architecture is already defined in `docs/DESIGN.md` — study it before implementing
- **Review design diagrams**: See `docs/diagrams/` for visual flows (architecture, task execution, session recovery, etc.)
- **Don't over-engineer**: Implement exactly what the task requires
- **Test as you go**: Add tests alongside implementation
- **Session context available**: This file is automatically included in your context
