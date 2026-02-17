# devloop Architecture Design

## Overview

devloop is a project-agnostic agent-driven development workflow system. It replaces brittle bash scripts with structured, queryable, crash-safe task automation.

## Core Concepts

### Tasks

Atomic units of work with rich metadata:

- **ID**: Hierarchical (e.g., "1.1", "2.3")
- **Status**: pending → in_progress → completed/failed/archived
- **Complexity**: simple | moderate | complex (maps to AI model)
- **Dependencies**: blockedBy relationships
- **Execution History**: Multiple attempts with logs
- **Results**: Verification output, commit hash

### Waves

Logical groupings of tasks (corresponds to implementation phases):

- Tasks within a wave can execute sequentially or in parallel (if not blocked)
- Wave completion triggers automatic archival (see [archive workflow](diagrams/archive-workflow.md))
- Provides checkpoint granularity for recovery

### Storage

JSONL (JSON Lines) format for tasks:

- One task per line
- Append-only for new tasks
- Rewrite for updates (acceptable for <1000 tasks)
- Git-friendly, debuggable with standard tools (cat, grep)

### Configuration

JSON-based project-specific settings:

- Paths to project artifacts (PRD, TODO, etc.)
- Verification command with timeout
- Multi-agent support with AI model mappings
- Execution policies (retries, halt-on-failure, auto-commit)
- Task ID format (hierarchical like "1.2" or JIRA-style like "DEV-1")

## Architecture Layers

See [architecture diagram](diagrams/architecture.md) for a visual representation of the component structure.

The system is organized into three main layers:

1. **CLI Layer**: Cobra-based command-line interface with subcommands
2. **Commands Layer**: Orchestrates workflows and handles I/O
3. **Core Components**: Storage, Executor, and Archiver working with support libraries (Config, Prompts, UI)

## Data Flow

### TODO Processing

See [TODO processing diagram](diagrams/todo-processing.md) for detailed workflow.

1. Parse TODO.md → Extract categories, priorities, items
2. Build context → Project info, tech stack, existing tasks
3. Execute AI agent (complex model) → Generate JSON task array
4. Validate and assign metadata → IDs, models, timestamps
5. Save to tasks.jsonl

### Task Execution

See [task execution diagram](diagrams/task-execution.md) for state transitions.

1. Query pending, unblocked tasks from tasks.jsonl
2. For each task:
   - Generate prompt with context and previous errors (if retry)
   - Run AI agent (model selected based on complexity)
   - Log output to `.devloop/logs/`
   - Run verification command
   - On success: auto-commit (if enabled), mark completed
   - On failure: record error, retry if attempts remain
   - Checkpoint session state
3. Auto-archive if wave complete (when enabled)

### Session Recovery

See [session recovery diagram](diagrams/session-recovery.md) for crash recovery flow.

1. Load `.devloop/state/session.json`
2. Find last checkpoint (task ID)
3. Query tasks after checkpoint
4. Resume execution from next task
5. Preserve attempt history and session context

## Storage Schema

### tasks.jsonl

```jsonl
{"id":"1.1","title":"...","status":"completed",...}
{"id":"1.2","title":"...","status":"in_progress",...}
```

### config.json

```json
{
  "version": "1.0",
  "project": {...},
  "verification": {...},
  "cli": {...},
  "execution": {...}
}
```

### session.json

```json
{
  "id": "uuid",
  "started_at": "2026-02-15T10:00:00Z",
  "last_checkpoint": "1.3",
  "tasks_completed": ["1.1", "1.2"],
  "tasks_failed": []
}
```

## AI Agent Integration

### Model Selection

- **Haiku**: Simple, single-file changes (< 50 lines)
- **Sonnet**: Moderate complexity, multi-file (50-200 lines)
- **Opus**: Complex algorithms, layout logic, state machines

### Prompt Structure

```text
Project Context: Name, tech stack, paths
Referenced Files: PRD, CLAUDE.md, etc.
Task: ID, title, description
Acceptance Criteria: Bulleted list
Verification: Command that will run
Previous Error: If retry attempt
Custom Instructions: From config
```

### Execution Modes

1. **Claude CLI**: `claude --model X --dangerously-skip-permissions -p "..."`
2. **Copilot CLI**: `copilot --model X --prompt "..."` (future)

## Verification Strategy

After each agent execution:

1. Run project-specific command (e.g., `npm run build && npm test`)
2. Set timeout to prevent hangs
3. Capture stdout/stderr
4. Log to `.devloop/logs/verify-TASKID-TIMESTAMP.log`
5. Success = exit code 0

## Archival System

When wave completes:

1. Query all completed tasks in wave
2. Export to `.devloop/archive/wave-N.jsonl`
3. Generate markdown summary `.devloop/archive/wave-N.md`
4. Update task statuses to "archived"
5. Update archive index
6. Remove from active context

## Error Handling

### Agent Failures

- Capture error output
- Include in next attempt prompt
- Limit retries (configurable)
- Mark failed after max attempts
- Optionally halt execution

### Verification Failures

- Treat as agent failure
- Provide output to next attempt
- Differentiate from agent runtime errors

### Crashes

- Session state checkpointed after each task
- Recovery: load session, resume from checkpoint
- Preserve attempt history

## Extension Points

### Adding AI Providers

Implement `AgentRunner` interface:

```go
type AgentRunner interface {
    Run(model, prompt, logPath string) (*AgentResult, error)
}
```

### Custom Verification

Configure any shell command:

```json
{
  "verification": {
    "command": "make test && make lint",
    "timeout_seconds": 600
  }
}
```

### Custom Prompts

Override templates in config:

```json
{
  "prompts": {
    "custom_instructions": "Always use tabs, not spaces..."
  }
}
```

## Performance Characteristics

- **Startup**: < 100ms (load config + JSONL)
- **Query**: < 10ms for 1000 tasks (in-memory index)
- **Task Execution**: Dominated by AI agent runtime (30s-5min)
- **JSONL Write**: < 1ms per task
- **Archive**: < 1s for 100 tasks

## Security Considerations

- AI agents execute with full project permissions
- Verification commands run in project directory
- Auto-commit creates git history (reversible)
- No network calls except to AI APIs
- Logs may contain sensitive code (local only)

## Testing Strategy

### Unit Tests

- Config load/save/validate
- JSONL operations
- Task filtering/querying
- Prompt generation

### Integration Tests

- Full workflow (init → process → run → archive)
- Mock AI agents (don't call real APIs)
- Crash recovery
- Error scenarios

### Manual Testing

- Use devloop to build itself
- Test on sample projects (various tech stacks)
- Verify cross-platform (Linux, macOS, Windows)

## Future Enhancements

- **Parallel execution**: Run independent tasks concurrently
- **Dependency DAG**: Visualize task dependencies
- **Web UI**: Browser-based status dashboard
- **Metrics**: Track success rates, durations, token usage
- **Templates**: Project templates with predefined configs
- **Plugins**: Custom task processors, verifiers
