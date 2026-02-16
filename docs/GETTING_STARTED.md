# Getting Started with devloop Development

## Prerequisites

- Go 1.21 or higher
- Access to Claude CLI (for dev-loop automation)
- Modal project (for using dev-loop.sh during development)

## Development Approach

This project uses **bootstrap development**: we use the existing Modal `dev-loop.sh` script to build the devloop tool, which will eventually replace dev-loop.sh!

## Project Structure

```
devloop/
├── cmd/devloop/          # CLI entry point
├── internal/             # Core packages
│   ├── config/           # Configuration management
│   ├── storage/          # Task storage (JSONL)
│   ├── executor/         # Task execution engine
│   ├── processor/        # TODO processing
│   ├── archiver/         # Archival system
│   ├── prompts/          # AI prompt templates
│   ├── commands/         # CLI commands
│   └── ui/               # Terminal UI helpers
├── docs/                 # Documentation
│   ├── TASKS.md          # Implementation roadmap
│   ├── DESIGN.md         # Architecture documentation
│   └── GETTING_STARTED.md # This file
├── scripts/              # Helper scripts
└── .devloop/             # Dev-loop state and config
```

## Quick Start

### 1. Build Current State

```bash
cd /home/moncef/code/devloop
go build ./cmd/devloop
./devloop
```

Current output is a placeholder. Implementation tracked in `docs/TASKS.md`.

### 2. Run Tests

```bash
go test ./...
```

### 3. Execute Tasks with Modal dev-loop.sh

The project is configured to use Modal's `dev-loop.sh` for automated task execution:

```bash
# Run all tasks in Wave 1
cd /home/moncef/code/devloop
~/code/modal/scripts/dev-loop.sh 1

# Run specific task range in Wave 2
~/code/modal/scripts/dev-loop.sh 2 2.1 2.3

# Or use the convenience wrapper
./scripts/dev-loop-run.sh 1
```

The dev-loop.sh script will:
1. Read tasks from `docs/TASKS.md`
2. Select appropriate model based on complexity
3. Execute each task with AI agent
4. Run verification: `go test ./... && go build ./cmd/devloop`
5. Auto-commit on success

## Implementation Waves

Tasks are organized into 7 waves (see `docs/TASKS.md`):

1. **Wave 1**: Core Infrastructure (config, storage)
2. **Wave 2**: CLI Commands Foundation
3. **Wave 3**: TODO Processing System
4. **Wave 4**: Execution Engine
5. **Wave 5**: Archival System
6. **Wave 6**: Session Management & Polish
7. **Wave 7**: Testing & Release

**Work sequentially** - later waves depend on earlier ones.

## Manual Development

If you prefer manual development instead of using dev-loop.sh:

### Implementing a Task

1. **Read the task** in `docs/TASKS.md`
2. **Check dependencies** - ensure prerequisite tasks are complete
3. **Implement** following requirements and acceptance criteria
4. **Test** your changes:
   ```bash
   go test ./...
   go build ./cmd/devloop
   ```
5. **Commit** using format: `task X.Y: <title>`
   ```bash
   git add -A
   git commit -m "task 1.2: implement configuration data structures"
   ```

### Example: Task 1.1

```bash
# Add dependencies
go get github.com/spf13/cobra@latest
go get github.com/fatih/color@latest
go get github.com/olekukonko/tablewriter@latest

# Verify build
go build ./cmd/devloop

# Commit
git add -A
git commit -m "task 1.1: project setup and dependencies"
```

## Configuration

The `.devloop/config.json` file configures how dev-loop.sh runs tasks:

```json
{
  "verification": {
    "command": "go test ./... && go build ./cmd/devloop"
  },
  "cli": {
    "models": {
      "simple": "claude-haiku-4-5-20251001",
      "moderate": "claude-sonnet-4-5-20250929",
      "complex": "claude-opus-4-6"
    }
  }
}
```

## Architecture References

- **Full Design**: `docs/DESIGN.md`
- **Coding Guidelines**: `CLAUDE.md`
- **Task List**: `docs/TASKS.md`

## Tips for Contributors

1. **Follow Go conventions** - PascalCase exports, camelCase private
2. **Write tests** - Don't wait for Wave 7
3. **Keep it simple** - KISS principle applies
4. **Document exports** - Godoc comments on all exported functions
5. **Handle errors** - Return errors, don't panic
6. **Test before committing** - Run `go test ./... && go build ./cmd/devloop`

## Common Commands

```bash
# Build
go build ./cmd/devloop

# Run without building
go run ./cmd/devloop

# Test all packages
go test ./...

# Test specific package
go test ./internal/config/...

# Test with coverage
go test -cover ./...

# Format code
go fmt ./...

# Clean dependencies
go mod tidy

# Run dev-loop automation
./scripts/dev-loop-run.sh 1
```

## Progress Tracking

Check implementation progress:

```bash
# View task list
cat docs/TASKS.md

# View dev-loop progress (after running automation)
cat .dev-loop-progress

# View recent commits
git log --oneline -10
```

## Troubleshooting

### Build fails
```bash
go mod tidy
go build ./cmd/devloop
```

### Tests fail
```bash
# Run specific failing test
go test -v ./internal/config -run TestLoadConfig

# Show test coverage
go test -cover ./...
```

### dev-loop.sh not found
```bash
# Ensure Modal project exists
ls ~/code/modal/scripts/dev-loop.sh

# Or use absolute path
/home/moncef/code/modal/scripts/dev-loop.sh 1
```

## Next Steps

1. **Read** `docs/DESIGN.md` to understand architecture
2. **Read** `CLAUDE.md` for coding guidelines
3. **Review** `docs/TASKS.md` Wave 1 tasks
4. **Execute** Wave 1 with dev-loop.sh or manually
5. **Iterate** through remaining waves

Happy building! 🚀
