# DevHive

Git Worktree + 複数AIエージェントによる並列開発の状態管理CLIツール

## 特徴

- **Single Source of Truth**: SQLiteで全ての状態を一元管理
- **環境非依存**: tmux/screen/複数ターミナル、どの環境でも動作
- **高速起動**: Go製バイナリで即座に応答
- **シンプル**: 状態管理に専念、プロセス管理は外部に委譲

## インストール

```bash
# ビルド
go build -o devhive ./cmd/devhive

# パスに追加
sudo ln -s $(pwd)/devhive /usr/local/bin/devhive
```

## クイックスタート

### 1. スプリント開始

```bash
devhive init sprint-01
```

### 2. ワーカー登録

```bash
devhive worker register security fix/security-auth --issue "#101"
devhive worker register frontend fix/ui-update --issue "#102"
```

### 3. ワーカー側の設定

各ワーカー環境で環境変数を設定:

```bash
export DEVHIVE_WORKER=security
```

### 4. 作業開始

```bash
devhive worker start --task "認証APIの実装"
```

### 5. 状態確認

```bash
devhive status
```

```
Sprint: sprint-01 (started: 2025-01-18 10:00)

WORKER    BRANCH             ISSUE  STATUS   TASK              MSGS
------    ------             -----  ------   ----              ----
security  fix/security-auth  #101   working  認証APIの実装     0
frontend  fix/ui-update      #102   pending                    0
```

### 6. ワーカー間通信

```bash
# メッセージ送信
devhive msg send frontend "認証APIの仕様が変わりました"

# 未読確認
devhive msg unread

# 全員に通知
devhive msg broadcast "15分後にマージします"
```

### 7. 状態監視

```bash
# 全ての変化を監視
devhive watch

# メッセージのみ監視
devhive watch --filter=message
```

## コマンド一覧

| コマンド | 説明 |
|----------|------|
| `devhive init <sprint-id>` | スプリント初期化 |
| `devhive status` | 全体状態表示 |
| `devhive sprint complete` | スプリント完了 |
| `devhive worker register <name> <branch>` | ワーカー登録 |
| `devhive worker start` | 作業開始 |
| `devhive worker complete` | 作業完了 |
| `devhive worker show` | 詳細表示 |
| `devhive worker task <task>` | タスク更新 |
| `devhive worker error <message>` | エラー報告 |
| `devhive msg send <to> <message>` | メッセージ送信 |
| `devhive msg broadcast <message>` | 全員に送信 |
| `devhive msg unread` | 未読確認 |
| `devhive msg read <id\|all>` | 既読化 |
| `devhive events` | イベントログ |
| `devhive watch` | 状態監視 |

詳細は [docs/commands.md](docs/commands.md) を参照。

## 環境変数

| 変数名 | 説明 |
|--------|------|
| `DEVHIVE_WORKER` | デフォルトのワーカー名 |

## アーキテクチャ

```
┌─────────────────────────────────────────────┐
│            DevHive Core CLI                  │
│         (状態管理・環境非依存)               │
└──────────────────┬──────────────────────────┘
                   │
              ┌────┴────┐
              │ SQLite  │
              │ ~/.devhive/state.db
              └─────────┘
```

DevHiveは状態管理に専念し、実行環境（tmux, screen等）との連携はオプションのスクリプトで提供。

## データベース

```
~/.devhive/state.db
```

## ドキュメント

- [設計書](docs/design.md)
- [コマンドリファレンス](docs/commands.md)

## License

MIT
