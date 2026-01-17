#!/bin/bash
# devhive-tmux.sh - tmuxセッション起動・管理スクリプト
#
# 使用法:
#   devhive-tmux.sh start <session-name> <worker1:path1> [worker2:path2] ...
#   devhive-tmux.sh attach <session-name>
#   devhive-tmux.sh stop <session-name>
#
# 例:
#   devhive-tmux.sh start sprint-01 security:./worktrees/security frontend:./worktrees/frontend
#   devhive-tmux.sh attach sprint-01
#   devhive-tmux.sh stop sprint-01

set -e

SCRIPT_NAME=$(basename "$0")

usage() {
    cat << EOF
Usage: $SCRIPT_NAME <command> [options]

Commands:
    start <session> <worker:path> [worker:path ...]
        Start a new tmux session with worker panes

    attach <session>
        Attach to an existing session

    stop <session>
        Stop and kill the session

    list
        List active devhive sessions

Examples:
    $SCRIPT_NAME start sprint-01 security:./worktrees/security frontend:./worktrees/frontend
    $SCRIPT_NAME attach sprint-01
    $SCRIPT_NAME stop sprint-01
EOF
    exit 1
}

# tmuxがインストールされているか確認
check_tmux() {
    if ! command -v tmux &> /dev/null; then
        echo "Error: tmux is not installed" >&2
        exit 1
    fi
}

# セッション開始
cmd_start() {
    local session_name="$1"
    shift

    if [ -z "$session_name" ] || [ $# -eq 0 ]; then
        echo "Error: session name and at least one worker:path required" >&2
        usage
    fi

    # セッションが既に存在するか確認
    if tmux has-session -t "$session_name" 2>/dev/null; then
        echo "Session '$session_name' already exists. Use 'attach' or 'stop' first." >&2
        exit 1
    fi

    # ワーカー情報をパース
    local workers=()
    local paths=()
    for arg in "$@"; do
        local worker="${arg%%:*}"
        local path="${arg#*:}"
        workers+=("$worker")
        paths+=("$path")
    done

    local num_workers=${#workers[@]}
    echo "Starting session '$session_name' with $num_workers workers..."

    # 最初のワーカーでセッション作成
    local first_path="${paths[0]}"
    local first_worker="${workers[0]}"

    # パスを絶対パスに変換
    first_path=$(cd "$first_path" 2>/dev/null && pwd || echo "$first_path")

    tmux new-session -d -s "$session_name" -c "$first_path"
    tmux send-keys -t "$session_name" "export DEVHIVE_WORKER=$first_worker" Enter
    tmux send-keys -t "$session_name" "clear && echo 'Worker: $first_worker' && echo 'Path: $first_path'" Enter

    # 残りのワーカー用にペインを作成
    for ((i=1; i<num_workers; i++)); do
        local worker="${workers[$i]}"
        local path="${paths[$i]}"
        path=$(cd "$path" 2>/dev/null && pwd || echo "$path")

        # 水平分割でペインを追加
        tmux split-window -t "$session_name" -h -c "$path"
        tmux send-keys -t "$session_name" "export DEVHIVE_WORKER=$worker" Enter
        tmux send-keys -t "$session_name" "clear && echo 'Worker: $worker' && echo 'Path: $path'" Enter
    done

    # レイアウトを均等に調整
    tmux select-layout -t "$session_name" tiled

    # 監視用ペインを追加（オプション）
    # tmux split-window -t "$session_name" -v -l 10
    # tmux send-keys -t "$session_name" "devhive watch" Enter

    echo "Session '$session_name' started with workers: ${workers[*]}"
    echo "Run '$SCRIPT_NAME attach $session_name' to connect"
}

# セッションにアタッチ
cmd_attach() {
    local session_name="$1"

    if [ -z "$session_name" ]; then
        echo "Error: session name required" >&2
        usage
    fi

    if ! tmux has-session -t "$session_name" 2>/dev/null; then
        echo "Session '$session_name' does not exist" >&2
        exit 1
    fi

    tmux attach-session -t "$session_name"
}

# セッション停止
cmd_stop() {
    local session_name="$1"

    if [ -z "$session_name" ]; then
        echo "Error: session name required" >&2
        usage
    fi

    if ! tmux has-session -t "$session_name" 2>/dev/null; then
        echo "Session '$session_name' does not exist" >&2
        exit 1
    fi

    tmux kill-session -t "$session_name"
    echo "Session '$session_name' stopped"
}

# セッション一覧
cmd_list() {
    echo "Active tmux sessions:"
    tmux list-sessions 2>/dev/null || echo "  (none)"
}

# メイン処理
check_tmux

case "${1:-}" in
    start)
        shift
        cmd_start "$@"
        ;;
    attach)
        shift
        cmd_attach "$@"
        ;;
    stop)
        shift
        cmd_stop "$@"
        ;;
    list)
        cmd_list
        ;;
    *)
        usage
        ;;
esac
