# Session Recovery Flow

This diagram shows how devloop handles crashes and resumes execution from checkpoints.

```mermaid
flowchart TD
    START([User runs 'devloop run'])
    LOAD_SESSION{Session file exists?}
    
    CREATE_SESSION[Create new session<br/>UUID, timestamp]
    LOAD_EXISTING[Load existing session<br/>Read checkpoint data]
    
    QUERY_ALL[Query all pending tasks]
    QUERY_RESUME[Query tasks after<br/>last checkpoint]
    
    EXEC_LOOP[Execute tasks sequentially]
    CHECKPOINT[Checkpoint after each task<br/>Save task ID, status, timestamp]
    
    CRASH{Crash or<br/>interrupt?}
    COMPLETE{All tasks<br/>done?}
    
    FINISH([Session complete])
    INTERRUPTED([Execution interrupted])
    
    RECOVER([User runs recovery])
    SHOW_PLAN[Show recovery plan<br/>Last completed, next task, remaining]
    CONFIRM{User confirms?}
    RESUME[Resume from checkpoint]
    CANCEL([Recovery cancelled])
    
    START --> LOAD_SESSION
    LOAD_SESSION -->|No| CREATE_SESSION
    LOAD_SESSION -->|Yes| LOAD_EXISTING
    
    CREATE_SESSION --> QUERY_ALL
    LOAD_EXISTING --> QUERY_RESUME
    
    QUERY_ALL --> EXEC_LOOP
    QUERY_RESUME --> EXEC_LOOP
    
    EXEC_LOOP --> CHECKPOINT
    CHECKPOINT --> CRASH
    
    CRASH -->|Yes| INTERRUPTED
    CRASH -->|No| COMPLETE
    
    COMPLETE -->|Yes| FINISH
    COMPLETE -->|No| EXEC_LOOP
    
    INTERRUPTED -.->|Later| RECOVER
    RECOVER --> SHOW_PLAN
    SHOW_PLAN --> CONFIRM
    CONFIRM -->|Yes| RESUME
    CONFIRM -->|No| CANCEL
    RESUME --> QUERY_RESUME
    
    style START fill:#e1f5ff
    style FINISH fill:#c8e6c9
    style INTERRUPTED fill:#fff9c4
    style CRASH fill:#ffcdd2
    style CHECKPOINT fill:#e8f5e9
    style RECOVER fill:#e1f5ff
```

## Session Management

### Session File Structure

Located at `.devloop/state/session.json`:

```json
{
  "id": "uuid-here",
  "started_at": "2026-02-17T03:00:00Z",
  "last_checkpoint": "DEV-3",
  "tasks_completed": ["DEV-1", "DEV-2", "DEV-3"],
  "tasks_failed": []
}
```

### Checkpoint Strategy

After each task execution (success or failure):

1. Update session with task ID and outcome
2. Save session file to disk
3. Continue to next task

This ensures that if devloop crashes or is interrupted, it can resume from the last successfully checkpointed task.

### Recovery Process

When running `devloop session recover`:

1. **Load Session**: Read the last session file
2. **Identify Checkpoint**: Find the last completed task ID
3. **Query Remaining**: Get all tasks after the checkpoint
4. **Display Plan**: Show user what will be resumed
5. **Confirm**: Wait for user approval
6. **Resume**: Continue execution from checkpoint

### Manual Resume

Users can also manually resume with:

```bash
devloop run --continue
```

This automatically loads the last session and continues from the checkpoint without additional prompts.

### Edge Cases

- **No Session**: If no session exists, creates a new one and starts from the beginning
- **All Tasks Complete**: Recovery exits gracefully if checkpoint shows all tasks are done
- **Corrupted Session**: If session file is invalid, prompts user to start fresh or recover manually
