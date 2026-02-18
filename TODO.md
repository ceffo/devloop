# TODO

- improve agentic development: clean out CLAUDE.md and copilot-instructions.md files, they should only forward to docs/AGENT_INSTRUCTIONS.md. Add to instructions to study design and design diagrams.
- support a --dry-run flag for all commands
- add a context from to all executors. capture termination signals at the main level, put the cancellation in the context, make sure all commands can gracefully abort.
- add golangci linting. create a sensible configuration. fix linting issues uncovered. 
- setup github workflow to build, check linting and test. put them on separate steps, dependent on the previous.
- clean out legacy config items and task migration

