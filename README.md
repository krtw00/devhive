# DevHive

> **⚠️ MIGRATION NOTICE (2026-01-20)**
>
> DevHiveの機能は[agentmine](https://github.com/krtw00/agentmine)に統合されます。
>
> **Why agentmine?**
> - ✅ **DevHive機能を全て含む** - worktree隔離、並列AI実行、状態管理
> - ✅ **強化されたセキュリティ** - スコープ制御（sparse-checkout + chmod）
> - ✅ **チーム協業対応** - Redmine的な共有DB、Web UI
> - ✅ **Memory Bank** - プロジェクト知識の永続化
> - ✅ **MCP統合** - Cursor/Windsurf等のIDE連携
>
> **Migration Path:**
> ```bash
> # DevHive（現在）
> devhive up
> devhive ps
>
> # agentmine（移行後）
> agentmine worker run <task-id> --exec
> agentmine worker status
> ```
>
> **Status:** DevHiveは保守モードに移行。新機能開発はagentmineで継続。
>
> **Repository:** https://github.com/krtw00/agentmine

---

Git Worktree + 複数AIエージェントによる並列開発の状態管理CLIツール。Docker風インターフェース。

## 特徴

- **Docker風操作**: `up`, `down`, `ps`, `logs` など直感的なコマンド
- **1ファイル設定**: `.devhive.yaml` で全て定義
- **自己完結**: プロジェクト内に全データ配置

## インストール

```bash
go build -o ~/bin/devhive ./cmd/devhive
```

## クイックスタート

### 1. 初期化

```bash
devhive init --template   # .devhive/ と テンプレート作成
```

### 2. 設定編集

```yaml
# .devhive.yaml
workers:
  frontend:
    branch: feat/ui
    role: "@frontend"
    task: フロントエンド実装
    tool: claude        # AIツール指定（claude/codex/gemini/generic）

  backend:
    branch: feat/api
    role: "@backend"
    task: API実装
    tool: codex
```

### 3. 起動

```bash
devhive up       # 全て自動セットアップ
devhive ps       # 状態確認
```

### 4. 完了

```bash
devhive down
```

## プロジェクト構成

```
myapp/
├── .devhive.yaml        # 設定ファイル（git管理）
└── .devhive/            # DevHiveデータ（gitignore）
    ├── devhive.db       # 状態DB
    ├── worktrees/       # Git Worktrees
    │   ├── frontend/
    │   └── backend/
    ├── roles/           # ロール定義
    │   └── custom.md
    ├── tasks/           # タスク詳細
    │   └── frontend.md
    └── workers/         # ワーカー管理情報
        └── frontend.md  # メモ、進捗ノート等
```

### MDファイルによる長文管理

タスクやロールが長い場合、MDファイルに分離:

```yaml
# .devhive.yaml
workers:
  frontend:
    branch: feat/ui
    role: "@frontend"
    # task省略 → .devhive/tasks/frontend.md を読み込み
```

```markdown
<!-- .devhive/tasks/frontend.md -->
# フロントエンド実装

## 目標
- 認証UIの実装
- ダッシュボード作成

## 詳細仕様
...
```

## コンテキストファイル自動生成

`devhive up` 時、各Worktreeにコンテキストファイルを自動生成:

| ファイル | 生成条件 | 用途 |
|----------|----------|------|
| `CONTEXT.md` | 常に | 汎用コンテキスト |
| `CLAUDE.md` | tool: claude | Claude Code用 |
| `AGENTS.md` | tool: codex | Codex用 |
| `GEMINI.md` | tool: gemini | Gemini用 |

これによりAIツールが自動的にロール・タスク情報を読み込みます。

## コマンド一覧

### 基本操作

| コマンド | 説明 |
|----------|------|
| `devhive init [-t]` | 初期化（-t: テンプレート作成） |
| `devhive up` | ワーカー起動 |
| `devhive down` | ワーカー停止 |
| `devhive ps` | ワーカー一覧 |
| `devhive status` | 全体サマリー |
| `devhive logs [-f]` | ログ表示 |

### ユーティリティ

| コマンド | 説明 |
|----------|------|
| `devhive progress <w> <0-100>` | 進捗更新 |
| `devhive merge <w> <branch>` | ブランチマージ |
| `devhive diff [w]` | 変更差分表示 |
| `devhive note <w> "msg"` | メモ追記 |
| `devhive clean [--all]` | 完了済み削除 |

### 通信（ワーカー↔PM）

| コマンド | 説明 |
|----------|------|
| `devhive request <type> [msg]` | PM にリクエスト（help/review/unblock/clarify） |
| `devhive report "msg"` | PM に進捗報告 |
| `devhive msgs` | 自分宛メッセージ表示 |
| `devhive inbox` | PM受信箱 |
| `devhive reply <w> "msg"` | ワーカーに返信 |
| `devhive broadcast "msg"` | 全員に送信 |

## ロール定義

ロールは以下のいずれかで定義できます：

1. **`.devhive.yaml` で定義**:
```yaml
roles:
  frontend:
    description: "フロントエンド担当"
    file: .devhive/roles/frontend.md
```

2. **`.devhive/roles/<name>.md` にファイルで定義**

3. **インラインで定義** (worker に直接):
```yaml
workers:
  dev:
    role: "開発担当"
```

## agentmineとの機能比較

| 機能 | DevHive | agentmine |
|------|---------|-----------|
| Worktree隔離 | ✅ | ✅ |
| 並列AI実行 | ✅ | ✅ |
| Docker風CLI | ✅ | ✅（実装予定） |
| スコープ制御 | ❌ | ✅ sparse-checkout + chmod |
| Web UI | ❌ | ✅ Next.js |
| チーム協業 | ❌ | ✅ 共有PostgreSQL |
| Memory Bank | ❌ | ✅ |
| MCP統合 | ❌ | ✅ |
| ワーカー通信 | ✅ | ⏳（統合予定） |
| 進捗管理 | ✅ | ⏳（統合予定） |

## gitignore

```
.devhive/
```

## License

MIT
