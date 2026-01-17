#!/bin/bash
# devhive-quickstart.sh - DevHive並列開発環境の一括セットアップ
#
# 使用法:
#   devhive-quickstart.sh <sprint-id> <worker1:branch1> [worker2:branch2] ...
#
# 例:
#   devhive-quickstart.sh sprint-01 security:fix/security-auth frontend:fix/ui-update

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKTREE_DIR="${DEVHIVE_WORKTREE_DIR:-./worktrees}"
TEMPLATE_DIR="$(dirname "$SCRIPT_DIR")/templates"

usage() {
    cat << EOF
Usage: $(basename "$0") <sprint-id> <worker:branch> [worker:branch ...]

This script will:
  1. Initialize a new DevHive sprint
  2. Create Git worktrees for each worker
  3. Copy CLAUDE.md template to each worktree
  4. Register workers in DevHive
  5. Start a tmux session with all workers

Example:
  $(basename "$0") sprint-01 security:fix/security-auth frontend:fix/ui-update

After running, you'll have:
  - A DevHive sprint initialized
  - Git worktrees in $WORKTREE_DIR/
  - CLAUDE.md in each worktree
  - A tmux session with panes for each worker

EOF
    exit 1
}

# 引数チェック
if [ $# -lt 2 ]; then
    usage
fi

SPRINT_ID="$1"
shift

# devhiveが利用可能か確認
if ! command -v devhive &> /dev/null; then
    echo "Error: devhive command not found. Build it first:" >&2
    echo "  go build -o devhive ./cmd/devhive" >&2
    echo "  sudo ln -s \$(pwd)/devhive /usr/local/bin/devhive" >&2
    exit 1
fi

# Gitリポジトリか確認
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "Error: Not a git repository" >&2
    exit 1
fi

echo "=== DevHive Quick Start ==="
echo "Sprint: $SPRINT_ID"
echo "Workers: $*"
echo ""

# 1. スプリント初期化
echo ">>> Initializing sprint..."
if devhive init "$SPRINT_ID"; then
    echo "Sprint '$SPRINT_ID' initialized"
else
    echo "Sprint may already exist, continuing..."
fi
echo ""

# ワーカー情報をパース
workers=()
branches=()
paths=()

for arg in "$@"; do
    worker="${arg%%:*}"
    branch="${arg#*:}"

    if [ "$worker" = "$branch" ]; then
        echo "Error: Invalid format '$arg'. Use 'worker:branch'" >&2
        exit 1
    fi

    workers+=("$worker")
    branches+=("$branch")
done

# 2. Worktree作成
echo ">>> Creating worktrees..."
mkdir -p "$WORKTREE_DIR"

for ((i=0; i<${#workers[@]}; i++)); do
    worker="${workers[$i]}"
    branch="${branches[$i]}"
    worktree_path="$WORKTREE_DIR/$worker"

    if [ -d "$worktree_path" ]; then
        echo "Worktree already exists: $worktree_path"
    else
        if git rev-parse --verify "$branch" > /dev/null 2>&1; then
            echo "Using existing branch '$branch'"
            git worktree add "$worktree_path" "$branch"
        else
            echo "Creating new branch '$branch' from main"
            git worktree add -b "$branch" "$worktree_path" main
        fi
    fi

    abs_path=$(cd "$worktree_path" && pwd)
    paths+=("$abs_path")
    echo "  $worker: $abs_path"
done
echo ""

# 3. CLAUDE.mdをコピー
echo ">>> Copying CLAUDE.md templates..."
if [ -f "$TEMPLATE_DIR/CLAUDE.md" ]; then
    for ((i=0; i<${#workers[@]}; i++)); do
        worker="${workers[$i]}"
        worktree_path="${paths[$i]}"

        if [ ! -f "$worktree_path/CLAUDE.md" ]; then
            cp "$TEMPLATE_DIR/CLAUDE.md" "$worktree_path/CLAUDE.md"
            echo "  Copied to $worktree_path/CLAUDE.md"
        else
            echo "  CLAUDE.md already exists in $worktree_path"
        fi
    done
else
    echo "  Warning: Template not found at $TEMPLATE_DIR/CLAUDE.md"
fi
echo ""

# 4. ワーカー登録
echo ">>> Registering workers in DevHive..."
for ((i=0; i<${#workers[@]}; i++)); do
    worker="${workers[$i]}"
    branch="${branches[$i]}"
    worktree_path="${paths[$i]}"

    devhive worker register "$worker" "$branch" --worktree "$worktree_path" || true
    echo "  Registered: $worker"
done
echo ""

# 5. 状態確認
echo ">>> Current status:"
devhive status
echo ""

# 6. tmuxセッション開始
echo ">>> Starting tmux session..."
tmux_args=""
for ((i=0; i<${#workers[@]}; i++)); do
    tmux_args+="${workers[$i]}:${paths[$i]} "
done

if tmux has-session -t "$SPRINT_ID" 2>/dev/null; then
    echo "Session '$SPRINT_ID' already exists."
    echo "Run: tmux attach-session -t $SPRINT_ID"
else
    "$SCRIPT_DIR/devhive-tmux.sh" start "$SPRINT_ID" $tmux_args
    echo ""
    echo "=== Setup Complete ==="
    echo ""
    echo "To attach to the session:"
    echo "  tmux attach-session -t $SPRINT_ID"
    echo ""
    echo "In each pane, start Claude Code:"
    echo "  claude"
    echo ""
    echo "The DEVHIVE_WORKER environment variable is already set in each pane."
fi
