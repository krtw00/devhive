#!/bin/bash
#
# DevHive Claude Code Hooks Setup
# Configures Claude Code to automatically update DevHive session state
#
# Usage:
#   devhive-setup-hooks.sh [--install|--uninstall|--show]
#
# This script modifies ~/.claude/settings.json to add hooks that:
#   - Set session to 'waiting_permission' before tool use
#   - Set session to 'running' after tool use
#   - Set session to 'idle' when Claude stops
#

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLAUDE_SETTINGS="$HOME/.claude/settings.json"
TEMPLATE_FILE="$SCRIPT_DIR/../templates/claude-hooks.json"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()  { echo -e "${BLUE}[INFO]${NC} $1"; }
log_ok()    { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

show_usage() {
    echo "DevHive Claude Code Hooks Setup"
    echo ""
    echo "Usage:"
    echo "  devhive-setup-hooks.sh --install    Install DevHive hooks"
    echo "  devhive-setup-hooks.sh --uninstall  Remove DevHive hooks"
    echo "  devhive-setup-hooks.sh --show       Show current hooks config"
    echo ""
    echo "Prerequisites:"
    echo "  - Claude Code CLI installed"
    echo "  - devhive in PATH"
    echo "  - DEVHIVE_WORKER environment variable set per session"
}

check_prerequisites() {
    if ! command -v devhive &> /dev/null; then
        log_error "devhive not found in PATH"
        echo "  Install devhive first and add to PATH"
        exit 1
    fi

    if ! command -v jq &> /dev/null; then
        log_error "jq not found (required for JSON manipulation)"
        echo "  Install: apt install jq / brew install jq"
        exit 1
    fi
}

show_hooks() {
    if [ ! -f "$CLAUDE_SETTINGS" ]; then
        log_info "No Claude settings file found at $CLAUDE_SETTINGS"
        return
    fi

    log_info "Current hooks configuration:"
    jq '.hooks // "No hooks configured"' "$CLAUDE_SETTINGS"
}

install_hooks() {
    check_prerequisites

    log_info "Installing DevHive hooks for Claude Code..."

    # Ensure .claude directory exists
    mkdir -p "$(dirname "$CLAUDE_SETTINGS")"

    # Create settings file if it doesn't exist
    if [ ! -f "$CLAUDE_SETTINGS" ]; then
        echo '{}' > "$CLAUDE_SETTINGS"
        log_info "Created new settings file"
    fi

    # Backup existing settings
    BACKUP_FILE="$CLAUDE_SETTINGS.backup.$(date +%Y%m%d_%H%M%S)"
    cp "$CLAUDE_SETTINGS" "$BACKUP_FILE"
    log_info "Backed up existing settings to $BACKUP_FILE"

    # Check if hooks already exist
    if jq -e '.hooks' "$CLAUDE_SETTINGS" > /dev/null 2>&1; then
        log_warn "Existing hooks found. Merging DevHive hooks..."

        # Merge hooks (DevHive hooks will be added/updated)
        DEVHIVE_HOOKS=$(cat "$TEMPLATE_FILE")

        # Use jq to merge
        jq --argjson devhive "$DEVHIVE_HOOKS" '
            .hooks.PreToolUse = ((.hooks.PreToolUse // []) + $devhive.hooks.PreToolUse | unique_by(.matcher // .hooks[0].command)) |
            .hooks.PostToolUse = ((.hooks.PostToolUse // []) + $devhive.hooks.PostToolUse | unique_by(.matcher // .hooks[0].command)) |
            .hooks.Stop = ((.hooks.Stop // []) + $devhive.hooks.Stop | unique_by(.hooks[0].command))
        ' "$CLAUDE_SETTINGS" > "$CLAUDE_SETTINGS.tmp"

        mv "$CLAUDE_SETTINGS.tmp" "$CLAUDE_SETTINGS"
    else
        # No existing hooks, just add DevHive hooks
        DEVHIVE_HOOKS=$(cat "$TEMPLATE_FILE")
        jq --argjson devhive "$DEVHIVE_HOOKS" '. + $devhive' "$CLAUDE_SETTINGS" > "$CLAUDE_SETTINGS.tmp"
        mv "$CLAUDE_SETTINGS.tmp" "$CLAUDE_SETTINGS"
    fi

    log_ok "DevHive hooks installed successfully!"
    echo ""
    echo "Usage:"
    echo "  1. Set DEVHIVE_WORKER before starting Claude Code:"
    echo "     export DEVHIVE_WORKER=<worker-name>"
    echo ""
    echo "  2. Or use a .devhive file in your project root:"
    echo "     echo 'project-name' > /path/to/project/.devhive"
    echo ""
    echo "  3. Session state will be automatically updated:"
    echo "     - waiting_permission: before tool execution"
    echo "     - running: after tool execution"
    echo "     - idle: when Claude stops"
}

uninstall_hooks() {
    if [ ! -f "$CLAUDE_SETTINGS" ]; then
        log_info "No Claude settings file found"
        return
    fi

    log_info "Removing DevHive hooks..."

    # Backup
    BACKUP_FILE="$CLAUDE_SETTINGS.backup.$(date +%Y%m%d_%H%M%S)"
    cp "$CLAUDE_SETTINGS" "$BACKUP_FILE"
    log_info "Backed up settings to $BACKUP_FILE"

    # Remove DevHive hooks (those containing 'devhive worker session')
    jq '
        if .hooks then
            .hooks.PreToolUse = [.hooks.PreToolUse[]? | select(.hooks[0].command | contains("devhive") | not)] |
            .hooks.PostToolUse = [.hooks.PostToolUse[]? | select(.hooks[0].command | contains("devhive") | not)] |
            .hooks.Stop = [.hooks.Stop[]? | select(.hooks[0].command | contains("devhive") | not)] |
            if (.hooks.PreToolUse | length) == 0 then del(.hooks.PreToolUse) else . end |
            if (.hooks.PostToolUse | length) == 0 then del(.hooks.PostToolUse) else . end |
            if (.hooks.Stop | length) == 0 then del(.hooks.Stop) else . end |
            if (.hooks | keys | length) == 0 then del(.hooks) else . end
        else .
        end
    ' "$CLAUDE_SETTINGS" > "$CLAUDE_SETTINGS.tmp"

    mv "$CLAUDE_SETTINGS.tmp" "$CLAUDE_SETTINGS"
    log_ok "DevHive hooks removed"
}

# Main
case "${1:-}" in
    --install)
        install_hooks
        ;;
    --uninstall)
        uninstall_hooks
        ;;
    --show)
        show_hooks
        ;;
    *)
        show_usage
        exit 1
        ;;
esac
