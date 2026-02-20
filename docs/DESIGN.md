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

### Storage

Tasks are stored via a pluggable `TaskStore` interface, enabling multiple storage backends:

- **JSONL (default)**: JSON Lines format, git-friendly, debuggable with standard tools
- **Beads**: Distributed graph issue tracker with atomic operations and memory decay (optional)

Both backends maintain the same task semantics; the choice is transparent to the executor.

## Storage Backends

devloop supports multiple task storage backends via a pluggable `TaskStore` interface:

### JSONL Backend (Default)

- **Trade-offs**: Simple, git-friendly, human-readable; requires full rewrite on updates
- **Best for**: Small to medium projects (<1000 tasks), rapid prototyping, version control
- **Format**: One task per line in `tasks.jsonl`
- **Operations**: Append for new tasks, in-memory index, full-file rewrite for updates
- **Dependencies**: None (built-in)

### Beads Backend

- **Trade-offs**: Distributed graph model with atomic operations and dependency queries; requires `bd` CLI
- **Best for**: Large projects, concurrent agent execution, complex task dependencies
- **Features**:
  - Hash-based task IDs preventing merge conflicts
  - Atomic claim operations for safe concurrent access
  - Graph-based dependency queries (`bd ready`, `bd dep tree`)
  - Memory decay via automatic compaction
  - Dolt-backed versioning with git integration
- **Storage**: Beads database in `.beads/`, task metadata in sidecar files (`.devloop/tasks/*.json`)
- **Dependencies**: `bd` CLI (install: `go install github.com/steveyegge/beads/cmd/bd@latest`)

The backend is selected in `config.json` via `storage.backend` (defaults to `"jsonl"`).

### Configuration

JSON-based project-specific settings:

- Paths to project artifacts (PRD, TODO, etc.)
- Verification command with timeout
- Multi-agent support with AI model mappings
- Execution policies (retries, halt-on-failure, auto-commit)
- Task ID format (hierarchical like "1.2" or JIRA-style like "DEV-1")

## Architecture Layers

See [architecture diagram](diagrams/architecture.md) for a visual representation of the component structure.

The system is organized into four main layers:

1. **CLI Layer**: Cobra-based command-line interface with subcommands
2. **Commands Layer**: Orchestrates workflows and handles I/O
3. **Knowledge Layer**: Optional persistent expertise via Mulch integration (see [Knowledge Layer](#knowledge-layer))
4. **Core Components**: Storage (TaskStore interface with JSONL/Beads backends), Executor, and Archiver working with support libraries (Config, Prompts, UI)

## Knowledge Layer

devloop integrates Mulch, a persistent expertise persistence layer, to eliminate cold-start agent behavior:

### Problem

Each task execution starts with fresh context. Agents cannot learn from previous sessions, causing:
- Repeated mistakes and re-discovery of known pitfalls
- Loss of project-specific conventions (coding style, testing patterns, architectural decisions)
- Inability to inject accumulated expertise into prompts

### Solution

Mulch provides a structured knowledge store (`.mulch/` directory) that:
- Records learnings after task completion via `mulch record <domain> --type <type> "<content>"`
- Injects expertise into task prompts via `mulch prime <domains>`
- Organizes knowledge by domain (e.g., "build", "testing", "architecture")

### Integration

When `knowledge.backend = "mulch"` in config:

1. **Before task execution**: Call `mulch prime <domains>` to retrieve relevant expertise context
2. **Inject into prompt**: Expertise is inserted into the task prompt under "Project Expertise" section
3. **Agent records learnings**: After completion, the agent uses `mulch record` to document significant insights
4. **Query expertise**: Run `devloop knowledge query [domain]` to inspect accumulated expertise

Example expertise injection:
```
## Project Expertise
The following is accumulated knowledge from previous sessions. Use it to avoid known pitfalls
and follow established patterns:

- convention: Always run go generate before go build
- failure: Race condition in TestFoo: add WaitGroup before channel send
- decision: Use interface for storage: enables Beads/JSONL swap without caller changes
```

Dependencies: `mulch` CLI (install: `npm install -g mulch-cli`)

### Configuration

```json
{
  "knowledge": {
    "backend": "mulch",
    "domains": ["build", "testing", "architecture"],
    "prime_budget": 2000,
    "inject_on_execute": true,
    "record_on_complete": true
  }
}
```

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
3. Auto-archive completed tasks (when enabled)

## Tiered Agents

For complex tasks, devloop can use a coordinator agent to autonomously decompose tasks into subtasks:

### Problem

Complex tasks are often monolithic — a single agent both plans and implements the entire solution. This can lead to:
- Missing decomposition into independently implementable subtasks
- Inefficient execution (complete recompilation due to single-task focus)
- Inability to parallelize work

### Solution

The **coordinator pattern** uses an optional coordinator agent:

1. **Coordinator phase** (for `complexity == "complex"`): Analyzes the task and decides whether to decompose it
   - If focused enough: signal `##DEVLOOP:PROCEED##` to execute as-is
   - If decomposable: create 2-5 focused subtasks with `bd create --parent` (Beads) or JSON output (JSONL)
2. **Subtask execution**: Standard dev agent executes each subtask sequentially
3. **Parallel-ready**: Subtasks can be executed in parallel in future versions

### Coordinator Mode

Enabled via:
- Configuration: `execution.coordinator.enabled = true` in `config.json`
- CLI flag: `devloop run --coordinate` (overrides config for this run)

The coordinator uses the same agent runner as dev tasks but with a different prompt template optimized for task decomposition.

### Example Flow

```
Original task (complex): "Implement full REST API with authentication, database, and testing"

Coordinator output:
  1. Create subtask: "Design database schema and models"
  2. Create subtask: "Implement authentication handlers" (blocked by subtask 1)
  3. Create subtask: "Implement CRUD endpoints" (blocked by subtask 1)
  4. Create subtask: "Add comprehensive tests" (blocked by subtasks 1-3)

Dev agent execution:
  - Task 1: Create models → success
  - Task 2: Auth handlers → success
  - Task 3: CRUD endpoints → success
  - Task 4: Tests → success
```

### Session Recovery

If devloop crashes after decomposition:
- Parent task remains `in_progress`
- Subtasks already created in task store
- On resume: skip coordinator, execute remaining subtasks

## Session Recovery

See [session recovery diagram](diagrams/session-recovery.md) for crash recovery flow.

1. Load `.devloop/state/session.json`
2. Find last checkpoint (task ID)
3. Query tasks after checkpoint
4. Resume execution from next task
5. Preserve attempt history and session context

## Storage Schema

### tasks.jsonl (JSONL Backend)

```jsonl
{"id":"1.1","title":"...","status":"completed",...}
{"id":"1.2","title":"...","status":"in_progress",...}
```

### Beads Backend

Tasks stored in Beads database (`.beads/` directory) with sidecar metadata:

**Beads task** (via `bd show <id> --json`):
- Hash-based ID (e.g., `bd-a1b2`)
- Title, description, status (open, in_progress, closed, blocked)
- Dependencies graph
- Labels and metadata

**Sidecar file** (`.devloop/tasks/<beads-id>.json`):
```json
{
  "devloop_id": "DEV-5",
  "complexity": "moderate",
  "acceptance_criteria": ["..."],
  "tags": ["..."],
  "max_attempts": 3,
  "execution": {
    "attempts": [
      {
        "number": 1,
        "started_at": "2026-02-15T10:00:00Z",
        "duration_seconds": 120,
        "status": "failed",
        "error": "..."
      }
    ],
    "total_duration": 120
  },
  "results": {
    "verification_output": "...",
    "commit_hash": "abc123",
    "completed_at": "2026-02-15T10:02:00Z"
  }
}
```

Dependencies are stored in Beads, not the sidecar. Use `bd dep tree` to inspect.

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

### .mulch/ Directory (Knowledge Layer)

When knowledge integration is enabled (Mulch backend):

```
.mulch/
  ├── domains/
  │   ├── build.md
  │   ├── testing.md
  │   └── architecture.md
  ├── records/
  │   ├── 2026-02-15/
  │   │   ├── build-convention-123.md
  │   │   └── testing-failure-456.md
  │   └── 2026-02-16/
  │       └── architecture-decision-789.md
  ├── index.json
  └── config.json
```

Knowledge is organized by domain and indexed for fast retrieval via `mulch query` and `mulch prime`.

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

When archiving tasks:

1. Query completed tasks
2. Export to `.devloop/archive/archive-TIMESTAMP.jsonl`
3. Generate markdown summary `.devloop/archive/archive-TIMESTAMP.md`
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

### Storage Backend

Implement the `TaskStore` interface to add a new storage backend:

```go
type TaskStore interface {
    LoadTasks() ([]*Task, error)
    SaveTask(task *Task) error
    UpdateTask(task *Task) error
    GetTask(id string) (*Task, error)
    QueryTasks(filter Filter) ([]*Task, error)
    QueryReadyTasks() ([]*Task, error) // tasks with no open blockers
}
```

The built-in `JSONLStore` implements this interface. To add a new backend (e.g., SQL database):
1. Create a struct implementing `TaskStore`
2. Register in the factory function: `NewTaskStore(cfg *config.Config) (TaskStore, error)`
3. Update config schema to accept the new backend name

This approach has enabled support for both JSONL and Beads backends with identical semantics.

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
