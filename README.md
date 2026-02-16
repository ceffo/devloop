# devloop

Agent-driven development workflow system.

## Overview

`devloop` is a project-agnostic tool for automating development workflows using AI agents. It replaces brittle bash scripts with a structured, queryable, crash-safe system for managing and executing development tasks.

**Key Features:**
- 🤖 **Intelligent TODO processing**: AI agent analyzes and groups TODO items into executable tasks
- 📊 **Rich metadata**: Track attempts, duration, errors, dependencies
- 🔍 **Queryable state**: Complex filtering and reporting on task status
- 💾 **Crash-safe**: Resume from any point with session management
- 📦 **Automatic archival**: Prevent context bloat by archiving completed waves
- 🎯 **Project-agnostic**: Configuration-driven for any project
- 🚀 **Single binary**: Easy distribution and deployment

## Installation

```bash
go install github.com/yourusername/devloop/cmd/devloop@latest
```

Or build from source:

```bash
git clone https://github.com/yourusername/devloop
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

See [docs/DESIGN.md](docs/DESIGN.md) for full architecture documentation.

**Storage:**
- Tasks stored in `.devloop/tasks.jsonl` (append-only JSONL)
- Configuration in `.devloop/config.json`
- Logs in `.devloop/logs/`
- Archived tasks in `.devloop/archive/`

**Workflow:**
1. TODO items → AI agent → Structured tasks
2. Tasks → AI agent execution with retries
3. Verification after each attempt
4. Auto-commit on success
5. Archive completed waves

## Commands

```bash
devloop init                    # Initialize project
devloop config show|validate    # View/validate configuration
devloop todo process FILE       # Convert TODO items to tasks
devloop run [--wave N]          # Execute tasks
devloop tasks list [--status X] # List tasks
devloop tasks show TASK_ID      # Show task details
devloop archive [--wave N]      # Archive completed wave
devloop session status|recover  # Session management
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
    "main_branch": "main"
  },
  "verification": {
    "command": "npm run build && npm test -- --run",
    "timeout_seconds": 300
  },
  "cli": {
    "tool": "claude",
    "models": {
      "simple": "claude-haiku-4-5-20251001",
      "moderate": "claude-sonnet-4-5-20250929",
      "complex": "claude-opus-4-6"
    }
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
  }
}
```

## Development Status

🚧 **This project is currently under development** 🚧

Implementation is tracked in [docs/TASKS.md](docs/TASKS.md).

## License

MIT
