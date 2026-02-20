# devloop Architecture

This diagram shows the layered architecture of the devloop system.

```mermaid
graph TD
    CLI[CLI Layer - Cobra<br/>init, config, todo, run, tasks, archive, session]
    
    CMD[Commands Layer<br/>Orchestrates workflow, handles I/O]
    
    TASKSTORE["TaskStore Interface<br/>(Abstract)"]
    JSONL["JSONLStore<br/>(JSONL files)"]
    BEADS["BeadsStore<br/>(Beads DB)"]
    
    EXEC[Executor<br/>Agent + Verify + Coordinator]
    ARCHIVE[Archiver<br/>JSONL + Markdown]
    
    KNOWLEDGE["KnowledgeClient Interface<br/>(Abstract)"]
    MULCH["MulchClient<br/>(Mulch backend)"]
    NOOP["NoopClient<br/>(Disabled)"]
    
    CONFIG[Config]
    PROMPTS[Prompts]
    UI[UI Helpers]
    
    CLI --> CMD
    CMD --> TASKSTORE
    CMD --> EXEC
    CMD --> ARCHIVE
    
    TASKSTORE --> JSONL
    TASKSTORE --> BEADS
    
    EXEC --> TASKSTORE
    EXEC --> KNOWLEDGE
    EXEC --> CONFIG
    EXEC --> PROMPTS
    EXEC --> UI
    
    KNOWLEDGE --> MULCH
    KNOWLEDGE --> NOOP
    
    ARCHIVE --> TASKSTORE
    ARCHIVE --> CONFIG
    
    CMD --> UI
    
    style CLI fill:#e1f5ff
    style CMD fill:#fff4e1
    style TASKSTORE fill:#ffe0b2
    style JSONL fill:#f0f0f0
    style BEADS fill:#f0f0f0
    style EXEC fill:#f0f0f0
    style ARCHIVE fill:#f0f0f0
    style KNOWLEDGE fill:#ffe0b2
    style MULCH fill:#f0f0f0
    style NOOP fill:#f0f0f0
    style CONFIG fill:#e8f5e9
    style PROMPTS fill:#e8f5e9
    style UI fill:#e8f5e9
```

## Component Descriptions

### CLI Layer

Entry point using Cobra framework. Provides command-line interface with subcommands:

- **init**: Initialize devloop in a project
- **config**: View and validate configuration
- **todo**: Process TODO items into tasks
- **run**: Execute the development workflow
- **tasks**: Manage and view tasks
- **archive**: Archive completed tasks
- **session**: Manage execution sessions

### Commands Layer

Orchestrates workflows by coordinating between storage, executor, and other components. Handles user I/O and error reporting.

### TaskStore Interface

Defines the contract for task persistence and querying:

- **JSONLStore**: File-based implementation using JSONL format
  - Stores tasks in `.devloop/tasks.jsonl`
  - In-memory querying and filtering
  - Suitable for single-user development workflows
- **BeadsStore**: High-performance database implementation
  - Uses Beads distributed database
  - Supports concurrent access and synchronization
  - Optimized for large-scale task management
  - Includes Syncer and Compactor interfaces for maintenance

### KnowledgeClient Interface

Manages knowledge base interactions and context priming:

- **MulchClient**: Knowledge base backend
  - Records knowledge from completed tasks
  - Primes execution context with relevant information
  - Enables contextual AI agent execution
- **NoopClient**: No-op implementation
  - Used when knowledge management is disabled
  - Returns empty priming context

### Core Components

- **Executor**: Task execution engine with AI agent integration, verification, and optional coordinator
  - Injects knowledge context into task prompts
  - Runs coordinator agent as optional pre-step for complex tasks
  - Handles retries, verification, and auto-commits
  - Manages session checkpointing
- **Archiver**: Archives completed tasks to JSONL and markdown summaries

### Support Components

- **Config**: Configuration management and validation
- **Prompts**: AI prompt templates for task execution and TODO processing
- **UI**: Terminal formatting helpers (colors, tables, progress indicators)

