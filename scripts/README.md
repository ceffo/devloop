# Dev-Loop Scripts

## Overview

This directory contains the **Universal Dev Loop** - a config-driven automation script that can work with any project that has a `.devloop/config.json` file.

## Scripts

### `dev-loop.sh`

The main config-driven dev loop script. **This is project-agnostic** and can be used in any project with proper configuration.

**Features:**
- Reads all configuration from `.devloop/config.json`
- Project-agnostic (no hardcoded paths or commands)
- Supports multiple CLI tools (claude, copilot)
- Auto-retry with error context
- Auto-commit on success
- Resume-safe via `.dev-loop-progress`
- Wave and task range filtering

**Usage:**
```bash
# Run all tasks
./scripts/dev-loop.sh

# Run all tasks in wave 1
./scripts/dev-loop.sh 1

# Run specific range in wave 2 (tasks 2.1 to 2.3)
./scripts/dev-loop.sh 2 2.1 2.3

# Use copilot instead of claude
CLI_TOOL=copilot ./scripts/dev-loop.sh 1
```

### `dev-loop-run.sh`

Convenience wrapper that runs `dev-loop.sh` from the project root.

**Usage:**
```bash
./scripts/dev-loop-run.sh 1        # Run wave 1
./scripts/dev-loop-run.sh 2 2.1 2.3  # Run tasks 2.1-2.3
```

## Configuration

The script reads from `.devloop/config.json`:

```json
{
  "project": {
    "name": "devloop"
  },
  "verification": {
    "command": "go test ./... && go build ./cmd/devloop"
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
    "auto_commit": true,
    "commit_format": "task {task_id}: {title}"
  },
  "files": {
    "tasks": "docs/TASKS.md"
  },
  "prompts": {
    "task_context_files": [
      "docs/DESIGN.md",
      "CLAUDE.md"
    ],
    "custom_instructions": "Follow conventions. Keep it simple."
  }
}
```

## How It Works

1. **Parse Config** - Load project settings from `.devloop/config.json`
2. **Parse Tasks** - Extract tasks from the configured tasks file (e.g., `docs/TASKS.md`)
3. **Filter Tasks** - Apply wave and range filters from CLI args
4. **For Each Task:**
   - Select appropriate model based on complexity metadata
   - Generate context-aware prompt with configured files
   - Run AI CLI tool (claude or copilot)
   - Execute verification command from config
   - Retry with error context if verification fails
   - Auto-commit if configured
   - Save progress
5. **Resume** - If interrupted, resume from last checkpoint

## Task Format

Tasks in `TASKS.md` should follow this format:

```markdown
### Task 1.2: Configuration data structures

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Implement configuration schema in `internal/config/`.

**Requirements:**
- Create structs
- Add validation

**Acceptance:**
- Compiles without errors
- Tests pass
```

The `**Complexity:**` metadata determines which model is used:
- `simple` → haiku (fast, cheap)
- `moderate` → sonnet (balanced)
- `complex` → opus (powerful, expensive)

## Using with Other Projects

To use this script with another project:

1. **Copy the script** to your project:
   ```bash
   cp scripts/dev-loop.sh /path/to/other-project/scripts/
   ```

2. **Create `.devloop/config.json`** in your project with:
   - Project name
   - Verification command (tests, build, lint, etc.)
   - Tasks file path
   - Context files for AI prompts
   - Model preferences

3. **Create tasks file** (e.g., `docs/TASKS.md`) with task definitions

4. **Run it**:
   ```bash
   cd /path/to/other-project
   ./scripts/dev-loop.sh 1
   ```

## Dependencies

- `bash` 4.0+
- `jq` or `python3` (for JSON parsing)
- `claude` or `copilot` CLI (whichever you configure)
- `git` (for auto-commit)

## Logs

Logs are stored in `.dev-loop-logs/`:
- `task-X.Y.log` - Full agent output for task X.Y
- `verify-X.Y.log` - Verification command output for task X.Y

## Progress Tracking

The `.dev-loop-progress` file tracks the last completed task ID. If the script is interrupted, it will resume from this checkpoint on the next run.

## Bootstrap Development

This script is being used to **bootstrap its own development**! We're using it to build the `devloop` tool, which will eventually replace this script with a proper Go binary.

**Meta moment:** The tool builds itself. 🤖
