# DevHive

Parallel development coordination tool using SQLite.

tmux + Git Worktree + multiple Claude Code instances による並列開発を管理するCLIツール。

## Features

- **Single Source of Truth**: SQLiteで全ての状態を一元管理
- **Worker Management**: ワーカーの状態追跡
- **Review Workflow**: レビュー依頼・承認フロー
- **Message System**: ワーカー間メッセージング
- **File Locking**: 競合防止のためのファイルロック
- **Event Logging**: 全操作の監査ログ
- **Fast**: Go製で高速起動

## Installation

```bash
# ビルド
cd ~/work/devhive
go build -o devhive ./cmd/devhive

# パスに追加（例）
sudo ln -s ~/work/devhive/devhive /usr/local/bin/devhive
```

## Quick Start

```bash
# スプリント初期化
devhive init sprint-01 --config path/to/sprint.conf

# ワーカー登録
devhive worker register security fix/security-auth -i "#313" -p 1

# 状態確認
devhive status

# ワーカー開始
devhive worker start security

# レビュー依頼
devhive review request abc1234 -w security -d "認証機能の追加"

# レビュー一覧
devhive review list

# レビュー承認
devhive review ok 1 "問題なし"

# メッセージ送信
devhive msg send mobile-layout "DuelTable.vueを編集予定" -f security

# 未読確認
devhive msg unread security

# ファイルロック
devhive lock acquire src/components/Foo.vue -w security
devhive lock release src/components/Foo.vue -w security
devhive lock list

# イベントログ
devhive events --limit 20
```

## Commands

| Command | Description |
|---------|-------------|
| `devhive init <sprint-id>` | スプリント初期化 |
| `devhive status` | 全体状態表示 |
| `devhive worker register <name> <branch>` | ワーカー登録 |
| `devhive worker start <name>` | ワーカー開始 |
| `devhive worker complete <name>` | ワーカー完了 |
| `devhive review request <commit>` | レビュー依頼 |
| `devhive review list` | 未処理レビュー一覧 |
| `devhive review ok <id> [comment]` | レビュー承認 |
| `devhive review fix <id> <comment>` | 要修正 |
| `devhive msg send <to> <message>` | メッセージ送信 |
| `devhive msg broadcast <message>` | 全員に送信 |
| `devhive msg unread [worker]` | 未読確認 |
| `devhive lock acquire <file>` | ファイルロック |
| `devhive lock release <file>` | ロック解除 |
| `devhive lock list` | ロック一覧 |
| `devhive events` | イベントログ |

## Database Location

```
~/.devhive/state.db
```

## License

MIT
