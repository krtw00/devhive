#!/bin/bash
#
# DevHive Send Task Script
# Sends initial tasks/prompts to AI tools running in tmux panes
#
# Usage:
#   devhive-send-task.sh <session-name> <tasks-file>
#
# Tasks file format:
#   [worker-name]
#   Task content for this worker...
#   Multiple lines supported.
#
#   [another-worker]
#   Another task...
#
# Example:
#   devhive-send-task.sh sprint-05 ~/.devhive/projects/myapp/sprints/sprint-01.tasks
#

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# =============================================================================
# Configuration
# =============================================================================

SESSION_NAME="${1:-}"
TASKS_FILE="${2:-}"

if [ -z "$SESSION_NAME" ] || [ -z "$TASKS_FILE" ]; then
    echo "Usage: devhive-send-task.sh <session-name> <tasks-file>"
    echo ""
    echo "Tasks file format:"
    echo "  [worker-name]"
    echo "  Task content..."
    echo ""
    echo "  [another-worker]"
    echo "  Another task..."
    exit 1
fi

# =============================================================================
# Colors
# =============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()  { echo -e "${BLUE}[INFO]${NC} $1"; }
log_ok()    { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# =============================================================================
# Validation
# =============================================================================

if ! tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
    log_error "tmux session not found: $SESSION_NAME"
    exit 1
fi

if [ ! -f "$TASKS_FILE" ]; then
    log_error "Tasks file not found: $TASKS_FILE"
    exit 1
fi

# =============================================================================
# Parse Tasks File
# =============================================================================

log_info "Parsing tasks file: $TASKS_FILE"

declare -A TASKS
current_worker=""

while IFS= read -r line || [ -n "$line" ]; do
    # Check for [worker-name] section header
    if [[ "$line" =~ ^\[([a-zA-Z0-9_-]+)\]$ ]]; then
        current_worker="${BASH_REMATCH[1]}"
        TASKS["$current_worker"]=""
    elif [ -n "$current_worker" ]; then
        TASKS["$current_worker"]+="$line"$'\n'
    fi
done < "$TASKS_FILE"

log_info "Found tasks for ${#TASKS[@]} workers"

# =============================================================================
# Get Worker -> Pane Mapping
# =============================================================================

# Get workers from devhive
WORKERS_JSON=$(devhive status --json 2>/dev/null || echo '{"workers":[]}')

declare -A WORKER_PANES
PANE=0
for row in $(echo "$WORKERS_JSON" | jq -r '.workers[] | @base64'); do
    WORKER_NAME=$(echo "$row" | base64 --decode | jq -r '.Name')
    WORKER_PANES["$WORKER_NAME"]=$PANE
    ((PANE++)) || true
done

# =============================================================================
# Send Tasks
# =============================================================================

echo ""
echo "╔════════════════════════════════════════════════════════════╗"
echo "║           Sending tasks to workers                          "
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

for worker in "${!TASKS[@]}"; do
    task_content="${TASKS[$worker]}"

    # Skip empty tasks
    if [ -z "$(echo "$task_content" | tr -d '[:space:]')" ]; then
        log_warn "[$worker] Empty task, skipping"
        continue
    fi

    # Get pane number
    pane="${WORKER_PANES[$worker]}"
    if [ -z "$pane" ]; then
        log_warn "[$worker] Worker not found in devhive, skipping"
        continue
    fi

    log_info "[$worker] Sending task to pane $pane..."

    # Get role file content if available
    role_content=""
    WORKER_JSON=$(echo "$WORKERS_JSON" | jq -r ".workers[] | select(.Name == \"$worker\")")
    ROLE_FILE=$(echo "$WORKER_JSON" | jq -r '.RoleFile // ""')

    if [ -n "$ROLE_FILE" ] && [ -f "$ROLE_FILE" ]; then
        role_content=$(cat "$ROLE_FILE")
        log_info "  Including role file: $ROLE_FILE"
    fi

    # Build full task (role content + task)
    full_task=""
    if [ -n "$role_content" ]; then
        full_task="以下のロール定義に従って作業してください。

--- ロール定義 ---
$role_content

--- タスク ---
$task_content"
    else
        full_task="$task_content"
    fi

    # Save to temp file for tmux buffer
    task_file="$TEMP_DIR/task-$worker.txt"
    echo "$full_task" > "$task_file"

    # Send via tmux buffer (handles long text safely)
    tmux load-buffer -b "task-$worker" "$task_file"
    tmux paste-buffer -b "task-$worker" -t "$SESSION_NAME:0.$pane"

    # Wait a bit then send Enter
    sleep 1
    tmux send-keys -t "$SESSION_NAME:0.$pane" Enter

    # Update devhive task
    DEVHIVE_WORKER="$worker" devhive worker task "タスク受信" 2>/dev/null || true

    log_ok "  Task sent"
    echo ""
done

echo "╔════════════════════════════════════════════════════════════╗"
echo "║           All tasks sent                                    "
echo "╚════════════════════════════════════════════════════════════╝"
echo ""
echo "Monitor progress:"
echo "  devhive status"
echo "  devhive watch"
echo ""
echo "Attach to session:"
echo "  tmux attach -t $SESSION_NAME"
echo ""
