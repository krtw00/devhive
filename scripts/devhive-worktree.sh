#!/bin/bash
# devhive-worktree.sh - Git Worktree作成補助スクリプト
#
# 使用法:
#   devhive-worktree.sh create <worker-name> <branch-name> [base-branch]
#   devhive-worktree.sh remove <worker-name>
#   devhive-worktree.sh list
#
# 例:
#   devhive-worktree.sh create security fix/security-auth main
#   devhive-worktree.sh remove security
#   devhive-worktree.sh list

set -e

SCRIPT_NAME=$(basename "$0")
WORKTREE_DIR="${DEVHIVE_WORKTREE_DIR:-./worktrees}"

usage() {
    cat << EOF
Usage: $SCRIPT_NAME <command> [options]

Commands:
    create <worker> <branch> [base-branch]
        Create a new worktree for a worker
        Default base-branch: main

    remove <worker>
        Remove a worker's worktree

    list
        List all worktrees

    setup <worker1:branch1> [worker2:branch2] ...
        Create multiple worktrees at once and register them in devhive

Environment:
    DEVHIVE_WORKTREE_DIR  Worktree base directory (default: ./worktrees)

Examples:
    $SCRIPT_NAME create security fix/security-auth
    $SCRIPT_NAME create frontend fix/ui-update develop
    $SCRIPT_NAME remove security
    $SCRIPT_NAME setup security:fix/security-auth frontend:fix/ui-update
EOF
    exit 1
}

# Gitリポジトリか確認
check_git_repo() {
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        echo "Error: Not a git repository" >&2
        exit 1
    fi
}

# Worktree作成
cmd_create() {
    local worker="$1"
    local branch="$2"
    local base="${3:-main}"

    if [ -z "$worker" ] || [ -z "$branch" ]; then
        echo "Error: worker name and branch required" >&2
        usage
    fi

    local worktree_path="$WORKTREE_DIR/$worker"

    # ディレクトリが既に存在するか確認
    if [ -d "$worktree_path" ]; then
        echo "Error: Worktree path already exists: $worktree_path" >&2
        exit 1
    fi

    # ベースブランチの存在確認
    if ! git rev-parse --verify "$base" > /dev/null 2>&1; then
        echo "Error: Base branch '$base' does not exist" >&2
        exit 1
    fi

    # ブランチが既に存在するか確認
    if git rev-parse --verify "$branch" > /dev/null 2>&1; then
        echo "Branch '$branch' already exists, using existing branch"
        git worktree add "$worktree_path" "$branch"
    else
        echo "Creating new branch '$branch' from '$base'"
        git worktree add -b "$branch" "$worktree_path" "$base"
    fi

    echo "Worktree created: $worktree_path (branch: $branch)"

    # 絶対パスを返す
    local abs_path
    abs_path=$(cd "$worktree_path" && pwd)
    echo "Absolute path: $abs_path"
}

# Worktree削除
cmd_remove() {
    local worker="$1"

    if [ -z "$worker" ]; then
        echo "Error: worker name required" >&2
        usage
    fi

    local worktree_path="$WORKTREE_DIR/$worker"

    if [ ! -d "$worktree_path" ]; then
        echo "Error: Worktree does not exist: $worktree_path" >&2
        exit 1
    fi

    git worktree remove "$worktree_path" --force
    echo "Worktree removed: $worktree_path"
}

# Worktree一覧
cmd_list() {
    echo "Git worktrees:"
    git worktree list
    echo ""
    echo "DevHive worktrees in $WORKTREE_DIR:"
    if [ -d "$WORKTREE_DIR" ]; then
        ls -la "$WORKTREE_DIR" 2>/dev/null || echo "  (empty)"
    else
        echo "  (directory does not exist)"
    fi
}

# 複数Worktree一括セットアップ
cmd_setup() {
    if [ $# -eq 0 ]; then
        echo "Error: at least one worker:branch required" >&2
        usage
    fi

    # worktreesディレクトリ作成
    mkdir -p "$WORKTREE_DIR"

    local workers=()
    local paths=()

    for arg in "$@"; do
        local worker="${arg%%:*}"
        local branch="${arg#*:}"

        if [ "$worker" = "$branch" ]; then
            echo "Error: Invalid format '$arg'. Use 'worker:branch'" >&2
            exit 1
        fi

        echo "=== Setting up $worker ($branch) ==="
        cmd_create "$worker" "$branch"
        echo ""

        workers+=("$worker")
        local abs_path
        abs_path=$(cd "$WORKTREE_DIR/$worker" && pwd)
        paths+=("$abs_path")
    done

    echo "=== Setup Complete ==="
    echo ""
    echo "To register workers in devhive:"
    for ((i=0; i<${#workers[@]}; i++)); do
        echo "  devhive worker register ${workers[$i]} <branch> --worktree ${paths[$i]}"
    done
    echo ""
    echo "To start tmux session:"
    local tmux_args=""
    for ((i=0; i<${#workers[@]}; i++)); do
        tmux_args+="${workers[$i]}:${paths[$i]} "
    done
    echo "  devhive-tmux.sh start <sprint-name> $tmux_args"
}

# メイン処理
check_git_repo

case "${1:-}" in
    create)
        shift
        cmd_create "$@"
        ;;
    remove)
        shift
        cmd_remove "$@"
        ;;
    list)
        cmd_list
        ;;
    setup)
        shift
        cmd_setup "$@"
        ;;
    *)
        usage
        ;;
esac
