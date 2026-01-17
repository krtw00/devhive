# DevHive 基本設計書

## 1. 概要

### 1.1 目的

DevHiveは、tmux + Git Worktree + 複数のAIエージェント（Claude Code等）による並列開発を効率的に管理するためのCLIツールである。

### 1.2 解決する課題

| 課題 | 従来の方法 | DevHiveでの解決 |
|------|-----------|----------------|
| 状態管理の分散 | alerts.log, shared-board.md等の複数ファイル | SQLite単一DB |
| ワーカー間連携 | ファイル書き込み→手動確認 | メッセージシステム |
| レビュー管理 | テキストファイル、手動追跡 | 構造化されたレビューフロー |
| ファイル競合 | 事後のマージ競合 | 事前ロック機構 |
| 監査・デバッグ | ログファイルのgrep | イベントログのクエリ |

### 1.3 設計原則

1. **Single Source of Truth**: 全状態をSQLiteで一元管理
2. **高速起動**: Go製バイナリで即座に応答
3. **シンプルなCLI**: 直感的なコマンド体系
4. **プロジェクト非依存**: 任意のプロジェクトで使用可能

## 2. アーキテクチャ

### 2.1 システム構成

```
┌─────────────────────────────────────────────────────────────┐
│                        DevHive CLI                          │
│                     (devhive コマンド)                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐       │
│  │ Sprint  │  │ Worker  │  │ Review  │  │ Message │  ...  │
│  │ Cmds    │  │ Cmds    │  │ Cmds    │  │ Cmds    │       │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘       │
│       │            │            │            │             │
│       └────────────┴────────────┴────────────┘             │
│                          │                                  │
│                    ┌─────┴─────┐                           │
│                    │  DB Layer │                           │
│                    │ (internal/db)                         │
│                    └─────┬─────┘                           │
│                          │                                  │
└──────────────────────────┼──────────────────────────────────┘
                           │
                    ┌──────┴──────┐
                    │   SQLite    │
                    │ ~/.devhive/ │
                    │  state.db   │
                    └─────────────┘
```

### 2.2 並列開発環境での配置

```
┌─────────────────────────────────────────────────────────────────┐
│ tmux session                                                     │
├─────────────┬─────────────┬─────────────┬─────────────┬────────┤
│   Pane 0    │   Pane 1    │   Pane 2    │   Pane 3    │ Pane 7 │
│   Monitor   │  Worker 1   │  Worker 2   │  Worker 3   │ Senior │
│  (script)   │(Claude Code)│(Claude Code)│(Claude Code)│Engineer│
├─────────────┴─────────────┴─────────────┴─────────────┴────────┤
│                                                                  │
│  各ペインから devhive コマンドを実行                             │
│  → SQLiteに状態が記録される                                     │
│  → 他のペインから状態を参照可能                                  │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
                              │
                    ┌─────────┴─────────┐
                    │  ~/.devhive/      │
                    │   state.db        │
                    │  (共有データベース) │
                    └───────────────────┘
```

## 3. データモデル

### 3.1 ER図

```
┌──────────────┐       ┌──────────────┐
│   sprints    │       │   workers    │
├──────────────┤       ├──────────────┤
│ id (PK)      │←──────│ sprint_id(FK)│
│ config_file  │       │ name (PK)    │
│ project_path │       │ pane_id      │
│ status       │       │ branch       │
│ started_at   │       │ issue        │
│ completed_at │       │ worktree_path│
└──────────────┘       │ status       │
                       │ current_task │
                       │ last_commit  │
                       │ error_count  │
                       │ last_error   │
                       │ updated_at   │
                       └──────┬───────┘
                              │
          ┌───────────────────┼───────────────────┐
          │                   │                   │
          ▼                   ▼                   ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   reviews    │    │   messages   │    │  file_locks  │
├──────────────┤    ├──────────────┤    ├──────────────┤
│ id (PK)      │    │ id (PK)      │    │ file_path(PK)│
│ worker (FK)  │    │ from_worker  │    │ locked_by(FK)│
│ commit_hash  │    │ to_worker    │    │ reason       │
│ description  │    │ message_type │    │ locked_at    │
│ status       │    │ subject      │    └──────────────┘
│ reviewer     │    │ content      │
│ comment      │    │ read_at      │
│ created_at   │    │ created_at   │
│ resolved_at  │    └──────────────┘
└──────────────┘

┌──────────────┐
│   events     │
├──────────────┤
│ id (PK)      │
│ event_type   │
│ worker       │
│ data (JSON)  │
│ created_at   │
└──────────────┘
```

### 3.2 テーブル詳細

#### sprints（スプリント）

| カラム | 型 | 説明 |
|--------|-----|------|
| id | TEXT | スプリントID（例: sprint-04） |
| config_file | TEXT | 設定ファイルパス |
| project_path | TEXT | プロジェクトパス |
| status | TEXT | active / completed / aborted |
| started_at | TIMESTAMP | 開始日時 |
| completed_at | TIMESTAMP | 完了日時 |

#### workers（ワーカー）

| カラム | 型 | 説明 |
|--------|-----|------|
| name | TEXT | ワーカー名（例: security） |
| sprint_id | TEXT | 所属スプリント |
| pane_id | INTEGER | tmuxペイン番号 |
| branch | TEXT | 作業ブランチ |
| issue | TEXT | 対応Issue番号 |
| worktree_path | TEXT | Worktreeパス |
| status | TEXT | pending/working/review_pending/completed/blocked/error |
| current_task | TEXT | 現在のタスク説明 |
| last_commit | TEXT | 最新コミットハッシュ |
| error_count | INTEGER | エラー回数 |
| last_error | TEXT | 最後のエラー内容 |
| updated_at | TIMESTAMP | 更新日時 |

#### reviews（レビュー）

| カラム | 型 | 説明 |
|--------|-----|------|
| id | INTEGER | レビューID |
| worker | TEXT | 依頼元ワーカー |
| commit_hash | TEXT | レビュー対象コミット |
| description | TEXT | 変更内容の説明 |
| status | TEXT | pending/ok/needs_fix/wont_fix |
| reviewer | TEXT | レビュアー名 |
| comment | TEXT | レビューコメント |
| created_at | TIMESTAMP | 依頼日時 |
| resolved_at | TIMESTAMP | 解決日時 |

#### messages（メッセージ）

| カラム | 型 | 説明 |
|--------|-----|------|
| id | INTEGER | メッセージID |
| from_worker | TEXT | 送信元 |
| to_worker | TEXT | 送信先（NULLはブロードキャスト） |
| message_type | TEXT | info/warning/conflict/question/answer/system |
| subject | TEXT | 件名 |
| content | TEXT | 本文 |
| read_at | TIMESTAMP | 既読日時 |
| created_at | TIMESTAMP | 送信日時 |

#### file_locks（ファイルロック）

| カラム | 型 | 説明 |
|--------|-----|------|
| file_path | TEXT | ロック対象ファイルパス |
| locked_by | TEXT | ロック取得ワーカー |
| reason | TEXT | ロック理由 |
| locked_at | TIMESTAMP | ロック取得日時 |

#### events（イベントログ）

| カラム | 型 | 説明 |
|--------|-----|------|
| id | INTEGER | イベントID |
| event_type | TEXT | イベント種別 |
| worker | TEXT | 関連ワーカー |
| data | TEXT | JSON形式の詳細データ |
| created_at | TIMESTAMP | 発生日時 |

## 4. コマンド体系

### 4.1 コマンド一覧

```
devhive
├── init <sprint-id>           # スプリント初期化
├── status                     # 全体状態表示
├── worker
│   ├── register <name> <branch>  # ワーカー登録
│   ├── start <name>              # 作業開始
│   ├── complete <name>           # 作業完了
│   └── status <name> <status>    # 状態更新
├── review
│   ├── request <commit>          # レビュー依頼
│   ├── list                      # 未処理一覧
│   ├── ok <id> [comment]         # 承認
│   └── fix <id> <comment>        # 要修正
├── msg
│   ├── send <to> <message>       # メッセージ送信
│   ├── broadcast <message>       # 全員に送信
│   ├── unread [worker]           # 未読確認
│   └── read <id|all>             # 既読化
├── lock
│   ├── acquire <file>            # ロック取得
│   ├── release <file>            # ロック解放
│   └── list                      # ロック一覧
├── events                     # イベントログ
└── version                    # バージョン表示
```

### 4.2 典型的なワークフロー

#### PM（プロジェクトマネージャー）

```bash
# 1. スプリント開始
devhive init sprint-05 --config ./sprint-05.conf

# 2. ワーカー登録
devhive worker register security fix/security-auth -i "#313" -p 1
devhive worker register quality fix/quality-check -i "#314" -p 2

# 3. 状態監視
devhive status

# 4. 問題発生時の確認
devhive events --limit 20
devhive msg unread
```

#### ワーカー

```bash
# 1. 作業開始
devhive worker start security

# 2. ファイルロック（必要な場合）
devhive lock acquire src/auth.py -w security

# 3. 他ワーカーへの連絡
devhive msg send quality "認証APIを変更します" -f security

# 4. 未読確認
devhive msg unread security

# 5. コミット後、レビュー依頼
devhive review request abc1234 -w security -d "認証機能の追加"

# 6. ロック解除
devhive lock release src/auth.py -w security

# 7. 作業完了
devhive worker complete security
```

#### シニアエンジニア（レビュアー）

```bash
# 1. レビュー待ち確認
devhive review list

# 2. レビュー実施後
devhive review ok 1 "問題なし"
# または
devhive review fix 1 "エラーハンドリングを追加してください"
```

## 5. イベント種別

全ての操作はeventsテーブルに記録される。

| イベント種別 | 説明 | dataの例 |
|-------------|------|----------|
| sprint_created | スプリント作成 | {"sprint_id": "sprint-05"} |
| sprint_completed | スプリント完了 | {"sprint_id": "sprint-05"} |
| worker_registered | ワーカー登録 | {"branch": "fix/xxx", "issue": "#123"} |
| worker_status_changed | ワーカー状態変更 | {"status": "working"} |
| review_requested | レビュー依頼 | {"commit": "abc1234"} |
| review_resolved | レビュー完了 | {"review_id": 1, "status": "ok"} |
| message_sent | メッセージ送信 | {"to": "quality", "type": "info"} |
| file_locked | ファイルロック | {"file": "src/auth.py"} |
| file_unlocked | ファイルアンロック | {"file": "src/auth.py"} |

## 6. 拡張ポイント

### 6.1 将来的な機能追加候補

1. **watch モード**: リアルタイムで状態変化を監視
2. **Web UI**: ブラウザから状態を確認
3. **Slack/Discord連携**: 通知の外部送信
4. **統計機能**: スプリントの振り返りデータ
5. **設定ファイル読み込み**: sprint.confの自動パース

### 6.2 他プロジェクトへの統合

DevHiveは特定プロジェクトに依存しない。統合方法:

1. **スクリプトから呼び出し**: `parallel-dev-*.sh` から `devhive` コマンドを呼び出し
2. **エージェント定義に記載**: ワーカーエージェントにdevhiveの使い方を記載
3. **CLAUDE.md参照**: プロジェクトのCLAUDE.mdからdevhiveドキュメントを参照

## 7. ファイル構成

```
devhive/
├── cmd/
│   └── devhive/
│       └── main.go          # CLIエントリーポイント
├── internal/
│   └── db/
│       ├── db.go            # データベース操作
│       └── schema.sql       # SQLiteスキーマ
├── docs/
│   ├── design.md            # 本ドキュメント
│   └── commands.md          # コマンドリファレンス
├── go.mod
├── go.sum
├── README.md
└── .gitignore
```

## 8. 技術スタック

| 項目 | 選定 | 理由 |
|------|------|------|
| 言語 | Go 1.21+ | 高速起動、単一バイナリ、クロスプラットフォーム |
| DB | SQLite3 | 組み込み、ファイルベース、トランザクション |
| CLI | spf13/cobra | Goの標準的CLIフレームワーク |
| SQLiteドライバ | mattn/go-sqlite3 | 成熟した実装、CGO依存 |
