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
        "matcher": ".*",
        "hooks": [
          {
            "type": "command",
            "command": "devhive session running 2>/dev/null || true"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "AskUserQuestion",
        "hooks": [
          {
            "type": "command",
            "command": "devhive session waiting_permission 2>/dev/null || true"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "devhive session stopped 2>/dev/null || true"
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

## 環境変数の設定

Hooksが正しく動作するには、`DEVHIVE_WORKER` 環境変数が設定されている必要があります。

### Worktree別の設定（推奨）

各Worktreeの `.envrc` ファイル（direnv使用）:

```bash
# .envrc
export DEVHIVE_WORKER=frontend
export DEVHIVE_PROJECT=myapp
```

### bashrc/zshrcに追加

```bash
# ~/.bashrc or ~/.zshrc
export DEVHIVE_WORKER=myworker
export DEVHIVE_PROJECT=myproject
```

## トラブルシューティング

### Hooksが動作しない

1. 環境変数を確認:
   ```bash
   echo $DEVHIVE_WORKER
   echo $DEVHIVE_PROJECT
   ```

2. 手動でコマンド実行:
   ```bash
   devhive session running
   ```

3. Claude Codeの設定を確認:
   ```bash
   cat ~/.claude/settings.json | jq '.hooks'
   ```
