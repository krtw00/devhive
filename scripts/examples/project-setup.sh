#!/bin/bash
#
# DevHive Project Setup Example
#
# This script demonstrates how to set up a new project for parallel development.
# Copy and customize this script for your specific project needs.
#
# Usage:
#   ./project-setup.sh <project-name> <git-repo-path>
#
# Example:
#   ./project-setup.sh my-app ~/work/my-app
#

set -e

# =============================================================================
# Configuration
# =============================================================================

PROJECT_NAME="${1:-my-project}"
GIT_REPO="${2:-$(pwd)}"
DEVHIVE_DIR="$HOME/.devhive/projects/$PROJECT_NAME"
SPRINT_ID="sprint-01"

# Worker definitions: "name:branch:role"
WORKERS=(
    "frontend:feat/frontend:frontend"
    "backend:feat/backend:backend"
)

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
# Pre-flight Checks
# =============================================================================

echo ""
echo "=========================================="
echo " DevHive Project Setup"
echo "=========================================="
echo ""

# Check devhive is installed
if ! command -v devhive &> /dev/null; then
    log_error "devhive not found in PATH"
    echo "  Install devhive first:"
    echo "    cd /path/to/devhive && go build -o devhive ./cmd/devhive"
    echo "    ln -s \$(pwd)/devhive ~/bin/devhive"
    exit 1
fi

# Check git repo exists
if [ ! -d "$GIT_REPO/.git" ]; then
    log_error "Not a git repository: $GIT_REPO"
    exit 1
fi

log_info "Project: $PROJECT_NAME"
log_info "Git Repo: $GIT_REPO"
log_info "Config Dir: $DEVHIVE_DIR"
echo ""

# =============================================================================
# Step 1: Create Project Directory Structure
# =============================================================================

log_info "Creating project directory structure..."

mkdir -p "$DEVHIVE_DIR"/{roles,sprints,worktrees}

log_ok "Directory structure created"

# =============================================================================
# Step 2: Copy Role Templates
# =============================================================================

log_info "Setting up role templates..."

DEVHIVE_REPO="$(dirname "$(dirname "$(dirname "$(realpath "$0")")")")"

if [ -f "$DEVHIVE_REPO/templates/WORKER_ROLE.md" ]; then
    cp "$DEVHIVE_REPO/templates/WORKER_ROLE.md" "$DEVHIVE_DIR/roles/worker.md"
    log_ok "Copied WORKER_ROLE.md template"
else
    log_warn "Template not found, creating minimal role file"
    cat > "$DEVHIVE_DIR/roles/worker.md" << 'EOF'
# Worker Role

## Work Start
```bash
devhive worker start --task "Your task"
devhive msg unread
devhive status
```

## Completion
```bash
devhive worker complete
```
EOF
fi

# =============================================================================
# Step 3: Create Sprint Configuration
# =============================================================================

log_info "Creating sprint configuration..."

cat > "$DEVHIVE_DIR/sprints/$SPRINT_ID.conf" << EOF
# Sprint: $SPRINT_ID
# Project: $PROJECT_NAME
# Created: $(date '+%Y-%m-%d')

# Format: worker_name:branch_name:issue_reference
EOF

for worker_def in "${WORKERS[@]}"; do
    IFS=':' read -r name branch role <<< "$worker_def"
    echo "$name:$branch:#000" >> "$DEVHIVE_DIR/sprints/$SPRINT_ID.conf"
done

log_ok "Sprint configuration created: $DEVHIVE_DIR/sprints/$SPRINT_ID.conf"

# =============================================================================
# Step 4: Initialize DevHive Sprint
# =============================================================================

log_info "Initializing DevHive sprint..."

export DEVHIVE_PROJECT="$PROJECT_NAME"
devhive init "$SPRINT_ID"

log_ok "Sprint initialized"

# =============================================================================
# Step 5: Create Worktrees and Register Workers
# =============================================================================

log_info "Creating worktrees and registering workers..."

cd "$GIT_REPO"

# Get base branch (main or master or develop)
BASE_BRANCH=$(git branch --list main master develop 2>/dev/null | head -1 | tr -d '* ')
if [ -z "$BASE_BRANCH" ]; then
    BASE_BRANCH=$(git rev-parse --abbrev-ref HEAD)
fi

for worker_def in "${WORKERS[@]}"; do
    IFS=':' read -r name branch role <<< "$worker_def"

    worktree_path="$DEVHIVE_DIR/worktrees/$name"

    log_info "  Creating worktree: $name -> $branch"

    # Create worktree
    git worktree add -b "$branch" "$worktree_path" "$BASE_BRANCH" 2>/dev/null || \
    git worktree add "$worktree_path" "$branch" 2>/dev/null || \
    log_warn "  Worktree already exists or branch conflict"

    # Copy role file as CLAUDE.md
    if [ -f "$DEVHIVE_DIR/roles/$role.md" ]; then
        cp "$DEVHIVE_DIR/roles/$role.md" "$worktree_path/CLAUDE.md"
    else
        cp "$DEVHIVE_DIR/roles/worker.md" "$worktree_path/CLAUDE.md"
    fi

    # Register worker
    devhive worker register "$name" "$branch" --role "$role" --worktree "$worktree_path" 2>/dev/null || \
    log_warn "  Worker already registered: $name"
done

log_ok "Worktrees created and workers registered"

# =============================================================================
# Step 6: Summary
# =============================================================================

echo ""
echo "=========================================="
echo " Setup Complete!"
echo "=========================================="
echo ""
echo "Project directory: $DEVHIVE_DIR"
echo ""
echo "Next steps:"
echo ""
echo "  1. Set environment variable:"
echo "     export DEVHIVE_PROJECT=$PROJECT_NAME"
echo ""
echo "  2. Start tmux session:"
echo "     devhive-tmux.sh start $SPRINT_ID \\"
for worker_def in "${WORKERS[@]}"; do
    IFS=':' read -r name branch role <<< "$worker_def"
    echo "       $name:$DEVHIVE_DIR/worktrees/$name \\"
done
echo ""
echo "  3. Check status:"
echo "     devhive status"
echo ""
echo "  4. Customize role files:"
echo "     $DEVHIVE_DIR/roles/"
echo ""
