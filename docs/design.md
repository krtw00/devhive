# DevHive 設計書

## 1. 概要

DevHiveは、Git Worktree + 複数のAIエージェント（Claude Code等）による並列開発の状態を一元管理するためのCLIツール。

## 2. 設計原則

1. **Docker風インターフェース**: `up`, `down`, `ps`, `logs` など直感的なコマンド
2. **1ファイル設定**: `.devhive.yaml` で全て定義
3. **自己完結**: プロジェクト内に全データ配置（グローバル設定不要）
4. **高速起動**: Go製バイナリで即座に応答

## 3. プロジェクト構成

```
myapp/
├── .devhive.yaml        # 設定ファイル（git管理）
└── .devhive/            # DevHiveデータ（gitignore）
    ├── devhive.db       # 状態DB
    ├── worktrees/       # Git Worktrees
    │   ├── frontend/
    │   └── backend/
    ├── roles/           # ロール定義（MD）
    ├── tasks/           # タスク詳細（MD）
    └── workers/         # ワーカー管理情報（MD）
```

## 4. 設定ファイル

```yaml
version: "1"
project: myapp

defaults:
  base_branch: develop

workers:
  frontend:
    branch: feat/ui
    role: "@frontend"
    task: フロントエンド実装
    tool: claude          # AIツール指定（省略時: generic）

  backend:
    branch: feat/api
    role: "@backend"
    task: API実装
    tool: codex
```

### Worker設定フィールド

| フィールド | 必須 | 説明 |
|-----------|------|------|
| branch | Yes | 作業ブランチ名 |
| role | No | ロール名（@builtin or カスタム） |
| task | No | タスク説明（.devhive/tasks/<name>.mdで上書き可） |
| tool | No | AIツール: claude, codex, gemini, generic（デフォルト） |
| worktree | No | Worktreeパスを上書き |
| disabled | No | true でスキップ |

## 5. コマンド体系

```
devhive
├── init [name]           # プロジェクト初期化
├── up [worker...]        # ワーカー起動（worktree自動作成）
├── down [worker...]      # ワーカー停止
├── ps                    # ワーカー一覧
├── status                # 全体サマリー
├── start <worker>        # 特定ワーカー開始
├── stop <worker>         # 特定ワーカー停止
├── logs [worker]         # ログ表示
├── rm <worker>           # ワーカー削除
├── exec <worker> <cmd>   # コマンド実行
├── roles                 # ロール一覧
├── config                # 設定表示
│
├── progress <w> <0-100>  # 進捗更新
├── merge <w> <branch>    # ブランチマージ
├── diff [worker]         # 変更差分表示
├── note <w> "msg"        # メモ追記
├── clean                 # 完了済み削除
│
├── request <type> [msg]  # PM にリクエスト
├── report "msg"          # PM に進捗報告
├── msgs                  # 自分宛メッセージ
├── inbox                 # PM受信箱
├── reply <w> "msg"       # ワーカーに返信
├── broadcast "msg"       # 全員に送信
│
├── session <state>       # セッション状態（Hooks用）
└── version               # バージョン表示
```

## 6. データモデル

### workers テーブル

| カラム | 型 | 説明 |
|--------|-----|------|
| name | TEXT (PK) | ワーカー名 |
| sprint_id | TEXT | 所属スプリント |
| branch | TEXT | 作業ブランチ |
| role_name | TEXT | ロール名（自由形式） |
| worktree_path | TEXT | Worktreeパス |
| tool | TEXT | AIツール (claude/codex/gemini/generic) |
| status | TEXT | pending/working/completed |
| session_state | TEXT | running/idle/stopped |
| progress | INTEGER | 進捗 (0-100) |
| current_task | TEXT | タスク説明 |

### events テーブル

| カラム | 型 | 説明 |
|--------|-----|------|
| id | INTEGER (PK) | イベントID |
| event_type | TEXT | イベント種別 |
| worker | TEXT | 関連ワーカー |
| data | TEXT | JSON詳細データ |
| created_at | TIMESTAMP | 発生日時 |

## 7. ロール定義

ロールは自由形式で、以下の方法で詳細を定義可能：

1. **`.devhive.yaml` で定義**:
```yaml
roles:
  frontend:
    description: "フロントエンド担当"
    file: .devhive/roles/frontend.md
```

2. **`.devhive/roles/<name>.md` にファイルで配置**

3. **DB の roles テーブルに登録**（オプション）

## 8. コンテキストファイル生成

`devhive up` 実行時、各Worktreeにコンテキストファイルを自動生成：

### 生成されるファイル

| ファイル | 条件 | 説明 |
|----------|------|------|
| CONTEXT.md | 常に生成 | 汎用コンテキスト（ロール、タスク、通信方法） |
| CLAUDE.md | tool: claude | Claude Code用指示書 |
| AGENTS.md | tool: codex | Codex用指示書 |
| GEMINI.md | tool: gemini | Gemini用指示書 |

### CONTEXT.md の内容

```markdown
# DevHive Worker Context

## Worker
- **Name**: frontend
- **Branch**: feat/ui
- **Role**: @frontend
- **Tool**: claude

## Project
- **Name**: myapp
- **Base Branch**: develop

## Role
フロントエンド開発担当...

## Task
フロントエンド実装...

## Communication
PMとの通信コマンド...
```

### ツール別ファイル

AIツールが自動的に読み込むファイル名を使用：
- **Claude Code**: `CLAUDE.md` を自動読み込み
- **Codex**: `AGENTS.md` を自動読み込み
- **Gemini**: `GEMINI.md` を自動読み込み

## 9. 技術スタック

| 項目 | 選定 |
|------|------|
| 言語 | Go 1.21+ |
| DB | SQLite3 (WAL) |
| CLI | spf13/cobra |
| YAML | gopkg.in/yaml.v3 |
