# devloop

Agent-driven development workflow system.

## Overview

`devloop` is a project-agnostic tool for automating development workflows using AI agents. It replaces brittle bash scripts with a structured, queryable, crash-safe system for managing and executing development tasks.

**Key Features:**

- 🤖 **Intelligent TODO processing**: AI agent analyzes and groups TODO items into executable tasks
- 📊 **Rich metadata**: Track attempts, duration, errors, dependencies
- 🔍 **Queryable state**: Complex filtering and reporting on task status
- 💾 **Crash-safe**: Resume from any point with session management
- 📦 **Automatic archival**: Prevent context bloat by archiving completed tasks
- 🎯 **Project-agnostic**: Configuration-driven for any project
- 🚀 **Single binary**: Easy distribution and deployment

## Installation

```bash
go install github.com/ceffo/devloop/cmd/devloop@latest
```

Or build from source:

```bash
git clone https://github.com/ceffo/devloop
cd devloop
go build -o devloop ./cmd/devloop
```

## Quick Start

```bash
# Initialize in your project
cd /path/to/your/project
devloop init

# Process TODO items into tasks
devloop todo process .todo/TODO.md

# Run the dev loop
devloop run

# Check status
devloop tasks list
devloop session status
```

## Architecture

See [docs/DESIGN.md](docs/DESIGN.md) for full architecture documentation and [docs/diagrams/](docs/diagrams/) for visual diagrams.

**Storage:**

- Tasks stored in `.devloop/tasks.jsonl` (append-only JSONL)
- Configuration in `.devloop/config.json`
- Logs in `.devloop/logs/`
- Archived tasks in `.devloop/archive/`
- Session state in `.devloop/state/`

**Workflow:**

1. TODO items → AI agent → Structured tasks
2. Tasks → AI agent execution with retries
3. Verification after each attempt
4. Auto-commit on success (optional)
5. Archive completed tasks (optional)
6. Session checkpointing for crash recovery

## Commands

```bash
# Initialize project
devloop init

# Configuration management
devloop config show             # Display current configuration
devloop config validate         # Validate configuration file

# TODO processing
devloop todo process FILE       # Convert TODO items to tasks

# Task execution
devloop run                      # Execute tasks
devloop run [--task ID]         # Run specific task
devloop run --continue          # Resume from last checkpoint

# Task management
devloop tasks list              # List all tasks
devloop tasks list --status X   # Filter by status
devloop tasks show TASK_ID      # Show task details
devloop tasks update ID --status X  # Update task status

# Archival
devloop archive                  # Archive completed tasks
devloop archive --auto          # Auto-archive completed tasks

# Session management
devloop session status          # Show current session
devloop session recover         # Recover from crash

# Task ID migration
devloop migrate-ids             # Migrate to JIRA-style IDs
```

## Configuration

Example `.devloop/config.json`:

```json
{
  "version": "1.0",
  "project": {
    "name": "myproject",
    "path": "/home/user/code/myproject",
    "tech_stack": "React + TypeScript",
    "main_branch": "main",
    "task_id_prefix": "DEV",
    "task_id_format": "jira"
  },
  "verification": {
    "command": "npm run build && npm test -- --run",
    "timeout_seconds": 300
  },
  "cli": {
    "agents": [
      {
        "tool": "claude",
        "models": {
          "simple": "claude-haiku-4-5-20251001",
          "moderate": "claude-sonnet-4-5-20250929",
          "complex": "claude-opus-4-6"
        }
      }
    ]
  },
  "execution": {
    "max_attempts": 2,
    "halt_on_failure": true,
    "auto_commit": true
  },
  "files": {
    "prd": "docs/PRD.md",
    "tasks": "docs/TASKS.md",
    "todo": ".todo/TODO.md"
  },
  "archival": {
    "auto_archive": true
  }
}
```

## Development Status

✅ **Core features implemented!**

The system is functional with CLI commands, task execution, TODO processing, archival, and session management. See [docs/TASKS.md](docs/TASKS.md) for detailed implementation progress.

## License

MIT
