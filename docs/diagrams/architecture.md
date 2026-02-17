# devloop Architecture

This diagram shows the layered architecture of the devloop system.

```mermaid
graph TD
    CLI[CLI Layer - Cobra<br/>init, config, todo, run, tasks, archive, session]
    
    CMD[Commands Layer<br/>Orchestrates workflow, handles I/O]
    
    STORAGE[Storage<br/>JSONL + Query]
    EXEC[Executor<br/>Agent + Verify]
    ARCHIVE[Archiver<br/>JSONL + Markdown]
    
    CONFIG[Config]
    PROMPTS[Prompts]
    UI[UI Helpers]
    
    CLI --> CMD
    CMD --> STORAGE
    CMD --> EXEC
    CMD --> ARCHIVE
    
    STORAGE --> CONFIG
    EXEC --> CONFIG
    EXEC --> PROMPTS
    ARCHIVE --> CONFIG
    
    EXEC --> UI
    CMD --> UI
    
    style CLI fill:#e1f5ff
    style CMD fill:#fff4e1
    style STORAGE fill:#f0f0f0
    style EXEC fill:#f0f0f0
    style ARCHIVE fill:#f0f0f0
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
- **archive**: Archive completed waves
- **session**: Manage execution sessions

### Commands Layer

Orchestrates workflows by coordinating between storage, executor, and other components. Handles user I/O and error reporting.

### Core Components

- **Storage**: JSONL-based task storage with in-memory querying and filtering
- **Executor**: Task execution engine with AI agent integration and verification
- **Archiver**: Archives completed waves to JSONL and markdown summaries

### Support Components

- **Config**: Configuration management and validation
- **Prompts**: AI prompt templates for task execution and TODO processing
- **UI**: Terminal formatting helpers (colors, tables, progress indicators)
