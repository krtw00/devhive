# Claude Code Hooks 連携

[← 目次に戻る](index.md)

## 概要

DevHiveはClaude Code Hooksと連携して、AIセッションの状態を自動的に追跡できます。

## セットアップ

`~/.claude/settings.json` に以下を追加:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash|Edit|Write|NotebookEdit",
        "hooks": [
          {
            "type": "command",
            "command": "devhive session waiting_permission 2>/dev/null || true"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Bash|Edit|Write|NotebookEdit",
        "hooks": [
          {
            "type": "command",
            "command": "devhive session running 2>/dev/null || true"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "devhive session idle 2>/dev/null || true"
          }
        ]
      }
    ]
  }
}
```

## セッション状態

| 状態 | アイコン | 説明 |
|------|----------|------|
| running | ▶ | AIがアクティブに実行中 |
| waiting_permission | ⏸ | ユーザー入力/権限待ち |
| idle | ○ | セッション開いているが待機中 |
| stopped | ■ | セッション終了 |

## 環境変数

`devhive up` コマンドでworktreeを作成すると、各worktreeに `.envrc` ファイルが自動生成されます。

```bash
# .devhive/worktrees/frontend/.envrc（自動生成）
export DEVHIVE_WORKER=frontend
```

direnvを使用している場合、worktreeディレクトリに移動すると自動的に環境変数が設定されます。

## トラブルシューティング

### Hooksが動作しない

1. 環境変数を確認:
   ```bash
   echo $DEVHIVE_WORKER
   ```

2. 手動でコマンド実行:
   ```bash
   devhive session running
   ```

3. Claude Codeの設定を確認:
   ```bash
   cat ~/.claude/settings.json | jq '.hooks'
   ```
