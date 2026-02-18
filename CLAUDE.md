# CLAUDE.md

Claude-specific guidance for the devloop project.

**See `docs/AGENT_INSTRUCTIONS.md` for architecture, conventions, and task workflow.**

## Claude-Specific Notes

- Session context automatically included in your workspace
- Use tools efficiently: parallel calls, batch edits, suppress verbose output
- Trust your analysis: if a sub-agent or tool completes successfully, verify with spot-checks rather than re-running
