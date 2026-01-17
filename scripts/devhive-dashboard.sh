#!/bin/bash
#
# DevHive Dashboard
# Real-time dashboard showing all workers status
#
# Usage:
#   devhive-dashboard.sh [--interval <seconds>]
#
# Examples:
#   devhive-dashboard.sh
#   devhive-dashboard.sh --interval 5
#

set -e

# =============================================================================
# Configuration
# =============================================================================

INTERVAL=2

while [[ $# -gt 0 ]]; do
    case $1 in
        --interval|-i)
            INTERVAL="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# =============================================================================
# Colors
# =============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'
BOLD='\033[1m'

# =============================================================================
# Dashboard Loop
# =============================================================================

clear

while true; do
    # Move cursor to top
    tput cup 0 0

    echo -e "${BOLD}╔════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}║                     DevHive Dashboard                               ║${NC}"
    echo -e "${BOLD}║                     $(date '+%Y-%m-%d %H:%M:%S')                              ║${NC}"
    echo -e "${BOLD}╚════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    # Check if DEVHIVE_PROJECT is set
    if [ -n "$DEVHIVE_PROJECT" ]; then
        echo -e "${CYAN}Project: $DEVHIVE_PROJECT${NC}"
        echo ""
    fi

    # Get status
    STATUS=$(devhive status 2>/dev/null || echo "No active sprint")

    echo "$STATUS"
    echo ""

    # Show attention needed
    WORKERS_JSON=$(devhive status --json 2>/dev/null || echo '{"workers":[]}')

    WAITING=$(echo "$WORKERS_JSON" | jq -r '.workers[] | select(.SessionState == "waiting_permission") | .Name' 2>/dev/null)
    ERRORS=$(echo "$WORKERS_JSON" | jq -r '.workers[] | select(.Status == "error") | .Name' 2>/dev/null)
    BLOCKED=$(echo "$WORKERS_JSON" | jq -r '.workers[] | select(.Status == "blocked") | .Name' 2>/dev/null)

    if [ -n "$WAITING" ] || [ -n "$ERRORS" ] || [ -n "$BLOCKED" ]; then
        echo -e "${BOLD}${YELLOW}⚠ Attention Needed:${NC}"

        if [ -n "$WAITING" ]; then
            for w in $WAITING; do
                echo -e "  ${YELLOW}⏸ $w: waiting_permission${NC}"
            done
        fi

        if [ -n "$ERRORS" ]; then
            for w in $ERRORS; do
                ERROR_MSG=$(echo "$WORKERS_JSON" | jq -r ".workers[] | select(.Name == \"$w\") | .LastError")
                echo -e "  ${RED}❌ $w: error - $ERROR_MSG${NC}"
            done
        fi

        if [ -n "$BLOCKED" ]; then
            for w in $BLOCKED; do
                echo -e "  ${RED}🚫 $w: blocked${NC}"
            done
        fi
        echo ""
    fi

    # Show recent events
    echo -e "${BOLD}Recent Events:${NC}"
    devhive events --limit 5 2>/dev/null || echo "  No events"
    echo ""

    # Show unread messages summary
    echo -e "${BOLD}Unread Messages:${NC}"
    for row in $(echo "$WORKERS_JSON" | jq -r '.workers[] | @base64' 2>/dev/null); do
        WORKER_NAME=$(echo "$row" | base64 --decode | jq -r '.Name')
        UNREAD=$(echo "$row" | base64 --decode | jq -r '.UnreadMessages')
        if [ "$UNREAD" -gt 0 ]; then
            echo -e "  ${CYAN}$WORKER_NAME: $UNREAD unread${NC}"
        fi
    done
    echo ""

    echo -e "${BLUE}Refresh: ${INTERVAL}s | Press Ctrl+C to exit${NC}"

    # Clear rest of screen
    tput ed

    sleep "$INTERVAL"
done
