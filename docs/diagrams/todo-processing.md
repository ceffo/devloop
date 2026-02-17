# TODO Processing Workflow

This diagram shows how TODO items are converted into structured tasks using AI agents.

```mermaid
flowchart TD
    START([User runs 'devloop todo process'])
    LOAD[Load TODO.md file]
    PARSE[Parse markdown<br/>Extract categories, priorities, items]
    
    CONTEXT[Build processing context<br/>Project info, tech stack, existing tasks]
    PROMPT[Generate AI prompt<br/>Include TODO items + context]
    
    AGENT[Execute AI agent<br/>Complex model - Opus/GPT-5]
    JSON[Parse JSON response<br/>Array of task objects]
    
    VALIDATE{Validate tasks}
    ASSIGN[Assign metadata<br/>IDs, models, timestamps]
    
    REVIEW{Review mode?}
    DISPLAY[Display task summary table]
    CONFIRM{User confirms?}
    
    SAVE[Save tasks to tasks.jsonl]
    SUCCESS([Tasks created successfully])
    CANCEL([Processing cancelled])
    
    START --> LOAD
    LOAD --> PARSE
    PARSE --> CONTEXT
    CONTEXT --> PROMPT
    PROMPT --> AGENT
    AGENT --> JSON
    JSON --> VALIDATE
    
    VALIDATE -->|Valid| ASSIGN
    VALIDATE -->|Invalid| AGENT
    
    ASSIGN --> REVIEW
    REVIEW -->|Yes| DISPLAY
    REVIEW -->|No| SAVE
    
    DISPLAY --> CONFIRM
    CONFIRM -->|Yes| SAVE
    CONFIRM -->|No| CANCEL
    
    SAVE --> SUCCESS
    
    style START fill:#e1f5ff
    style SUCCESS fill:#c8e6c9
    style CANCEL fill:#ffcdd2
    style AGENT fill:#fff9c4
    style VALIDATE fill:#f0f0f0
    style CONFIRM fill:#f0f0f0
```

## Processing Details

### Input Format

TODO items in markdown format:

```markdown
## Wave 1: Core Infrastructure

### High Priority
- [ ] Task description here
- [ ] Another task

### Medium Priority
- [ ] Task description
```

### AI Agent Processing

The AI agent (using a complex model like Opus or GPT-5) analyzes TODO items and generates structured tasks with:

- **Hierarchical IDs**: Assigns IDs like "1.1", "1.2", "2.1" based on wave and sequence
- **Complexity Assessment**: Determines if task is simple/moderate/complex
- **Model Selection**: Maps complexity to appropriate AI model
- **Dependencies**: Identifies blockedBy relationships between tasks
- **Acceptance Criteria**: Generates specific, testable success criteria

### Task Metadata

Each generated task includes:

- `id`: Hierarchical or JIRA-style identifier
- `title`: Brief task description
- `description`: Detailed requirements
- `complexity`: simple | moderate | complex
- `model`: AI model to use for execution
- `status`: Initially set to "pending"
- `acceptance_criteria`: List of success requirements
- `dependencies`: Tasks that must complete first
- `created_at`, `updated_at`: Timestamps

### Review Mode

When `--review` flag is used, the system displays all generated tasks in a table and prompts for confirmation before saving to storage. This allows users to verify the AI's task breakdown before committing.
