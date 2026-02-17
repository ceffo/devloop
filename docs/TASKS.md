# devloop Implementation Tasks

This file tracks the implementation of the devloop system. Tasks are organized into waves corresponding to the implementation phases.

## Wave 1: Core Infrastructure

### Task 1.1: Project setup and dependencies

**Model:** `claude-haiku-4-5-20251001` | **Complexity:** `simple`

Set up the Go project with required dependencies.

**Requirements:**

- Add cobra for CLI framework: `go get github.com/spf13/cobra@latest`
- Add color for terminal output: `go get github.com/fatih/color@latest`
- Add tablewriter for formatting: `go get github.com/olekukonko/tablewriter@latest`
- Create `.gitignore` with standard Go patterns
- Verify `go mod tidy` runs successfully

**Acceptance:**

- All dependencies in go.mod
- `go build ./cmd/devloop` succeeds
- .gitignore includes: `*.exe`, `*.test`, `.devloop/`, `devloop` binary

### Task 1.2: Configuration data structures

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Implement configuration schema and loading in `internal/config/`.

**Requirements:**

- Create `internal/config/config.go` with all structs from design doc:
  - Config, ProjectConfig, VerificationConfig, CLIConfig, ExecutionConfig
  - FilesConfig, ArchivalConfig, PromptsConfig
- Use proper struct tags: `json:"field_name"`
- Create `LoadConfig(path string) (*Config, error)` function
- Handle missing file with sensible defaults
- Create `SaveConfig(path string, cfg *Config) error` function

**Acceptance:**

- Compiles without errors
- Can load sample config.json
- Can save config to JSON
- Missing fields use defaults

### Task 1.3: Configuration validation

**Model:** `claude-haiku-4-5-20251001` | **Complexity:** `simple`

Add validation logic to ensure configuration is valid.

**Requirements:**

- Create `internal/config/validate.go`
- Implement `Validate() error` method on Config
- Check: project path exists
- Check: verification command is not empty
- Check: model names are valid (match known models)
- Check: timeout > 0
- Check: max_attempts > 0

**Acceptance:**

- Invalid config returns descriptive errors
- Valid config passes validation
- All edge cases covered (empty strings, negative numbers)

### Task 1.4: Task data structures

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Implement task storage structures in `internal/storage/`.

**Requirements:**

- Create `internal/storage/task.go` with:
  - Task struct with all fields from design doc
  - TaskMetadata, TaskExecution, Attempt, TaskResults structs
- Use `time.Time` for timestamps
- Use proper JSON tags
- Add helper methods:
  - `IsBlocked() bool` - check if task has unresolved blockedBy
  - `CanStart() bool` - check if task is pending and not blocked
  - `AddAttempt(attempt Attempt)` - append attempt to history

**Acceptance:**

- Compiles without errors
- Can marshal/unmarshal Task to/from JSON
- Helper methods work correctly

### Task 1.5: JSONL storage operations

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Implement JSONL read/write operations.

**Requirements:**

- Create `internal/storage/storage.go`
- Implement `Storage` struct with config reference
- Methods:
  - `NewStorage(cfg *Config) *Storage` - constructor
  - `LoadTasks() ([]*Task, error)` - read all tasks from JSONL
  - `SaveTask(task *Task) error` - append task to JSONL
  - `UpdateTask(task *Task) error` - rewrite file with updated task
  - `GetTask(id string) (*Task, error)` - find task by ID
- Use `encoding/json` for JSON operations
- Create parent directories if missing

**Acceptance:**

- Can write and read tasks to/from JSONL
- Update modifies existing task correctly
- Handles non-existent files gracefully
- File permissions are 0644

### Task 1.6: Task filtering and queries

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Add query capabilities for filtering tasks.

**Requirements:**

- Create `internal/storage/index.go`
- Implement `Filter` struct with optional fields:
  - Status string
  - Wave int
  - Complexity string
  - Tags []string
  - BlockedBy []string (empty means not blocked)
- Implement `QueryTasks(filter Filter) ([]*Task, error)` in storage.go
- Filter logic: AND across fields, OR within Tags
- Return tasks sorted by ID (ascending)

**Acceptance:**

- Can filter by status
- Can filter by wave
- Can filter by complexity
- Can filter by tags (any match)
- Can find unblocked tasks (empty BlockedBy in filter)
- Results sorted by ID

## Wave 2: CLI Commands Foundation

### Task 2.1: CLI framework setup

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Set up cobra CLI structure with root command and subcommands.

**Requirements:**

- Create `cmd/devloop/main.go` with root command
- Root command metadata: name, short/long descriptions
- Create placeholder subcommands (no implementation yet):
  - `init`, `config`, `todo`, `run`, `tasks`, `archive`, `session`
- Each subcommand has: Use, Short, Long, RunE
- Add `--config` global flag for config file path (default: `.devloop/config.json`)
- Add `--verbose` global flag for debug output

**Acceptance:**

- `go build ./cmd/devloop` succeeds
- `./devloop --help` shows all subcommands
- `./devloop <subcommand> --help` shows subcommand help
- Flags are recognized

### Task 2.2: Implement devloop init command

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Create the `devloop init` command to initialize a project.

**Requirements:**

- Create `internal/commands/init.go`
- Implement InitCmd cobra command
- Create `.devloop/` directory structure:
  - `config.json`, `tasks.jsonl`, `logs/`, `archive/`, `state/`
- Detect project settings:
  - Name (from current dir name)
  - Path (current dir absolute path)
  - Check for common files (package.json, go.mod, etc.) to guess tech stack
- Generate config.json with detected values
- Prompt before overwriting existing .devloop/

**Acceptance:**

- Creates `.devloop/` structure
- Generates valid config.json
- Detects project metadata
- Warns if .devloop/ exists
- `devloop init` in test project works

### Task 2.3: Implement devloop config commands

**Model:** `claude-haiku-4-5-20251001` | **Complexity:** `simple`

Add `devloop config show` and `devloop config validate` commands.

**Requirements:**

- Create `internal/commands/config.go`
- `config show`: Pretty-print config JSON with colors
- `config validate`: Load config and run Validate(), print results
- Use fatih/color for output formatting
- Show ✓/✗ symbols for validation results

**Acceptance:**

- `devloop config show` displays JSON
- `devloop config validate` catches invalid configs
- Output is readable and colored

### Task 2.4: Implement devloop tasks list command

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Create task listing with filtering options.

**Requirements:**

- Create `internal/commands/tasks.go`
- Implement `tasks list` subcommand
- Flags:
  - `--status STATUS` - filter by status
  - `--wave N` - filter by wave
  - `--complexity LEVEL` - filter by complexity
  - `--tags TAG1,TAG2` - filter by tags
- Use tablewriter for formatted output
- Columns: ID | Title | Status | Complexity | Attempts | Duration
- Color-code status: green (completed), yellow (in_progress), red (failed)

**Acceptance:**

- Lists all tasks when no filters
- Filters work correctly
- Table is formatted nicely
- Colors display correctly

### Task 2.5: Implement devloop tasks show command

**Model:** `claude-haiku-4-5-20251001` | **Complexity:** `simple`

Display detailed task information.

**Requirements:**

- Implement `tasks show TASK_ID` subcommand
- Display all task fields in readable format:
  - Metadata section
  - Description
  - Acceptance criteria (bulleted)
  - Execution history (table of attempts)
  - Results (if completed)
- Show full attempt details: model, duration, result, log path

**Acceptance:**

- Shows complete task details
- Formats nicely
- Handles missing task ID gracefully

### Task 2.6: Implement devloop tasks update command

**Model:** `claude-haiku-4-5-20251001` | **Complexity:** `simple`

Allow manual task status updates.

**Requirements:**

- Implement `tasks update TASK_ID --status STATUS` subcommand
- Validate status values (pending|in_progress|completed|failed|blocked|archived)
- Update task.UpdatedAt timestamp
- Save to JSONL

**Acceptance:**

- Updates task status
- Validates status values
- Rejects invalid task IDs
- Updates timestamp

## Wave 3: TODO Processing System

### Task 3.1: TODO file parser

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Parse TODO markdown files into structured data.

**Requirements:**

- Create `internal/processor/todo.go`
- Implement TodoItem struct: ID, Category, Content, Priority
- Implement `ParseTodoFile(path string) ([]TodoItem, error)`
- Parse markdown headings as categories
- Parse list items as todos
- Extract priority from markers (high/medium/low or !!/!/-)
- Handle nested lists
- Handle checkboxes `- [ ]` and `- [x]`

**Acceptance:**

- Parses sample TODO.md correctly
- Extracts categories and items
- Handles various markdown formats
- Skips completed items `- [x]`

### Task 3.2: Prompt template system

**Model:** `claude-haiku-4-5-20251001` | **Complexity:** `simple`

Create prompt templates for AI agents.

**Requirements:**

- Create `internal/prompts/templates.go`
- Define const for TodoProcessingPrompt (from design doc)
- Define const for TaskExecutionPrompt (from design doc)
- Create template helper functions:
  - `RenderTodoPrompt(project ProjectConfig, todos []TodoItem, nextID string) string`
  - `RenderTaskPrompt(cfg *Config, task *Task, attempt int, prevError string) string`
- Use `text/template` for variable substitution

**Acceptance:**

- Prompts render correctly with variables
- Handles empty/nil values gracefully
- Output matches design doc examples

### Task 3.3: AI agent runner abstraction

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Create abstraction for running AI CLI tools.

**Requirements:**

- Create `internal/executor/agent.go`
- Define AgentRunner interface: `Run(model, prompt, logPath string) (*AgentResult, error)`
- Implement ClaudeRunner struct
- Implement CopilotRunner struct (stub for now)
- AgentResult struct: Success bool, Output string, LogPath string, Error error
- Claude runner executes: `claude --model MODEL --dangerously-skip-permissions -p "PROMPT"`
- Redirect stdout/stderr to log file
- Return success based on exit code

**Acceptance:**

- Interface is well-defined
- ClaudeRunner can execute claude CLI
- Logs are written to file
- Captures errors correctly

### Task 3.4: TODO to tasks conversion

**Model:** `claude-opus-4-6` | **Complexity:** `complex`

Implement AI-driven TODO processing to generate tasks.

**Requirements:**

- Create `ProcessTodoItems(cfg *Config, todos []TodoItem, review bool) ([]*Task, error)` in processor/todo.go
- Generate next available task ID (query max existing ID + 1)
- Render TodoProcessingPrompt with context
- Execute Opus model (complex reasoning needed)
- Parse JSON output (array of task objects)
- Convert to Task structs with metadata:
  - Set SourceType = "todo"
  - Set SourceTodoItem = todo ID
  - Assign model based on complexity (from config.CLI.Models)
  - Set CreatedAt, UpdatedAt
  - Set Status = "pending"
  - Set MaxAttempts from config
- If review=true, display tasks and prompt for confirmation
- Return tasks for saving

**Acceptance:**

- Parses TODO items and calls agent
- Handles JSON parsing robustly
- Assigns models correctly based on complexity
- Review mode prompts user
- Returns valid Task structs

### Task 3.5: Implement devloop todo process command

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Create CLI command to process TODO files.

**Requirements:**

- Create `internal/commands/todo.go`
- Implement `todo process FILE` subcommand
- Flags:
  - `--review` - show tasks and confirm before saving
  - `--wave N` - assign to specific wave (default: auto-detect next wave)
- Load config
- Parse TODO file
- Call ProcessTodoItems
- Display summary table of generated tasks
- Save tasks to JSONL if approved

**Acceptance:**

- `devloop todo process .todo/TODO.md` works
- Review mode shows tasks and prompts
- Tasks are saved to JSONL
- Wave assignment works

## Wave 4: Execution Engine

### Task 4.1: Verification runner

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Implement verification command execution.

**Requirements:**

- Create `internal/executor/verify.go`
- Implement `RunVerification(cfg *Config, taskID string) (*VerifyResult, error)`
- VerifyResult struct: Success bool, Output string, LogPath string, Duration int
- Execute verification command from config in project directory
- Set timeout from config.Verify.TimeoutSeconds
- Capture stdout/stderr
- Write log to `.devloop/logs/verify-TASKID-TIMESTAMP.log`
- Return success based on exit code

**Acceptance:**

- Executes verification command
- Respects timeout
- Captures output
- Writes logs
- Returns correct success status

### Task 4.2: Auto-commit logic

**Model:** `claude-haiku-4-5-20251001` | **Complexity:** `simple`

Implement automatic git commits for completed tasks.

**Requirements:**

- Create `internal/executor/commit.go`
- Implement `AutoCommit(cfg *Config, task *Task) (string, error)`
- Generate commit message from template in config.Execution.CommitFormat
- Template variables: {task_id}, {title}, {description}, {complexity}
- Execute git commands:
  - `git add -A`
  - `git commit -m "MESSAGE"`
- Capture commit hash from output
- Return commit hash

**Acceptance:**

- Generates commit message correctly
- Executes git commands
- Returns commit hash
- Handles git errors gracefully

### Task 4.3: Session state management

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Implement session tracking for crash recovery.

**Requirements:**

- Create `internal/executor/session.go`
- Session struct: ID, StartedAt, LastCheckpoint, TasksCompleted, TasksFailed
- Implement:
  - `LoadSession(cfg *Config) *Session` - load or create new
  - `SaveSession(cfg *Config, session *Session) error` - write to .devloop/state/session.json
  - `CheckpointSession(session *Session, taskID string)` - update checkpoint
- Use UUID for session ID (add dependency: `go get github.com/google/uuid`)
- Create state directory if missing

**Acceptance:**

- Creates new session with UUID
- Saves session to JSON
- Loads existing session
- Checkpoints update LastCheckpoint

### Task 4.4: Main execution loop

**Model:** `claude-opus-4-6` | **Complexity:** `complex`

Implement the core dev loop execution engine.

**Requirements:**

- Create `internal/executor/executor.go`
- Implement `ExecuteDevLoop(cfg *Config, wave int, taskID string, continueSession bool) error`
- Workflow:
  1. Load or resume session
  2. Query tasks (pending, not blocked, matching wave/taskID if specified)
  3. For each task:
     - Mark in_progress
     - For each attempt (up to MaxAttempts):
       - Generate prompt (with previous error if retry)
       - Run agent (select model from task.Model)
       - Record attempt
       - If agent succeeds, run verification
       - If verification succeeds: commit (if enabled), mark completed, break
       - If verification fails: record error, retry if attempts remain
     - If all attempts fail: mark failed, halt if configured
     - Checkpoint after each task
  4. Return summary
- Handle interrupts gracefully (SIGINT/SIGTERM)
- Log progress with colors and progress indicators

**Acceptance:**

- Executes tasks in order
- Handles retries correctly
- Runs verification after each attempt
- Auto-commits on success
- Checkpoints state
- Respects halt_on_failure
- Handles interrupts

### Task 4.5: Implement devloop run command

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Create CLI command to execute dev loop.

**Requirements:**

- Create `internal/commands/run.go`
- Implement `run` command
- Flags:
  - `--wave N` - run only tasks in wave N
  - `--task ID` - run specific task
  - `--continue` - resume from last checkpoint
- Load config
- Call ExecuteDevLoop with parameters
- Display summary at end:
  - Tasks completed
  - Tasks failed
  - Total duration
  - Success rate

**Acceptance:**

- `devloop run` executes all pending tasks
- Filters work (--wave, --task)
- --continue resumes from checkpoint
- Summary is accurate

## Wave 5: Archival System

### Task 5.1: Archive data structures

**Model:** `claude-haiku-4-5-20251001` | **Complexity:** `simple`

Define archive metadata structures.

**Requirements:**

- Create `internal/archiver/archiver.go`
- Define Archive struct:
  - Wave int
  - ArchivedAt time.Time
  - TaskCount int
  - TaskIDs []string
  - OutputPath string
- Define ArchiveIndex map[int]Archive for `.devloop/archive/index.json`
- Implement:
  - `LoadArchiveIndex(cfg *Config) (map[int]Archive, error)`
  - `SaveArchiveIndex(cfg *Config, index map[int]Archive) error`

**Acceptance:**

- Structs defined correctly
- Can load/save archive index
- Handles missing index file

### Task 5.2: JSONL archive export

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Export completed tasks to archive JSONL.

**Requirements:**

- Implement `ArchiveWaveToJSONL(cfg *Config, storage *storage.Storage, wave int) (string, error)`
- Query all tasks with status=completed and wave=N
- Write to `.devloop/archive/wave-N.jsonl`
- One task per line (JSONL format)
- Return output file path

**Acceptance:**

- Exports completed tasks to JSONL
- File format is valid JSONL
- Handles empty wave gracefully
- Returns output path

### Task 5.3: Markdown archive summary

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Generate human-readable archive summaries.

**Requirements:**

- Implement `GenerateArchiveSummary(cfg *Config, tasks []*storage.Task, wave int) (string, error)`
- Create markdown document with:
  - Header: "Wave N - Completed Tasks"
  - Archived timestamp
  - For each task:
    - Heading: Task ID and title
    - Complexity badge
    - Description
    - Completed timestamp
    - Attempts count
    - Commit hash
    - Separator
- Write to `.devloop/archive/wave-N.md`
- Return file path

**Acceptance:**

- Generates readable markdown
- Includes all task details
- Formats nicely
- Returns file path

### Task 5.4: Archive workflow orchestration

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Orchestrate the full archival process.

**Requirements:**

- Implement `ArchiveWave(cfg *Config, storage *storage.Storage, wave int) error` in archiver.go
- Workflow:
  1. Validate wave has completed tasks
  2. Export to JSONL
  3. Generate markdown summary
  4. Update tasks status to "archived"
  5. Update archive index
  6. Save archive index
- Create archive directory if missing
- Return error if wave has no completed tasks

**Acceptance:**

- Full archival workflow works
- Creates both JSONL and markdown
- Updates task statuses
- Updates index
- Handles errors gracefully

### Task 5.5: Implement devloop archive command

**Model:** `claude-haiku-4-5-20251001` | **Complexity:** `simple`

Create CLI command for archiving waves.

**Requirements:**

- Create `internal/commands/archive.go`
- Implement `archive` command
- Flags:
  - `--wave N` - archive specific wave (required)
  - `--auto` - auto-detect completed waves and archive all
- Load config and storage
- Call ArchiveWave
- Display summary of archived tasks

**Acceptance:**

- `devloop archive --wave 1` works
- `devloop archive --auto` finds and archives completed waves
- Displays summary

### Task 5.6: Auto-archival integration

**Model:** `claude-haiku-4-5-20251001` | **Complexity:** `simple`

Add automatic archival to executor when wave completes.

**Requirements:**

- Modify `internal/executor/executor.go`
- After all tasks in wave complete:
  - Check if config.Archival.AutoArchive is true
  - If yes, call archiver.ArchiveWave
  - Log archival action
- Only archive if ALL tasks in wave are completed or failed

**Acceptance:**

- Wave auto-archives when complete
- Respects config setting
- Logs action
- Doesn't archive incomplete waves

## Wave 6: Session Management & Polish

### Task 6.1: Session status command

**Model:** `claude-haiku-4-5-20251001` | **Complexity:** `simple`

Display current session information.

**Requirements:**

- Create `internal/commands/session.go`
- Implement `session status` subcommand
- Display:
  - Session ID
  - Started at
  - Last checkpoint (task ID)
  - Tasks completed (count + list)
  - Tasks failed (count + list)
  - Current state (running/idle)
- Use colors for visual clarity

**Acceptance:**

- Shows session details
- Formats nicely
- Handles no session gracefully

### Task 6.2: Session recover command

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Implement crash recovery from checkpoint.

**Requirements:**

- Implement `session recover` subcommand
- Load session state
- Find last checkpoint task
- Query tasks after checkpoint (by ID comparison)
- Display recovery plan:
  - Last completed: Task X
  - Will resume from: Task Y
  - Remaining: N tasks
- Prompt for confirmation
- Call ExecuteDevLoop with continue=true

**Acceptance:**

- Recovers from checkpoint
- Shows recovery plan
- Resumes execution correctly
- Handles no session gracefully

### Task 6.3: Terminal UI improvements

**Model:** `claude-haiku-4-5-20251001` | **Complexity:** `simple`

Add consistent UI formatting across all commands.

**Requirements:**

- Create `internal/ui/ui.go`
- Implement helper functions:
  - `Success(format string, args ...interface{})` - green ✓
  - `Error(format string, args ...interface{})` - red ✗
  - `Warning(format string, args ...interface{})` - yellow ⚠
  - `Info(format string, args ...interface{})` - blue ℹ
  - `Section(title string)` - bold header with separator
  - `FormatDuration(seconds int) string` - human-readable duration
  - `FormatStatus(status string) string` - colored status
- Use fatih/color for colors
- Support NO_COLOR env var

**Acceptance:**

- UI helpers work across all commands
- Colors display correctly
- NO_COLOR disables colors
- Output is readable

### Task 6.4: Progress indicators

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Add progress bars/spinners for long operations.

**Requirements:**

- Add dependency: `go get github.com/schollz/progressbar/v3`
- Update executor.go to show progress:
  - Overall wave progress: "Task 3/10"
  - Attempt progress: "Attempt 1/2"
  - Verification running: spinner
- Update todo processor to show: "Processing TODO items..." with spinner
- Make progress optional with --quiet flag

**Acceptance:**

- Progress indicators display during execution
- Updates in real-time
- --quiet suppresses progress
- Doesn't interfere with output

### Task 6.5: Error handling improvements

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Improve error messages and handling throughout.

**Requirements:**

- Define custom error types in `internal/errors/errors.go`:
  - ErrConfigNotFound
  - ErrTaskNotFound
  - ErrInvalidConfig
  - ErrVerificationFailed
  - ErrAgentFailed
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Add error hints in commands (e.g., "Run 'devloop init' first")
- Log errors to file in `.devloop/logs/errors.log`
- Add --debug flag for verbose error output

**Acceptance:**

- Errors are informative
- Hints guide users
- Debug mode shows stack traces
- Error log captures all errors

### Task 6.6: Documentation generation

**Model:** `claude-haiku-4-5-20251001` | **Complexity:** `simple`

Generate documentation from code.

**Requirements:**

- Create `docs/ARCHITECTURE.md` documenting:
  - Component structure
  - Data flow
  - Storage format
  - Extension points
- Create `docs/CLI.md` documenting all commands (auto-generate from cobra)
- Create `docs/CONFIG.md` documenting all config options
- Add godoc comments to all exported functions
- Create `make docs` command to regenerate

**Acceptance:**

- Documentation is complete
- Godoc comments on all exports
- make docs regenerates docs
- Docs are readable

## Wave 7: Testing & Release

### Task 7.1: Unit tests for config

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Add comprehensive unit tests for config package.

**Requirements:**

- Create `internal/config/config_test.go`
- Test LoadConfig with valid/invalid JSON
- Test Validate with various invalid configs
- Test SaveConfig round-trip
- Test default value handling
- Achieve >80% coverage

**Acceptance:**

- All tests pass
- Coverage >80%
- Tests are readable
- `go test ./internal/config/...` succeeds

### Task 7.2: Unit tests for storage

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Add comprehensive unit tests for storage package.

**Requirements:**

- Create `internal/storage/storage_test.go`
- Create `internal/storage/task_test.go`
- Test JSONL read/write
- Test UpdateTask correctness
- Test QueryTasks filtering
- Test task helper methods (IsBlocked, CanStart)
- Use temp directories for test files
- Achieve >80% coverage

**Acceptance:**

- All tests pass
- Coverage >80%
- Tests clean up temp files
- `go test ./internal/storage/...` succeeds

### Task 7.3: Integration tests

**Model:** `claude-opus-4-6` | **Complexity:** `complex`

Create end-to-end integration tests.

**Requirements:**

- Create `tests/integration_test.go`
- Set up test project structure in temp directory
- Test full workflow:
  1. devloop init
  2. Create sample TODO file
  3. devloop todo process (mock agent with fake tasks)
  4. devloop tasks list
  5. devloop run (mock agent and verify)
  6. devloop archive
  7. Verify all files created correctly
- Mock agent execution (don't call real Claude API)
- Test crash recovery (interrupt mid-execution, resume)
- Test error scenarios (verification fails, agent fails)

**Acceptance:**

- Integration tests pass
- Full workflow tested
- Mocking works correctly
- `go test ./tests/...` succeeds

### Task 7.4: CLI smoke tests

**Model:** `claude-haiku-4-5-20251001` | **Complexity:** `simple`

Test all CLI commands execute without panics.

**Requirements:**

- Create `tests/cli_test.go`
- Test each command with --help flag
- Test each command with invalid arguments (should error, not panic)
- Test missing config scenarios
- Test all flags are recognized

**Acceptance:**

- All commands show help
- Invalid args produce errors, not panics
- Missing config handled gracefully
- `go test ./tests/...` succeeds

### Task 7.5: Build and release automation

**Model:** `claude-sonnet-4-5-20250929` | **Complexity:** `moderate`

Set up build automation and release process.

**Requirements:**

- Create `Makefile` with targets:
  - `make build` - build binary
  - `make test` - run all tests
  - `make install` - install to $GOPATH/bin
  - `make clean` - remove artifacts
  - `make lint` - run golangci-lint
- Create `.goreleaser.yml` for multi-platform builds
- Add dependency: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
- Create GitHub Actions workflow (`.github/workflows/test.yml`)
  - Run on push/PR
  - Run tests
  - Run lint
  - Build for linux/mac/windows

**Acceptance:**

- `make build` produces binary
- `make test` runs all tests
- `make lint` passes
- GitHub Actions workflow defined
- goreleaser config valid

### Task 7.6: Example project and tutorial

**Model:** `claude-haiku-4-5-20251001` | **Complexity:** `simple`

Create example project and getting started guide.

**Requirements:**

- Create `examples/sample-project/` with:
  - Simple Go/Node project
  - `.todo/TODO.md` with sample items
  - Sample tasks in expected format
- Create `docs/TUTORIAL.md` with step-by-step guide:
  - Installation
  - Initialize project
  - Process TODOs
  - Run dev loop
  - Check status
  - Archive
- Add screenshots/GIFs to docs
- Create `docs/FAQ.md` with common questions

**Acceptance:**

- Example project runs successfully
- Tutorial is complete and clear
- FAQ covers common issues
- Docs are polished

## Completion Criteria

All tasks completed when:

- ✓ All 7 waves complete
- ✓ Unit test coverage >80%
- ✓ Integration tests pass
- ✓ All CLI commands functional
- ✓ Documentation complete
- ✓ Example project works
- ✓ Build/release automation ready
- ✓ Successfully used in Modal project migration
