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
├── .devhive.yaml    # 設定ファイル（git管理）
├── .devhive.db      # 状態DB（gitignore）
├── .worktrees/      # Git Worktrees（gitignore）
│   ├── frontend/
│   └── backend/
└── src/
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

  backend:
    branch: feat/api
    role: "@backend"
    task: API実装
```

## 5. コマンド体系

```
devhive
├── up [worker...]      # ワーカー起動（worktree自動作成）
├── down [worker...]    # ワーカー停止
├── ps                  # ワーカー一覧
├── start <worker>      # 特定ワーカー開始
├── stop <worker>       # 特定ワーカー停止
├── logs [worker]       # ログ表示
├── rm <worker>         # ワーカー削除
├── exec <worker> <cmd> # コマンド実行
├── roles               # ロール一覧
├── config              # 設定表示
├── session <state>     # セッション状態更新（Hooks用）
└── version             # バージョン表示
```

## 6. データモデル

### workers テーブル

| カラム | 型 | 説明 |
|--------|-----|------|
| name | TEXT (PK) | ワーカー名 |
| sprint_id | TEXT | 所属スプリント |
| branch | TEXT | 作業ブランチ |
| role_name | TEXT | ロール名 |
| worktree_path | TEXT | Worktreeパス |
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

## 7. 組み込みロール

| ロール | 説明 |
|--------|------|
| @frontend | フロントエンド |
| @backend | バックエンド |
| @test | テスト・QA |
| @docs | ドキュメント |
| @security | セキュリティ |
| @devops | CI/CD |

## 8. 技術スタック

| 項目 | 選定 |
|------|------|
| 言語 | Go 1.21+ |
| DB | SQLite3 (WAL) |
| CLI | spf13/cobra |
| YAML | gopkg.in/yaml.v3 |
