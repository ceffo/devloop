# Task Execution Flow

This diagram shows the lifecycle of a task from pending to completion or failure.

```mermaid
stateDiagram-v2
    [*] --> pending: Task created
    
    pending --> in_progress: Execution starts
    
    in_progress --> running_agent: Generate prompt
    running_agent --> running_verification: Agent succeeds
    running_agent --> recording_error: Agent fails
    
    running_verification --> auto_commit: Verification passes
    running_verification --> recording_error: Verification fails
    
    auto_commit --> completed: Commit created
    
    recording_error --> retry_check: Record attempt
    retry_check --> in_progress: Attempts < max_attempts
    retry_check --> failed: Max attempts reached
    
    completed --> archived: Wave completion
    failed --> [*]
    archived --> [*]
    
    note right of running_agent
        Execute AI agent with
        task prompt and context
    end note
    
    note right of running_verification
        Run verification command
        (e.g., tests, build)
    end note
    
    note right of auto_commit
        Auto-commit if enabled
        in configuration
    end note
```

## Execution Details

### Task States

- **pending**: Task created but not yet started
- **in_progress**: Task execution has begun
- **completed**: Task finished successfully with verification passed
- **failed**: Task failed after all retry attempts exhausted
- **archived**: Completed task moved to archive when wave completes

### Execution Steps

1. **Generate Prompt**: Build AI agent prompt with task context, project info, and previous errors (if retry)
2. **Run Agent**: Execute AI CLI tool (Claude, Copilot, etc.) with selected model based on complexity
3. **Verify**: Run project-specific verification command (tests, build, lint, etc.)
4. **Auto-commit**: Create git commit if verification passes and auto-commit is enabled
5. **Retry Logic**: If agent or verification fails, retry up to `max_attempts` configured in settings

### Checkpointing

After each task completes or fails, the session state is checkpointed to enable crash recovery.
