#!/bin/bash
#
# DevHive Launch Script
# Launches AI tools (Claude Code / Codex) in tmux panes for each worker
#
# Usage:
#   devhive-launch.sh <session-name> [--tool claude|codex]
#
# Prerequisites:
#   - tmux session created by devhive-tmux.sh
#   - Workers registered with devhive
#   - DEVHIVE_PROJECT environment variable set
#
# Examples:
#   devhive-launch.sh sprint-05
#   devhive-launch.sh sprint-05 --tool codex
#

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# =============================================================================
# Configuration
# =============================================================================

SESSION_NAME="${1:-}"
TOOL="claude"  # Default tool

# Parse arguments
shift || true
while [[ $# -gt 0 ]]; do
    case $1 in
        --tool)
            TOOL="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

if [ -z "$SESSION_NAME" ]; then
    echo "Usage: devhive-launch.sh <session-name> [--tool claude|codex]"
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

# Check tmux session exists
if ! tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
    log_error "tmux session not found: $SESSION_NAME"
    echo "  Run: devhive-tmux.sh start $SESSION_NAME worker1:/path/to/worktree1 ..."
    exit 1
fi

# Check devhive is available
if ! command -v devhive &> /dev/null; then
    log_error "devhive not found in PATH"
    exit 1
fi

# Check DEVHIVE_PROJECT
if [ -z "$DEVHIVE_PROJECT" ]; then
    log_warn "DEVHIVE_PROJECT not set, using global database"
fi

# =============================================================================
# Get Workers
# =============================================================================

log_info "Fetching worker list..."

# Get workers from devhive
WORKERS_JSON=$(devhive status --json 2>/dev/null || echo '{"workers":[]}')
WORKER_COUNT=$(echo "$WORKERS_JSON" | jq -r '.workers | length')

if [ "$WORKER_COUNT" -eq 0 ]; then
    log_error "No workers registered"
    echo "  Register workers first: devhive worker register <name> <branch>"
    exit 1
fi

log_info "Found $WORKER_COUNT workers"

# =============================================================================
# Launch AI Tool in Each Pane
# =============================================================================

echo ""
echo "╔════════════════════════════════════════════════════════════╗"
echo "║           Launching $TOOL for workers                       "
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

PANE=0
for row in $(echo "$WORKERS_JSON" | jq -r '.workers[] | @base64'); do
    _jq() {
        echo "$row" | base64 --decode | jq -r "${1}"
    }

    WORKER_NAME=$(_jq '.Name')
    ROLE_NAME=$(_jq '.RoleName')
    WORKTREE=$(_jq '.WorktreePath')

    log_info "[$PANE] $WORKER_NAME (role: $ROLE_NAME)"

    # Get role args if available
    ROLE_ARGS=""
    if [ -n "$ROLE_NAME" ]; then
        ROLE_ARGS=$(devhive role show "$ROLE_NAME" --json 2>/dev/null | jq -r '.Args // ""')
    fi

    # Build command based on tool
    case $TOOL in
        claude)
            CMD="claude"
            if [ -n "$ROLE_ARGS" ]; then
                CMD="claude $ROLE_ARGS"
            fi
            ;;
        codex)
            CMD="codex"
            if [ -n "$ROLE_ARGS" ]; then
                CMD="codex $ROLE_ARGS"
            fi
            ;;
        *)
            log_error "Unknown tool: $TOOL"
            exit 1
            ;;
    esac

    # Set environment and launch
    tmux send-keys -t "$SESSION_NAME:0.$PANE" "export DEVHIVE_WORKER=$WORKER_NAME" Enter
    sleep 0.5
    tmux send-keys -t "$SESSION_NAME:0.$PANE" "devhive worker session running" Enter
    sleep 0.5
    tmux send-keys -t "$SESSION_NAME:0.$PANE" "$CMD" Enter

    log_ok "  Launched: $CMD"

    ((PANE++)) || true
done

echo ""
echo "╔════════════════════════════════════════════════════════════╗"
echo "║           All workers launched                              "
echo "╚════════════════════════════════════════════════════════════╝"
echo ""
echo "Attach to session:"
echo "  tmux attach -t $SESSION_NAME"
echo ""
echo "Check status:"
echo "  devhive status"
echo ""
echo "Send tasks:"
echo "  devhive-send-task.sh $SESSION_NAME <tasks-file>"
echo ""
