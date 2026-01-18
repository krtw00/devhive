# DevHive

Git Worktree + 複数AIエージェントによる並列開発の状態管理CLIツール

## 特徴

- **Single Source of Truth**: SQLiteで全ての状態を一元管理
- **プロジェクト分離**: プロジェクトごとに独立したDB
- **環境非依存**: tmux/screen/複数ターミナル、どの環境でも動作
- **高速起動**: Go製バイナリで即座に応答
- **Claude Code連携**: Hooksによるセッション状態の自動追跡
- **direnv連携**: Worktreeごとの環境変数自動設定

## インストール

```bash
# ビルド
go build -o ~/bin/devhive ./cmd/devhive

# PATHに追加（~/.zshrc or ~/.bashrc）
export PATH="$HOME/bin:$PATH"
```

## クイックスタート

### 1. プロジェクト設定

```bash
cd /path/to/your-project
echo "your-project-name" > .devhive
```

### 2. スプリント開始

```bash
devhive init sprint-01
```

### 3. ロール作成

```bash
devhive role create frontend -d "フロントエンド担当" -f roles/frontend.md
devhive role create backend -d "バックエンド担当" -f roles/backend.md
```

### 4. ワーカー登録（Worktree自動作成）

```bash
devhive worker register fe-auth feat/auth-ui --role frontend --create-worktree
devhive worker register be-api feat/api --role backend --create-worktree
```

### 5. 状態確認

```bash
devhive status
```

```
Project: your-project
Sprint: sprint-01 (started: 2025-01-18 10:00)

WORKER   ROLE      BRANCH        STATUS     SESSION    TASK  MSGS
------   ----      ------        ------     -------    ----  ----
fe-auth  frontend  feat/auth-ui  ⏳ pending  ■ stopped        0
be-api   backend   feat/api      ⏳ pending  ■ stopped        0
```

### 6. ワーカーとして作業

```bash
# Worktreeに移動（direnvで環境変数が自動設定）
cd ~/.devhive/projects/your-project/worktrees/fe-auth
direnv allow  # 初回のみ

# 作業開始
devhive worker start --task "認証UIの実装"

# Claude Codeを起動（Hooksでセッション状態が自動追跡）
claude
```

## ドキュメント

| ドキュメント | 説明 |
|-------------|------|
| [ユーザーガイド](docs/user-guide.md) | 使い方の詳細 |
| [仕様書](docs/specification.md) | 技術仕様 |
| [コマンドリファレンス](docs/commands/index.md) | 全コマンドの詳細 |
| [設計書](docs/design.md) | アーキテクチャ設計 |

## コマンド一覧

### スプリント管理

| コマンド | 説明 |
|----------|------|
| `devhive init <sprint-id>` | スプリント初期化 |
| `devhive status` | 現在のステータス表示 |
| `devhive projects` | 全プロジェクト一覧 |
| `devhive sprint complete` | スプリント完了 |
| `devhive sprint setup <file>` | 一括ワーカー登録 |
| `devhive sprint report` | レポート生成 |

### ワーカー管理

| コマンド | 説明 |
|----------|------|
| `devhive worker register <name> <branch>` | ワーカー登録 |
| `devhive worker start` | 作業開始 |
| `devhive worker complete` | 作業完了 |
| `devhive worker status <status>` | ステータス更新 |
| `devhive worker session <state>` | セッション状態更新 |
| `devhive worker show` | 詳細表示 |
| `devhive worker task <task>` | タスク更新 |
| `devhive worker error <message>` | エラー報告 |

### ロール管理

| コマンド | 説明 |
|----------|------|
| `devhive role create <name>` | ロール作成 |
| `devhive role list` | ロール一覧 |
| `devhive role show <name>` | ロール詳細 |
| `devhive role update <name>` | ロール更新 |
| `devhive role delete <name>` | ロール削除 |

### メッセージ

| コマンド | 説明 |
|----------|------|
| `devhive msg send <to> <message>` | メッセージ送信 |
| `devhive msg broadcast <message>` | 全員に送信 |
| `devhive msg unread` | 未読確認 |
| `devhive msg read <id\|all>` | 既読化 |

### 監視・クリーンアップ

| コマンド | 説明 |
|----------|------|
| `devhive events` | イベントログ |
| `devhive watch` | リアルタイム監視 |
| `devhive cleanup all` | 古いデータ削除 |

## 環境変数

| 変数名 | 説明 |
|--------|------|
| `DEVHIVE_PROJECT` | プロジェクト名（.devhiveファイルで自動検出可） |
| `DEVHIVE_WORKER` | ワーカー名（direnvで自動設定可） |

## プロジェクト検出優先順位

1. `--project` / `-P` フラグ（最優先）
2. `.devhive` ファイル（cwdから上位へ検索）
3. パス検出（`~/.devhive/projects/<name>/...` 配下）
4. `DEVHIVE_PROJECT` 環境変数（最低優先度）

## アーキテクチャ

```
プロジェクト (your-project)
  └── スプリント (sprint-01)
        └── ワーカー (fe-auth, be-api, ...)
              ├── ロール (frontend, backend, ...)
              ├── ブランチ (feat/auth-ui)
              ├── Worktree (~/.devhive/projects/your-project/worktrees/fe-auth)
              └── タスク (具体的な作業内容)
```

```
~/.devhive/
├── projects/
│   └── your-project/
│       ├── devhive.db        # SQLite DB
│       └── worktrees/        # Git Worktrees
│           ├── fe-auth/
│           │   └── .envrc    # direnv設定
│           └── be-api/
│               └── .envrc
```

## Claude Code Hooks

`~/.claude/settings.json` に設定することで、セッション状態が自動追跡される:

```json
{
  "hooks": {
    "PreToolUse": [{
      "matcher": "Bash|Edit|Write|NotebookEdit",
      "hooks": [{
        "type": "command",
        "command": "devhive worker session waiting_permission 2>/dev/null || true"
      }]
    }],
    "PostToolUse": [{
      "matcher": "Bash|Edit|Write|NotebookEdit",
      "hooks": [{
        "type": "command",
        "command": "devhive worker session running 2>/dev/null || true"
      }]
    }],
    "Stop": [{
      "hooks": [{
        "type": "command",
        "command": "devhive worker session idle 2>/dev/null || true"
      }]
    }]
  }
}
```

## License

MIT
