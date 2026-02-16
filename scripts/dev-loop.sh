#!/usr/bin/env bash
set -euo pipefail

# ─────────────────────────────────────────────────────────────────────
# Universal Dev Loop — Config-Driven Edition
#
# Project-agnostic AI-driven development loop that reads configuration
# from .devloop/config.json. Works with any project that has this config.
#
# Features:
# - Reads verification command from config
# - Reads task file path from config
# - Reads context files for prompts from config
# - Supports multiple CLI tools (claude, copilot)
# - Auto-retry with error context
# - Auto-commit on success
# - Resume-safe via .dev-loop-progress
#
# Usage:
#   ./scripts/dev-loop.sh [wave]              # Run all tasks in wave N
#   ./scripts/dev-loop.sh [wave] [start] [end] # Run tasks X.Y to X.Z in wave
#   CLI_TOOL=copilot ./scripts/dev-loop.sh    # Use copilot instead of claude
# ─────────────────────────────────────────────────────────────────────

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CONFIG_FILE="${PROJECT_DIR}/.devloop/config.json"
LOG_DIR="${PROJECT_DIR}/.dev-loop-logs"

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

# ─── Config Loading ──────────────────────────────────────────────────

if [[ ! -f "$CONFIG_FILE" ]]; then
    echo -e "${RED}Error: Config file not found: ${CONFIG_FILE}${NC}"
    echo -e "${YELLOW}Run 'devloop init' or create .devloop/config.json${NC}"
    exit 1
fi

# Check if jq is available, if not try python
if command -v jq &> /dev/null; then
    JSON_TOOL="jq"
    get_config_value() {
        jq -r "$1" "$CONFIG_FILE" 2>/dev/null || echo ""
    }
else
    JSON_TOOL="python3"
    get_config_value() {
        python3 -c "import json, sys; data=json.load(open('$CONFIG_FILE')); print($1)" 2>/dev/null || echo ""
    }
fi

# Load configuration values
PROJECT_NAME=$(get_config_value '.project.name')
TASKS_FILE="$PROJECT_DIR/$(get_config_value '.files.tasks')"
VERIFY_COMMAND=$(get_config_value '.verification.command')
CLI_TOOL_CONFIG=$(get_config_value '.cli.tool')
MAX_RETRIES=$(get_config_value '.execution.max_attempts')
AUTO_COMMIT=$(get_config_value '.execution.auto_commit')
COMMIT_FORMAT=$(get_config_value '.execution.commit_format')

# CLI tool override via environment
CLI_TOOL="${CLI_TOOL:-${CLI_TOOL_CONFIG:-claude}}"

# Model mappings from config
MODEL_SIMPLE=$(get_config_value '.cli.models.simple')
MODEL_MODERATE=$(get_config_value '.cli.models.moderate')
MODEL_COMPLEX=$(get_config_value '.cli.models.complex')

# Context files for prompts
mapfile -t CONTEXT_FILES < <(get_config_value '.prompts.task_context_files[]' 2>/dev/null || echo "")
CUSTOM_INSTRUCTIONS=$(get_config_value '.prompts.custom_instructions')

# Validation
if [[ -z "$TASKS_FILE" || ! -f "$TASKS_FILE" ]]; then
    echo -e "${RED}Error: Tasks file not found: ${TASKS_FILE}${NC}"
    exit 1
fi

if [[ -z "$VERIFY_COMMAND" ]]; then
    echo -e "${RED}Error: verification.command not set in config${NC}"
    exit 1
fi

# ─── Model Selection ─────────────────────────────────────────────────

get_model_for_task() {
    local task_id="$1"

    # Find the complexity metadata for this task
    local complexity_line
    complexity_line=$(awk "/^### Task ${task_id}:/{flag=1; next} flag && /\*\*Complexity:\*\*/{print; exit}" "$TASKS_FILE")

    if [[ -z "$complexity_line" ]]; then
        echo "$MODEL_MODERATE"
        return
    fi

    # Extract complexity level
    local complexity
    complexity=$(echo "$complexity_line" | grep -oP '\`\K[^`]+(?=\`)' | head -1)

    case "$complexity" in
        simple)   echo "$MODEL_SIMPLE" ;;
        moderate) echo "$MODEL_MODERATE" ;;
        complex)  echo "$MODEL_COMPLEX" ;;
        *)        echo "$MODEL_MODERATE" ;;
    esac
}

# ─── CLI-specific Flags ──────────────────────────────────────────────

get_cli_flags() {
    local model="$1"
    case "$CLI_TOOL" in
        claude)
            echo "--model $model --dangerously-skip-permissions -p"
            ;;
        copilot)
            echo "--model $model --allow-all-tools --allow-all-paths --no-ask-user -p"
            ;;
        *)
            echo -e "${RED}Unknown CLI_TOOL: ${CLI_TOOL}. Use 'claude' or 'copilot'.${NC}" >&2
            exit 1
            ;;
    esac
}

# ─── Dynamic Task Parsing ────────────────────────────────────────────

parse_task_ids() {
    grep "^### Task [0-9]\+\.[0-9]\+:" "$TASKS_FILE" | \
    sed 's/^### Task \([0-9]\+\.[0-9]\+\):.*/\1/'
}

get_task_description() {
    local task_id="$1"
    grep "^### Task ${task_id}:" "$TASKS_FILE" | \
    sed "s/^### Task ${task_id}: //"
}

# ─── Filter tasks by wave and range ──────────────────────────────────

filter_tasks() {
    local wave="$1"
    local start_task="${2:-}"
    local end_task="${3:-}"

    local all_tasks
    mapfile -t all_tasks < <(parse_task_ids)

    local filtered=()
    for task_id in "${all_tasks[@]}"; do
        # Extract wave number (first part before dot)
        local task_wave="${task_id%%.*}"

        # Filter by wave if specified
        if [[ -n "$wave" && "$task_wave" != "$wave" ]]; then
            continue
        fi

        # Filter by range if specified
        if [[ -n "$start_task" ]]; then
            # Check if task_id >= start_task
            if [[ "$task_id" < "$start_task" ]]; then
                continue
            fi
        fi

        if [[ -n "$end_task" ]]; then
            # Check if task_id <= end_task
            if [[ "$task_id" > "$end_task" ]]; then
                continue
            fi
        fi

        filtered+=("$task_id")
    done

    printf '%s\n' "${filtered[@]}"
}

# ─── Prompt Generation ───────────────────────────────────────────────

generate_prompt() {
    local task_id="$1"
    local extra_context="${2:-}"

    # Build context file list
    local context_files_text=""
    if [[ ${#CONTEXT_FILES[@]} -gt 0 ]]; then
        context_files_text="Read these files first:\n"
        for file in "${CONTEXT_FILES[@]}"; do
            if [[ -n "$file" && -f "$PROJECT_DIR/$file" ]]; then
                context_files_text+="- ${file}\n"
            fi
        done
    fi

    cat <<PROMPT
You are working on the ${PROJECT_NAME} project.

${context_files_text}
Execute **Task ${task_id}** from ${TASKS_FILE#$PROJECT_DIR/}.

Follow the acceptance criteria exactly. When done:
1. Run \`${VERIFY_COMMAND}\` to verify.
2. Fix any issues until verification passes.
3. Do NOT commit — the dev-loop script handles commits.

${CUSTOM_INSTRUCTIONS}

Focus only on this task. Do not work on other tasks.
${extra_context}
PROMPT
}

generate_retry_prompt() {
    local task_id="$1"
    local error_output="$2"
    generate_prompt "$task_id" "

IMPORTANT: A previous attempt at this task failed verification. Here is the error output:

\`\`\`
${error_output}
\`\`\`

Fix these errors. The code from the previous attempt is already in place — do not start from scratch. Just fix what's broken.
"
}

# ─── Task Execution ──────────────────────────────────────────────────

run_task() {
    local task_id="$1"
    local prompt="$2"
    local model
    model=$(get_model_for_task "$task_id")
    local log_file="${LOG_DIR}/task-${task_id}.log"
    local cli_flags
    cli_flags=$(get_cli_flags "$model")

    echo -e "${BLUE}  CLI:   ${BOLD}${CLI_TOOL}${NC}"
    echo -e "${BLUE}  Model: ${BOLD}${model}${NC}"
    echo -e "${BLUE}  Launching ${CLI_TOOL} (non-interactive, all tools authorized)...${NC}"

    # shellcheck disable=SC2086
    $CLI_TOOL $cli_flags "$prompt" 2>&1 | tee "$log_file"

    return ${PIPESTATUS[0]}
}

run_verify() {
    local task_id="$1"
    local verify_log="${LOG_DIR}/verify-${task_id}.log"
    echo -e "${BLUE}  Running: ${VERIFY_COMMAND}${NC}"

    if eval "$VERIFY_COMMAND" > "$verify_log" 2>&1; then
        echo -e "${GREEN}  ✓ Verification passed${NC}"
        return 0
    else
        echo -e "${RED}  ✗ Verification failed${NC}"
        tail -80 "$verify_log"
        return 1
    fi
}

# ─── Main ────────────────────────────────────────────────────────────

cd "$PROJECT_DIR"
mkdir -p "$LOG_DIR"

# Parse arguments: wave [start_task] [end_task]
WAVE="${1:-}"
START_TASK="${2:-}"
END_TASK="${3:-}"

# Build task list
if [[ -n "$WAVE" ]]; then
    mapfile -t TASK_IDS < <(filter_tasks "$WAVE" "$START_TASK" "$END_TASK")
else
    mapfile -t TASK_IDS < <(parse_task_ids)
fi

if [[ ${#TASK_IDS[@]} -eq 0 ]]; then
    echo -e "${RED}No tasks found matching criteria.${NC}"
    exit 1
fi

# Get descriptions
declare -a TASK_DESCRIPTIONS=()
for task_id in "${TASK_IDS[@]}"; do
    TASK_DESCRIPTIONS+=("$(get_task_description "$task_id")")
done

echo -e "${BOLD}${BLUE}"
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║              Universal Dev Loop (Config-Driven)              ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo -e "${NC}"
echo -e "${BLUE}Project:  ${BOLD}${PROJECT_NAME}${NC}"
echo -e "${BLUE}CLI:      ${BOLD}${CLI_TOOL}${NC} ${YELLOW}(override with CLI_TOOL=copilot)${NC}"
echo -e "${BLUE}Verify:   ${VERIFY_COMMAND}${NC}"
echo -e "${BLUE}Wave:     ${WAVE:-all}${NC}"
echo ""
echo -e "${BLUE}Tasks to run: ${#TASK_IDS[@]}${NC}"
for i in "${!TASK_IDS[@]}"; do
    echo -e "  ${TASK_IDS[$i]}: ${TASK_DESCRIPTIONS[$i]}"
done
echo ""

# Resume from last completed task
START_INDEX=0
if [[ -f ".dev-loop-progress" ]]; then
    LAST_COMPLETED=$(cat .dev-loop-progress)
    echo -e "${YELLOW}Resuming after task ${LAST_COMPLETED}${NC}"
    for i in "${!TASK_IDS[@]}"; do
        if [[ "${TASK_IDS[$i]}" == "$LAST_COMPLETED" ]]; then
            START_INDEX=$((i + 1))
            break
        fi
    done
fi

FAILED=0
for i in "${!TASK_IDS[@]}"; do
    if [[ $i -lt $START_INDEX ]]; then
        continue
    fi

    TASK_ID="${TASK_IDS[$i]}"
    DESCRIPTION="${TASK_DESCRIPTIONS[$i]}"

    echo ""
    echo -e "${BOLD}${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}[$(date +%H:%M:%S)] Task ${TASK_ID}: ${DESCRIPTION}${NC}"
    echo -e "${BOLD}${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    # ── Run AI CLI with retries ──
    TASK_SUCCEEDED=false
    for attempt in $(seq 1 "$MAX_RETRIES"); do
        echo -e "${BLUE}  Attempt ${attempt}/${MAX_RETRIES}${NC}"

        if [[ $attempt -eq 1 ]]; then
            PROMPT=$(generate_prompt "$TASK_ID")
        else
            ERROR_OUTPUT=$(tail -80 "${LOG_DIR}/verify-${TASK_ID}.log" 2>/dev/null || echo "Unknown error")
            PROMPT=$(generate_retry_prompt "$TASK_ID" "$ERROR_OUTPUT")
        fi

        run_task "$TASK_ID" "$PROMPT" || true

        if run_verify "$TASK_ID"; then
            TASK_SUCCEEDED=true
            break
        fi

        if [[ $attempt -lt $MAX_RETRIES ]]; then
            echo -e "${YELLOW}  Retrying with error context...${NC}"
        fi
    done

    if [[ "$TASK_SUCCEEDED" == false ]]; then
        echo -e "${RED}  ✗ Task ${TASK_ID} failed after ${MAX_RETRIES} attempts.${NC}"
        echo -e "${RED}  Halting loop. Fix manually, then re-run to resume.${NC}"
        echo -e "${RED}  Logs: ${LOG_DIR}/task-${TASK_ID}.log${NC}"
        FAILED=1
        break
    fi

    # ── Auto-commit ──
    if [[ "$AUTO_COMMIT" == "true" ]]; then
        if git diff --quiet && git diff --cached --quiet; then
            echo -e "${YELLOW}  No changes to commit (task may have been a no-op)${NC}"
        else
            # Generate commit message from template
            COMMIT_MSG="${COMMIT_FORMAT}"
            COMMIT_MSG="${COMMIT_MSG//\{task_id\}/${TASK_ID}}"
            COMMIT_MSG="${COMMIT_MSG//\{title\}/${DESCRIPTION}}"

            git add -A
            git commit -m "$COMMIT_MSG"
            echo -e "${GREEN}  ✓ Committed: ${COMMIT_MSG}${NC}"
        fi
    fi

    # Save progress
    echo "$TASK_ID" > .dev-loop-progress

    echo -e "${GREEN}  ━━━ Task ${TASK_ID} complete [$(date +%H:%M:%S)] ━━━${NC}"
done

echo ""
if [[ $FAILED -eq 0 ]]; then
    echo -e "${BOLD}${GREEN}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                    All tasks complete!                       ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    rm -f .dev-loop-progress
else
    echo -e "${BOLD}${RED}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║              Loop halted — see logs for details             ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    exit 1
fi
