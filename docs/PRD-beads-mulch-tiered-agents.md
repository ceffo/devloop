# PRD: Memory Model Evolution — Beads, Mulch, and Tiered Agents

## Overview

devloop is an agentic development workflow tool. Agents execute tasks from a JSONL task store,
verify their work, and commit changes. This PRD defines three major improvements to devloop's
memory model that make it better suited for long-horizon agentic programming:

1. **Storage abstraction** — decouple task store logic so multiple backends are supported
2. **Beads integration** — replace the monolithic `tasks.jsonl` with the `bd` graph issue tracker
3. **Mulch integration** — add a persistent agent knowledge layer via the `mulch` CLI
4. **Tiered agent architecture** — introduce an optional coordinator agent that decomposes complex tasks

These improvements are **additive and backward-compatible**: existing JSONL projects continue to
work unchanged. New capabilities are opt-in via config.

## Context and Motivation

### Current Limitations

**JSONL task store** (`internal/storage/storage.go`):
- Updates rewrite the entire file (acceptable but fragile at scale)
- No atomic "claim" operation for safe concurrent access
- No graph model — dependency tracking is a simple list of IDs with no structural queries
- Archiving requires manual intervention; there is no memory decay
- The `storage.Storage` concrete type is referenced throughout the codebase, making alternative backends impossible

**Stateless agents**:
- Each task execution starts cold — no memory of what succeeded or failed in previous sessions
- Repeated mistakes happen: agents re-discover known pitfalls, re-implement already-learned patterns
- Project-specific conventions (coding style, testing patterns, architectural decisions) must be re-injected via the prompt template

**Monolithic executor**:
- A single agent both plans and implements each task
- Complex tasks cannot be autonomously decomposed into parallel subtasks
- No separation between the concern of "what to build" and "how to build it"

### Solution Summary

| Problem | Solution | Tool |
|---|---|---|
| Brittle JSONL, no graph | Interface + Beads backend | `bd` CLI |
| Cold-start agents | Persistent expertise layer | `mulch` CLI |
| No task decomposition | Coordinator agent pattern | Prompt + `bd` CLI |

## Goals

1. Define a `TaskStore` interface so storage backends are interchangeable
2. Implement a `BeadsStore` backend that delegates task lifecycle to `bd`
3. Implement a `MulchClient` knowledge layer that injects expertise into prompts and records learnings after task completion
4. Add a `devloop knowledge` CLI subcommand for managing project expertise
5. Add a coordinator agent execution mode for autonomous task decomposition
6. Maintain full backward compatibility — JSONL remains the default backend

## Non-Goals

- Replacing the agent runner (Claude/Copilot CLI invocation is unchanged)
- Building a web UI or dashboard
- Implementing multi-machine distributed execution
- Supporting more than two tiers of agent hierarchy (coordinator → dev is sufficient)

---

## Requirements

### 1. Storage Abstraction Layer

#### 1.1 TaskStore Interface

Define a `TaskStore` interface in `internal/storage/store.go` with the following methods:

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

`QueryReadyTasks` replaces the executor's current two-step pattern of `QueryTasks` + `filterReadyTasks`.
`JSONLStore` implements it with the existing `filterReadyTasks` logic; `BeadsStore` delegates to
`bd ready --json`.

> **Note:** `Storage.ConfigPath()` exists in the current codebase but is not included in `TaskStore`.
> Callers should derive this from `config.Config.Project.Path` instead. Verify no callers depend on
> it via the storage type before removing.

#### 1.2 JSONLStore Rename

Rename the existing `Storage` struct to `JSONLStore`. It must implement `TaskStore` with zero
behavioral changes. All existing tests must pass after the rename.

The `NewStorage(cfg)` constructor becomes `NewJSONLStore(cfg)` returning `*JSONLStore`.

#### 1.3 Factory Function

Add `NewTaskStore(cfg *config.Config) (TaskStore, error)` in `internal/storage/store.go`:
- Returns a `*JSONLStore` when `cfg.Storage.Backend == "jsonl"` or backend is unset
- Returns a `*BeadsStore` when `cfg.Storage.Backend == "beads"`
- Returns an error for unknown backends

#### 1.4 Caller Updates

Update all packages that currently reference `*storage.Storage` to use `storage.TaskStore`:
- `internal/executor/executor.go` — `ExecuteDevLoop` and `executeTask`
- `internal/executor/session.go` — session management
- `internal/archiver/` — archive operations
- `internal/commands/run.go`, `archive.go`, `tasks.go` — command handlers
- `internal/processor/todo.go`, `prd.go` — `generateNextTaskID` function

#### 1.5 Config Schema Extension

Add `StorageConfig` to `internal/config/config.go`:

```go
type StorageConfig struct {
    Backend string `json:"backend,omitempty"` // "jsonl" (default) | "beads"
}
```

Add `Storage StorageConfig` field to `Config`. Default value: `"jsonl"`. Existing configs
without this field continue to behave identically.

---

### 2. Beads Storage Backend

Beads (`bd`) is a distributed, git-backed graph issue tracker designed for AI agents. It provides
hash-based IDs, atomic claim operations, dependency graph queries, and memory decay (compaction).

Install: `go install github.com/steveyegge/beads/cmd/bd@latest`

#### 2.1 BeadsStore Implementation

Create `internal/storage/beads.go` with `BeadsStore` implementing `TaskStore`.

`BeadsStore` communicates with Beads exclusively via the `bd` CLI (same pattern as the `agent`
package's use of `claude` and `copilot` CLI tools). It must not import Beads as a Go library.

Constructor: `NewBeadsStore(cfg *config.Config) (*BeadsStore, error)`

The constructor **must fail-fast** if the `bd` binary is not on `$PATH`. Return a descriptive error
including all install options:
- `go install github.com/steveyegge/beads/cmd/bd@latest`
- `npm install -g @beads/bd`
- `brew install beads`

#### 2.2 Task ID Mapping

devloop uses hierarchical IDs ("1.1", "2.3") or prefix-style IDs ("DEV-1"). Beads uses hash-based
IDs (`bd-a1b2`, `bd-f14c`) to prevent merge collisions. These must be reconciled.

**Strategy:** Let Beads assign hash IDs and maintain a bidirectional mapping using Beads' built-in
KV store (`bd kv`), which syncs automatically via Dolt:

```
bd kv set "devloop:DEV-5" "bd-x7f3"    # devloop ID → Beads ID
bd kv set "beads:bd-x7f3" "DEV-5"      # reverse lookup
```

`BeadsStore.GetTask("DEV-5")` resolves the Beads ID via `bd kv get "devloop:DEV-5"`, then calls
`bd show <beads-id> --json`. If the input is already a Beads hash ID, it's used directly.

`BeadsStore.SaveTask(task)` creates the task via `bd create`, captures the returned hash ID, and
writes both KV entries. The sidecar filename uses the Beads hash ID.

> **Why not `--id`?** Beads supports `bd create --id <custom-id>`, but devloop IDs like "1.1"
> contain dots that conflict with Beads' hierarchical notation (`bd-parent.1`). Hash IDs are safer.

#### 2.3 Task Sidecar Files

Beads tracks task lifecycle (status, dependencies, metadata). devloop tracks execution details
(attempt history, log paths, durations, commit hashes) that are internal and not meaningful to
Beads. Store this as a sidecar JSON file per task:

Path: `.devloop/tasks/<beads-id>.json`

Sidecar structure stores only devloop-internal execution data:
- `devloop_id` (original devloop ID, e.g. "DEV-5")
- `complexity` (simple/moderate/complex — maps to AI model)
- `acceptance_criteria`
- `tags`
- `max_attempts`
- `execution` (attempts, total_duration)
- `results` (verification_output, commit_hash, completed_at)

**Dependencies (`blocked_by`) are NOT stored in the sidecar.** Beads is the single source of truth
for dependency tracking via `bd dep add/remove/tree`. `BeadsStore.GetTask()` populates
`task.BlockedBy` from `bd show <id> --json` output. This eliminates the dual-source-of-truth
problem that would arise from maintaining deps in both systems.

The `BeadsStore.GetTask(id)` method must merge the Beads task data with the sidecar to return
a complete `*storage.Task`.

#### 2.4 Status Mapping

devloop and Beads use different status vocabularies. `BeadsStore` must translate between them:

| devloop status | Beads status | Notes |
|---|---|---|
| `pending` | `open` | Default state after creation |
| `in_progress` | `in_progress` | Set via `bd update --claim` |
| `completed` | `closed` | Via `bd close --reason "Verification passed"` |
| `failed` | `closed` + label | Via `bd close --reason "..."` then `bd label add <id> failed` |
| `blocked` | `blocked` | Via `bd dep add` (status computed from open blockers) |
| `archived` | `closed` + compacted | Via `bd admin compact` |

> **Important:** The canonical way to close a Beads issue is `bd close <id> --reason "..."`,
> NOT `bd update <id> --status closed`. `bd close` supports multiple IDs and records an audit trail.

#### 2.5 Beads CLI Operations

Map `TaskStore` operations to `bd` CLI calls:

| TaskStore method | `bd` CLI command |
|---|---|
| `SaveTask` | `bd create "<title>" --body-file=desc.md --json` + KV mapping |
| `GetTask` | `bd kv get "devloop:<id>"` → `bd show <beads-id> --json` |
| `LoadTasks` | `bd list --json --status open` |
| `QueryTasks` (by status) | `bd list --json --status <mapped-status>` |
| `QueryReadyTasks` | `bd ready --json` |
| `UpdateTask` (in_progress) | `bd update <id> --claim --json` |
| `UpdateTask` (completed) | `bd close <id> --reason "Verification passed"` |
| `UpdateTask` (failed) | `bd close <id> --reason "<error>"` + `bd label add <id> failed` |

Use `--body-file=<path>` for task descriptions instead of inline `--description` to avoid shell
escaping issues with multi-line content. Write a temp file, pass it, then clean up.

The `bd` binary must be located via `exec.LookPath`. If `bd` is not found and the config
requests the Beads backend, `NewBeadsStore` returns a descriptive error (see §2.1).

#### 2.6 Task Creation Flow

When creating a Beads task via `BeadsStore.SaveTask(task)`:

1. Write the human-readable description to a temp file
2. Call `bd create "<title>" --body-file=<tmpfile> --json` — capture the returned Beads hash ID
3. Write KV mapping: `bd kv set "devloop:<devloop-id>" "<beads-id>"` and reverse
4. Write sidecar file at `.devloop/tasks/<beads-id>.json` with complexity, acceptance criteria, tags, etc.
5. If `task.BlockedBy` is non-empty, call `bd dep add <beads-id> <blocker-beads-id>` for each
6. Clean up temp file

All devloop-specific metadata (complexity, acceptance criteria, tags, max attempts) lives in the
sidecar file, NOT in the Beads description. The Beads description is the human-readable task
description only — no embedded YAML or structured data.

#### 2.7 Dependency Linking

Beads is the **single source of truth** for task dependencies. devloop does not maintain a parallel
`blocked_by` list.

When `SaveTask` is called with non-empty `BlockedBy`:
1. Create the task with `bd create` (step 2 of §2.6)
2. Resolve each blocker's devloop ID to its Beads ID via `bd kv get "devloop:<blocker-id>"`
3. For each resolved blocker, call `bd dep add <new-beads-id> <blocker-beads-id>`
4. If a `bd dep add` call fails, log a warning and continue (the task exists but the dependency
   link is missing — recoverable via `bd dep add` manually)

When `QueryReadyTasks` is called, use `bd ready --json` to get only tasks with no open blockers,
matching devloop's existing `filterReadyTasks` semantics.

#### 2.8 Beads Initialization

Update `devloop init` (in `internal/commands/init.go`) to detect `storage.backend = "beads"` and:
1. Check if `bd` is installed; print installation instructions if not:
   - `go install github.com/steveyegge/beads/cmd/bd@latest`
   - `npm install -g @beads/bd`
   - `brew install beads`
2. Run `bd init` in the project directory
3. Inform the user that stealth mode is available via `bd init --stealth` for personal use

#### 2.9 Archive with Beads Backend

When `devloop archive` is run against a Beads backend:
1. Generate the existing `.devloop/archive/archive-TIMESTAMP.jsonl` and `.md` files (unchanged behavior)
2. Run `bd admin compact --auto --all --tier 1` to apply Beads memory decay (summarize old closed tasks)
3. Run `bd sync` to persist changes to git (see §2.10)

> **Note:** The compact command is under `bd admin`, not top-level `bd compact`.

#### 2.10 Beads Sync Lifecycle

Beads uses a Dolt-powered database as source of truth, with JSONL exported for git portability.
**`bd sync` must be called to persist changes to git** — it exports to JSONL, commits to Dolt,
pulls remote changes, and pushes. Without `bd sync`, task changes won't survive a `git clone` or
sync with other agents.

`BeadsStore` must call `bd sync` at these points:
- After `ExecuteDevLoop` completes (before printing the summary)
- After `devloop archive` (after compaction)
- After `devloop migrate --to beads` completes

Consider adding a `Sync() error` method to `BeadsStore` (not on `TaskStore` — it's backend-specific).
The executor calls it via a type assertion when appropriate.

#### 2.11 Migration Command

Add `devloop migrate` subcommand with `--to beads` and `--to jsonl` flags:

**`devloop migrate --to beads`:**
1. Read all tasks from existing `tasks.jsonl`
2. For each task, call `BeadsStore.SaveTask(task)` to create it in Beads (with correct status mapping)
3. Re-create all `blocked_by` dependency links via `bd dep add`
4. Run `bd sync` to persist to git
5. Print a summary of migrated tasks
6. On success, rename `tasks.jsonl` to `tasks.jsonl.migrated` as a backup

**`devloop migrate --to jsonl`:**
1. Read all tasks from Beads via `BeadsStore.LoadTasks()`
2. Write each task to a new `tasks.jsonl` (with reverse status mapping)
3. Print a summary of migrated tasks

Bidirectional migration provides a safety net for users evaluating the Beads backend.

---

### 3. Mulch Knowledge Layer

Mulch is a structured expertise persistence layer. Agents call `mulch record` to write learnings
and `mulch query` / `mulch prime` to read them. Everything lives in `.mulch/` in the project.

Install: `npm install -g mulch-cli`

#### 3.1 Knowledge Package

Create `internal/knowledge/` package with:

**`internal/knowledge/client.go`** — interface:

```go
type Client interface {
    Prime(domains []string, budget int, context string) (string, error) // returns context string
    Record(domain, recordType, content string) error
    IsEnabled() bool
}

func NewClient(cfg *config.Config) Client
```

**`internal/knowledge/mulch.go`** — `MulchClient` implementation:
- `Prime(domains, budget, context)` — runs `mulch prime [domains...] --format markdown --budget <budget> --context "<context>"`, captures and returns stdout. The `budget` parameter limits output tokens to avoid consuming excessive prompt space. The `context` parameter (typically the task title) improves relevance.
- `Record(domain, recordType, content)` — runs `mulch record <domain> --type <type> "<content>"`
- If `mulch` binary is not found, `Prime` returns empty string (non-fatal); `Record` returns an error
- All CLI calls have a **15-second timeout** to prevent hangs

**`internal/knowledge/noop.go`** — `NoopClient`:
- `Prime` returns `""`, `Record` returns `nil`, `IsEnabled` returns `false`
- Used when `knowledge.backend == "none"` or knowledge config is absent

`NewClient` returns `NoopClient` if `cfg.Knowledge.Backend == ""` or `"none"`, otherwise `MulchClient`.

#### 3.2 Knowledge Config

Add `KnowledgeConfig` to `internal/config/config.go`:

```go
type KnowledgeConfig struct {
    Backend           string   `json:"backend,omitempty"`            // "none" (default) | "mulch"
    Domains           []string `json:"domains,omitempty"`            // e.g. ["build","testing","architecture"]
    PrimeBudget       int      `json:"prime_budget,omitempty"`       // max tokens for mulch prime output (default: 2000)
    InjectOnExecute   bool     `json:"inject_on_execute,omitempty"`  // default: true when backend != "none"
    RecordOnComplete  bool     `json:"record_on_complete,omitempty"` // default: true when backend != "none"
}
```

Add `Knowledge KnowledgeConfig` field to `Config`. Default: backend `"none"`, all features disabled.

#### 3.3 Prompt Injection

Add `MulchContext` and `KnowledgeDomains` fields to `prompts.TaskPromptData`:

```go
MulchContext      string   // injected from mulch prime; empty if knowledge disabled
KnowledgeDomains  []string // available domains for recording
```

Add a new section to `TaskExecutionPrompt` template (rendered only when `MulchContext` is non-empty):

```
{{if .MulchContext}}
## Project Expertise
The following is accumulated knowledge from previous sessions. Use it to avoid known pitfalls
and follow established patterns:

{{.MulchContext}}
{{end}}
```

Update `prompts.RenderTaskPrompt` to accept a `knowledge.Client` parameter. Call
`client.Prime(domains, cfg.Knowledge.PrimeBudget, task.Title)` before rendering if
`cfg.Knowledge.InjectOnExecute` is true. The task title is passed as context to improve relevance.

Update `executor.executeTask` to pass the knowledge client to `prompts.RenderTaskPrompt`.

#### 3.4 Knowledge Recording Instructions in Prompt

Add a second new section to `TaskExecutionPrompt` (rendered only when domains are configured):

```
{{if .KnowledgeDomains}}
## Knowledge Recording
After completing this task, record any significant learnings using the mulch CLI.
Only record genuinely reusable knowledge — skip if nothing notable.

Available domains: {{joinDomains .KnowledgeDomains}}

Examples:
  mulch record build --type convention "Always run go generate before go build"
  mulch record testing --type failure "Race condition in TestFoo: add WaitGroup before channel send"
  mulch record architecture --type decision "Use interface for storage: enables Beads/JSONL swap without caller changes"
{{end}}
```

> **Note:** `joinDomains` requires registering a custom `template.FuncMap` with `strings.Join`.
> Go's `text/template` has no built-in join function.

This is agent-driven recording. The agent decides what is worth recording. devloop does not
auto-extract or auto-record.

Available Mulch record types: `convention`, `pattern`, `failure`, `decision`, `reference`, `guide`.

#### 3.5 Mulch Initialization

Update `devloop init` to detect `knowledge.backend = "mulch"` and:
1. Check if `mulch` (npm package `mulch-cli`) is installed
2. Run `mulch init` in the project directory
3. For each configured domain in `knowledge.domains`: run `mulch add <domain>`
4. Run `mulch setup <provider>` where `<provider>` is mapped from devloop's agent tool config:

| devloop `agent.tool` | Mulch provider |
|---|---|
| `claude` | `claude` |
| `copilot` | `codex` |
| (other) | skip `mulch setup` |

> **Note:** Mulch supports: `claude`, `cursor`, `codex`, `gemini`, `windsurf`, `aider`.
> If devloop's agent tool doesn't map to a known Mulch provider, skip `mulch setup` and
> log a note that the user can run it manually.

#### 3.6 `devloop knowledge` Subcommand

Add a new top-level CLI command `devloop knowledge` with subcommands:

| Subcommand | Action | `mulch` equivalent |
|---|---|---|
| `devloop knowledge status` | Show expertise freshness and record counts | `mulch status` |
| `devloop knowledge query [domain]` | Query records for a domain (or all) | `mulch query [domain]` |
| `devloop knowledge search <query>` | Search across all domains | `mulch search <query>` |
| `devloop knowledge compact` | Compact stale records | `mulch compact --auto` |
| `devloop knowledge doctor` | Health check: validate domains, find issues | `mulch doctor` |
| `devloop knowledge diff [ref]` | Show expertise changes since a git ref | `mulch diff [ref]` |
| `devloop knowledge prime [domains...]` | Print the context that would be injected | `mulch prime [domains...]` |

These commands are thin wrappers that exec `mulch` directly. They exist so users don't need
to know mulch command syntax.

---

### 4. Tiered Agent Architecture (Coordinator Mode)

#### 4.1 Overview

When `devloop run --coordinate` is passed, `complex` tasks are first processed by a coordinator
agent that autonomously decomposes them into subtasks. The subtasks are created directly in the
task store (Beads or JSONL) and then executed by the standard dev agent.

Coordinator mode is only triggered for tasks with `complexity == "complex"`. Simple and moderate
tasks execute directly, unchanged.

#### 4.2 Coordinator Config

Add `CoordinatorConfig` to `ExecutionConfig`:

```go
type CoordinatorConfig struct {
    Enabled    bool   `json:"enabled,omitempty"`     // default: false
    Agent      string `json:"agent,omitempty"`        // agent name (defaults to default agent)
    Model      string `json:"model,omitempty"`        // model override (defaults to "complex" model)
    MaxSubtasks int   `json:"max_subtasks,omitempty"` // max subtasks per task (default: 5)
}
```

Add `Coordinator CoordinatorConfig` to `ExecutionConfig`.

#### 4.3 Coordinator Prompt Template

Add `CoordinatorPrompt` to `internal/prompts/templates.go`:

```
You are the devloop Coordinator Agent. Your role is to analyze a complex task and decide
whether to decompose it into smaller, focused subtasks.

## Project Context
Name: {{.ProjectName}}
Tech Stack: {{.TechStack}}
Path: {{.ProjectPath}}

## Task to Evaluate
ID: {{.TaskID}}
Title: {{.Title}}
Description: {{.Description}}

Acceptance Criteria:
{{range .AcceptanceCriteria}}- {{.}}
{{end}}

## Task Store Commands
You have access to the following commands to create and link subtasks:
{{.TaskStoreInstructions}}

## Instructions
1. Analyze this task carefully.
2. If the task is focused enough to be implemented in one coding session (even if it's complex):
   Output exactly on its own line: ##DEVLOOP:PROCEED##
3. If the task genuinely needs to be split (multiple distinct components or concerns):
   - Create 2 to {{.MaxSubtasks}} focused subtasks using the task store commands
   - Each subtask must be independently implementable and verifiable
   - Use dependencies to order subtasks when necessary
   - After creating subtasks, output on its own line: ##DEVLOOP:DECOMPOSED##

Output ONLY the commands (one per line) followed by the sentinel.
Do not explain your reasoning. Do not include any other text.
```

#### 4.4 Task Store Instructions by Backend

The `{{.TaskStoreInstructions}}` field is populated based on the active backend:

**JSONL backend**: The coordinator writes a JSON file to `.devloop/coordinator-output.json`
that devloop reads and processes:
```
Write subtask definitions as a JSON array to the file .devloop/coordinator-output.json.
Each object must have: title, description, complexity, acceptance_criteria, blocked_by, tags.
```

**Beads backend**: The coordinator uses the `bd` CLI directly with hierarchical epics:
```
bd update {{.TaskID}} --claim                              : claim the parent task
bd create "<title>" --parent {{.TaskID}} --body-file=desc.md --json  : create subtask (auto-assigns {{.TaskID}}.1, .2, etc.)
```

Beads' `--parent` flag creates proper parent-child hierarchy (up to 3 levels). Subtask IDs are
auto-generated as `<parent>.1`, `<parent>.2`, etc. Dependencies between subtasks within the same
parent can be added via `bd dep add <subtask-id> <other-subtask-id>`.

This is cleaner than manual `bd dep add` for the parent-child relationship and produces a proper
tree visible via `bd dep tree`.

#### 4.5 Coordinator Execution Flow

In `internal/executor/executor.go`, add `runCoordinator` function:

```
func runCoordinator(ctx, cfg, store, task, agentRunner, model, notifier) (decomposed bool, err error)
```

1. Build coordinator prompt via `prompts.RenderCoordinatorPrompt`
2. Run agent (same `agentRunner` as dev tasks, coordinator model from config)
3. Parse output — use distinctive sentinels to avoid false matches in noisy agent output:
   - If last non-empty line is `##DEVLOOP:PROCEED##`: return `decomposed=false`
   - If JSONL backend and `##DEVLOOP:DECOMPOSED##`: read `.devloop/coordinator-output.json`, create subtasks, return `decomposed=true`
   - If Beads backend and `##DEVLOOP:DECOMPOSED##`: re-query `bd ready` — new subtasks are already in the store
4. If `decomposed=true`, add newly created subtasks to the execution queue

In `ExecuteDevLoop`, before executing a `complex` task, check `cfg.Execution.Coordinator.Enabled`:
- If true: call `runCoordinator` first
- If coordinator says `PROCEED`: execute normally
- If `DECOMPOSED`: skip the original task (it's now tracked by subtasks), continue to next

**Session Recovery:** If devloop crashes after decomposition but before all subtasks complete:
- The parent task is `in_progress` with subtasks created in the store
- On resume: if the parent task is `in_progress` and has child tasks (Beads: check via `bd show --json`;
  JSONL: check for `decomposed_into` field in sidecar), skip the coordinator and execute remaining subtasks
- With Beads backend: `bd ready --json` naturally returns only the unfinished subtasks

#### 4.6 `--coordinate` Flag

Add `--coordinate` flag to `devloop run` command. When set, it overrides
`cfg.Execution.Coordinator.Enabled = true` for this run only, without modifying the config file.

---

### 5. Updated `devloop init` Wizard

The existing `devloop init` interactive wizard must be extended to configure the new features.
New prompts (shown only if user selects "extended setup"):

1. "Task store backend? [jsonl (default) / beads]"
   - If `beads`: verify `bd` is installed, print install instructions if not
   - If `beads`: run `bd init`

2. "Enable agent knowledge layer? [no (default) / mulch]"
   - If `mulch`: verify `mulch` (npm) is installed
   - If `mulch`: run `mulch init`
   - If `mulch`: "Knowledge domains (comma-separated, e.g. build,testing,architecture):"
   - If `mulch`: run `mulch add <domain>` for each

3. "Enable coordinator agent for complex tasks? [yes / no (default)]"
   - If `yes`: set `execution.coordinator.enabled = true`

---

### 6. Documentation Updates

#### 6.1 DESIGN.md

Update `docs/DESIGN.md` to:
- Add a "Storage Backends" section describing JSONL vs Beads
- Add a "Knowledge Layer" section describing Mulch integration
- Add a "Tiered Agents" section describing the coordinator pattern
- Update the "Storage Schema" section to include sidecar files and `.mulch/` layout
- Update the "Extension Points" section: `TaskStore` interface replaces `AgentRunner` as primary extension point example

#### 6.2 Architecture Diagram

Update `docs/diagrams/architecture.md` to show:
- `TaskStore` interface with `JSONLStore` and `BeadsStore` as implementations
- `KnowledgeClient` interface with `MulchClient` as implementation
- Knowledge injection in the executor flow
- Coordinator agent as optional pre-step before dev agent

#### 6.3 AGENT_INSTRUCTIONS.md

Add a section "Knowledge Layer" explaining:
- How to query accumulated expertise: `devloop knowledge query`
- How the knowledge is injected into prompts automatically
- The expectation that agents record learnings at task completion

---

## Technical Constraints

- **Go version**: maintain compatibility with the current `go.mod` Go version
- **No new library dependencies** for Beads or Mulch — both are CLI tools, not Go libraries
- **Backward compatibility**: `storage.backend = "jsonl"` must remain the default; no config file migration required for existing projects
- **Config schema version**: Adding `StorageConfig` and `KnowledgeConfig` is additive — omitted fields default safely. Document this in the config section; a version bump to "1.1" is optional but recommended for clarity
- **Test coverage**: all new packages (`internal/storage/beads.go`, `internal/knowledge/`) must have unit tests with mocked CLI execution (similar to the existing `agent` package test patterns)
- **Error messages**: when `bd` or `mulch` binary is not found, error messages must include the exact install commands (Beads: `go install`, `npm install -g @beads/bd`, or `brew install beads`; Mulch: `npm install -g mulch-cli`)
- **CLI timeouts**: all `bd` CLI calls must have a 10-second timeout; `mulch prime` must have a 15-second timeout. Use `context.WithTimeout` on the `exec.CommandContext`
- **Atomic operations**: `BeadsStore.UpdateTask` must use `bd update <id> --claim` for `in_progress` transitions to prevent race conditions in future multi-agent scenarios
- **Template functions**: Register a custom `template.FuncMap` for `joinDomains` (wrapping `strings.Join`) before parsing any templates that use it

## Success Criteria

1. `go test ./...` passes with no regressions after the storage interface refactor
2. A project configured with `storage.backend = "beads"` can complete a full `devloop run` workflow (init → process → run → archive)
3. A project configured with `knowledge.backend = "mulch"` injects expertise context into task prompts and records learnings after completion
4. `devloop run --coordinate` successfully decomposes a complex task into subtasks on a Beads-backed project
5. A project using only `tasks.jsonl` (default config) is unaffected by all changes
6. `devloop migrate --to beads` successfully migrates all tasks from a JSONL file to Beads with correct status and dependency links
