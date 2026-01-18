# DevHive 仕様書

## 概要

DevHiveは、複数のAIワーカー（Claude Code等）による並列開発を調整するためのCLIツール。SQLiteベースの状態管理により、ワーカー間の協調作業を実現する。

---

## アーキテクチャ

### データストレージ

- **場所**: `~/.devhive/projects/<project-name>/devhive.db`
- **形式**: SQLite（WALモード有効）
- **分離**: プロジェクトごとに独立したDB

### プロジェクト検出優先順位

1. `--project` / `-P` フラグ（最優先）
2. `.devhive` ファイル（cwdから上位へ検索）
3. パス検出（`~/.devhive/projects/<name>/...` 配下）
4. `DEVHIVE_PROJECT` 環境変数（最低優先度）

---

## データモデル

### sprints テーブル

| カラム | 型 | 説明 |
|--------|-----|------|
| id | TEXT | スプリントID（PK） |
| status | TEXT | active / completed |
| started_at | DATETIME | 開始日時 |
| completed_at | DATETIME | 完了日時（NULL可） |

### roles テーブル

| カラム | 型 | 説明 |
|--------|-----|------|
| name | TEXT | ロール名（PK） |
| description | TEXT | 説明 |
| role_file | TEXT | ロールファイルパス |
| args | TEXT | 追加引数 |
| created_at | DATETIME | 作成日時 |

### workers テーブル

| カラム | 型 | 説明 |
|--------|-----|------|
| name | TEXT | ワーカー名（PK） |
| sprint_id | TEXT | スプリントID（FK） |
| role_name | TEXT | ロール名（FK、NULL可） |
| branch | TEXT | ブランチ名 |
| worktree_path | TEXT | Worktreeパス |
| status | TEXT | pending/working/completed/blocked/error |
| session_state | TEXT | running/waiting_permission/idle/stopped |
| current_task | TEXT | 現在のタスク |
| last_commit | TEXT | 最後のコミットハッシュ |
| error_count | INTEGER | エラー回数 |
| last_error | TEXT | 最後のエラーメッセージ |
| created_at | DATETIME | 作成日時 |
| updated_at | DATETIME | 更新日時 |

### messages テーブル

| カラム | 型 | 説明 |
|--------|-----|------|
| id | INTEGER | メッセージID（PK） |
| from_worker | TEXT | 送信元（NULL=PM） |
| to_worker | TEXT | 送信先（NULL=broadcast） |
| message | TEXT | メッセージ内容 |
| msg_type | TEXT | info/warning/question/answer/system |
| subject | TEXT | 件名 |
| read | BOOLEAN | 既読フラグ |
| created_at | DATETIME | 作成日時 |

### events テーブル

| カラム | 型 | 説明 |
|--------|-----|------|
| id | INTEGER | イベントID（PK） |
| event_type | TEXT | イベント種別 |
| worker_name | TEXT | 関連ワーカー（NULL可） |
| data | TEXT | JSONデータ |
| created_at | DATETIME | 作成日時 |

---

## 環境変数

| 変数 | 説明 | 用途 |
|------|------|------|
| DEVHIVE_PROJECT | プロジェクト名 | DBファイルの選択（最低優先度） |
| DEVHIVE_WORKER | ワーカー名 | コマンドの対象ワーカー省略時に使用 |

### DEVHIVE_WORKER の使用箇所

| コマンド | 効果 |
|----------|------|
| `worker start [name]` | 引数省略時に使用 |
| `worker complete [name]` | 引数省略時に使用 |
| `worker status [name] <status>` | 引数省略時に使用 |
| `worker show [name]` | 引数省略時に使用 |
| `worker task <task>` | 対象ワーカー |
| `worker error <msg>` | 対象ワーカー |
| `worker session <state>` | 対象ワーカー |
| `msg send/broadcast` | 送信元ワーカー |
| `msg unread/read` | 受信者ワーカー |

---

## CLI コマンド

### グローバルオプション

```
-h, --help              ヘルプを表示
-P, --project <name>    プロジェクト名を指定
--json                  JSON形式で出力（対応コマンドのみ）
```

### コマンド一覧

| コマンド | 説明 |
|----------|------|
| `init <sprint-id>` | スプリント初期化 |
| `status` | 現在のステータス表示 |
| `projects` | 全プロジェクト一覧 |
| `sprint complete` | スプリント完了 |
| `sprint setup <file>` | 一括ワーカー登録 |
| `sprint report` | レポート生成 |
| `role create/list/show/update/delete` | ロール管理 |
| `worker register/start/complete/status/show/task/error/session` | ワーカー管理 |
| `msg send/broadcast/unread/read` | メッセージ管理 |
| `events` | イベント一覧 |
| `watch` | リアルタイム監視 |
| `cleanup events/messages/worktrees/all` | クリーンアップ |

---

## ワーカーステータス

| ステータス | アイコン | 説明 |
|-----------|----------|------|
| pending | ⏳ | 待機中 |
| working | 🔨 | 作業中 |
| completed | ✅ | 完了 |
| blocked | 🚫 | ブロック中 |
| error | ❌ | エラー |

## セッション状態

| 状態 | アイコン | 説明 |
|------|----------|------|
| running | ▶ | アクティブに実行中 |
| waiting_permission | ⏸ | ユーザー入力待ち |
| idle | ○ | 待機中 |
| stopped | ■ | セッション終了 |

---

## ファイル形式

### .devhive ファイル

プロジェクトルートに配置。内容はプロジェクト名のみ。

```
duel-log-app
```

### スプリント設定（YAML形式）

```yaml
# Sprint XX: 説明
# Issues: #xx, #yy

workers:
  - name: worker-name      # 一意のワーカー名
    branch: branch-name    # 作業ブランチ
    role: role-name        # ロール名
    task: |                # タスク詳細（マルチライン）
      タスクの説明...
```

### ロールファイル（Markdown）

```markdown
# ロール名

## 基本ルール
- ルール1
- ルール2

## 技術スタック
- 技術1
- 技術2

## コマンド
- コマンド1
- コマンド2
```

---

## Claude Code Hooks

### 設定ファイル

`~/.claude/settings.json`

### フック種類

| フック | タイミング | 設定する状態 |
|--------|-----------|-------------|
| PreToolUse | ツール実行前 | waiting_permission |
| PostToolUse | ツール実行後 | running |
| Stop | セッション終了 | idle |

### matcher パターン

- `Bash|Edit|Write|NotebookEdit` - 主要な変更系ツール

---

## Worktree管理

### 自動作成

```bash
devhive worker register <name> <branch> --create-worktree
```

**作成先**: `~/.devhive/projects/<project>/worktrees/<worker-name>`

### direnv連携

Worktree作成時に`.envrc`を自動生成:

```bash
export DEVHIVE_WORKER=<worker-name>
```

---

## イベント種別

| イベント | 説明 |
|----------|------|
| sprint_started | スプリント開始 |
| sprint_completed | スプリント完了 |
| worker_registered | ワーカー登録 |
| worker_status_changed | ステータス変更 |
| worker_session_changed | セッション状態変更 |
| message_sent | メッセージ送信 |
| error_reported | エラー報告 |

---

## スキーマ移行

既存DBへの後方互換性を維持するため、起動時に自動移行を実行:

```go
func (db *DB) migrate() error {
    // 存在しないカラムを追加
    if !db.columnExists("workers", "session_state") {
        db.conn.Exec(`ALTER TABLE workers ADD COLUMN session_state TEXT...`)
    }
    // ...
}
```
