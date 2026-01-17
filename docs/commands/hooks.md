# Claude Code Hooks 連携

[← 目次に戻る](index.md)

## 概要

DevHiveはClaude Code Hooksと連携して、AIセッションの状態を自動的に追跡できます。

## セットアップ

### 自動セットアップ

```bash
# インストール
./scripts/devhive-setup-hooks.sh install

# アンインストール
./scripts/devhive-setup-hooks.sh uninstall
```

### 手動セットアップ

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
            "command": "devhive worker session running 2>/dev/null || true"
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
            "command": "devhive worker session waiting_permission 2>/dev/null || true"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "devhive worker session stopped 2>/dev/null || true"
          }
        ]
      }
    ]
  }
}
```

## Hook種類

### PreToolUse

ツール実行前に呼ばれます。セッション状態を `running` に設定。

```json
{
  "matcher": ".*",
  "hooks": [{
    "type": "command",
    "command": "devhive worker session running 2>/dev/null || true"
  }]
}
```

### PostToolUse

ツール実行後に呼ばれます。`AskUserQuestion` 後に `waiting_permission` に設定。

```json
{
  "matcher": "AskUserQuestion",
  "hooks": [{
    "type": "command",
    "command": "devhive worker session waiting_permission 2>/dev/null || true"
  }]
}
```

### Stop

セッション終了時に呼ばれます。セッション状態を `stopped` に設定。

```json
{
  "hooks": [{
    "type": "command",
    "command": "devhive worker session stopped 2>/dev/null || true"
  }]
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

Hooksが正しく動作するには、環境変数が設定されている必要があります。

### bashrc/zshrcに追加

```bash
# ~/.bashrc or ~/.zshrc
export DEVHIVE_WORKER=myworker
export DEVHIVE_PROJECT=myproject
```

### Worktree別の設定

各Worktreeの `.envrc` ファイル（direnv使用）:

```bash
# .envrc
export DEVHIVE_WORKER=frontend
export DEVHIVE_PROJECT=myapp
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
   devhive worker session running
   ```

3. Claude Codeの設定を確認:
   ```bash
   cat ~/.claude/settings.json | jq '.hooks'
   ```

### セッション状態が更新されない

1. ワーカーが登録されているか確認:
   ```bash
   devhive worker show
   ```

2. プロジェクトが正しいか確認:
   ```bash
   devhive status
   ```

## 高度な設定

### 未読メッセージの自動チェック

Stop hookでメッセージを確認:

```json
{
  "hooks": [
    {
      "type": "command",
      "command": "devhive worker session stopped 2>/dev/null; devhive msg unread 2>/dev/null || true"
    }
  ]
}
```

### タスク自動更新

特定のツール使用時にタスクを更新:

```json
{
  "matcher": "Edit|Write",
  "hooks": [{
    "type": "command",
    "command": "devhive worker task 'コード編集中' 2>/dev/null || true"
  }]
}
```
