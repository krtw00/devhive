# DevHive 基本設計書

## 1. 概要

### 1.1 目的

DevHiveは、Git Worktree + 複数のAIエージェント（Claude Code等）による並列開発の状態を一元管理するためのCLIツールである。

### 1.2 解決する課題

| 課題 | 従来の方法 | DevHiveでの解決 |
|------|-----------|----------------|
| 状態管理の分散 | alerts.log, shared-board.md等の複数ファイル | SQLite単一DB |
| ワーカー間連携 | ファイル書き込み→手動確認 | メッセージシステム |
| 監査・デバッグ | ログファイルのgrep | イベントログのクエリ |

### 1.3 設計原則

1. **Single Source of Truth**: 全状態をSQLiteで一元管理
2. **高速起動**: Go製バイナリで即座に応答
3. **シンプルなCLI**: 直感的なコマンド体系
4. **環境非依存**: tmux/screen/複数ターミナル等、どの環境でも動作
5. **最小限の機能**: 状態管理に専念、プロセス管理は外部に委譲

## 2. アーキテクチャ

### 2.1 ハイブリッドアーキテクチャ

DevHiveはコア機能（状態管理）に専念し、実行環境との連携はオプションとする。

```
┌─────────────────────────────────────────────────────────┐
│                    DevHive Core CLI                      │
│              (状態管理に専念・環境非依存)                 │
├─────────────────────────────────────────────────────────┤
│  Sprint │ Worker │ Message │ Events │ Watch             │
└────────────────────────┬────────────────────────────────┘
                         │
                    ┌────┴────┐
                    │ SQLite  │
                    │~/.devhive/state.db
                    └────┬────┘
                         │
     ┌───────────────────┼───────────────────┐
     │                   │                   │
┌────┴────┐        ┌─────┴─────┐       ┌─────┴─────┐
│  tmux   │        │ 複数      │       │ VS Code   │
│ 連携    │        │ターミナル │       │ Tasks     │
│スクリプト│        │ (手動)    │       │ (将来)    │
└─────────┘        └───────────┘       └───────────┘
```

### 2.2 並列開発環境での配置

```
┌─────────────────────────────────────────────────────────────────┐
│ 任意の実行環境（tmux / 複数ターミナル / etc）                     │
├─────────────────┬─────────────────┬─────────────────┬───────────┤
│    Worker 1     │    Worker 2     │    Worker 3     │  Senior   │
│  (Claude Code)  │  (Claude Code)  │  (Claude Code)  │ Engineer  │
│                 │                 │                 │           │
│ DEVHIVE_WORKER  │ DEVHIVE_WORKER  │ DEVHIVE_WORKER  │           │
│ =security       │ =quality        │ =mobile         │           │
├─────────────────┴─────────────────┴─────────────────┴───────────┤
│                                                                  │
│  各環境から devhive コマンドを実行                               │
│  → SQLiteに状態が記録される                                     │
│  → 他の環境から状態を参照可能                                    │
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

### 3.1 正規化スキーマ

データベースは正規化されており、マスタテーブルとコアテーブルに分かれている。

### 3.2 ER図

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│ message_types   │     │     roles       │     │  event_types    │
│    (マスタ)     │     │    (マスタ)     │     │    (マスタ)     │
├─────────────────┤     ├─────────────────┤     ├─────────────────┤
│ name (PK)       │     │ name (PK)       │     │ name (PK)       │
│ description     │     │ description     │     │ description     │
└────────┬────────┘     │ role_file       │     └────────┬────────┘
         │              │ created_at      │              │
         │ FK           └────────┬────────┘              │ FK
         ▼                       │ FK                    ▼
┌─────────────────┐     ┌────────┴────────┐     ┌─────────────────┐
│    messages     │     │    workers      │     │     events      │
├─────────────────┤     ├─────────────────┤     ├─────────────────┤
│ id (PK)         │     │ name (PK)       │     │ id (PK)         │
│ from_worker     │     │ sprint_id (FK)  │     │ event_type (FK) │
│ to_worker (FK)──│────→│ branch          │     │ worker          │
│ message_type(FK)│     │ role_name (FK)  │     │ data (JSON)     │
│ subject         │     │ worktree_path   │     │ created_at      │
│ content         │     │ status          │     └─────────────────┘
│ read_at         │     │ current_task    │
│ created_at      │     │ last_commit     │
└─────────────────┘     │ error_count     │
                        │ last_error      │
┌─────────────────┐     │ updated_at      │
│    sprints      │     └────────┬────────┘
├─────────────────┤              │
│ id (PK)         │←─────────────┘ FK
│ config_file     │
│ project_path    │
│ status          │
│ started_at      │
│ completed_at    │
└─────────────────┘
```

### 3.3 テーブル詳細

#### マスタテーブル

##### roles（ロール）

| カラム | 型 | 説明 |
|--------|-----|------|
| name | TEXT (PK) | ロール名（例: security） |
| description | TEXT | ロールの説明 |
| role_file | TEXT | ロール定義ファイルのパス |
| created_at | TIMESTAMP | 作成日時 |

##### message_types（メッセージ種別）

| カラム | 型 | 説明 |
|--------|-----|------|
| name | TEXT (PK) | info/warning/question/answer/system |
| description | TEXT | 種別の説明 |

##### event_types（イベント種別）

| カラム | 型 | 説明 |
|--------|-----|------|
| name | TEXT (PK) | イベント種別名 |
| description | TEXT | 種別の説明 |

#### コアテーブル

##### sprints（スプリント）

| カラム | 型 | 説明 |
|--------|-----|------|
| id | TEXT (PK) | スプリントID（例: sprint-04） |
| config_file | TEXT | 設定ファイルパス |
| project_path | TEXT | プロジェクトパス |
| status | TEXT | active / completed / aborted |
| started_at | TIMESTAMP | 開始日時 |
| completed_at | TIMESTAMP | 完了日時 |

##### workers（ワーカー）

| カラム | 型 | 説明 | FK |
|--------|-----|------|-----|
| name | TEXT (PK) | ワーカー名 | - |
| sprint_id | TEXT | 所属スプリント | sprints.id (CASCADE) |
| branch | TEXT | 作業ブランチ | - |
| role_name | TEXT | ロール名 | roles.name (SET NULL) |
| worktree_path | TEXT | Worktreeパス | - |
| status | TEXT | pending/working/completed/blocked/error | - |
| current_task | TEXT | 現在のタスク説明 | - |
| last_commit | TEXT | 最新コミットハッシュ | - |
| error_count | INTEGER | エラー回数 | - |
| last_error | TEXT | 最後のエラー内容 | - |
| updated_at | TIMESTAMP | 更新日時 | - |

##### messages（メッセージ）

| カラム | 型 | 説明 | FK |
|--------|-----|------|-----|
| id | INTEGER (PK) | メッセージID | - |
| from_worker | TEXT | 送信元 | - |
| to_worker | TEXT | 送信先 | workers.name (CASCADE) |
| message_type | TEXT | メッセージ種別 | message_types.name (RESTRICT) |
| subject | TEXT | 件名 | - |
| content | TEXT | 本文 | - |
| read_at | TIMESTAMP | 既読日時 | - |
| created_at | TIMESTAMP | 送信日時 | - |

##### events（イベントログ）

| カラム | 型 | 説明 | FK |
|--------|-----|------|-----|
| id | INTEGER (PK) | イベントID | - |
| event_type | TEXT | イベント種別 | event_types.name (RESTRICT) |
| worker | TEXT | 関連ワーカー |
| data | TEXT | JSON形式の詳細データ |
| created_at | TIMESTAMP | 発生日時 |

## 4. コマンド体系

### 4.1 コマンド一覧

```
devhive
├── init <sprint-id>              # スプリント初期化
├── status                        # 全体状態表示
├── sprint
│   └── complete                  # スプリント完了
├── worker
│   ├── register <name> <branch>  # ワーカー登録
│   ├── start [name]              # 作業開始
│   ├── complete [name]           # 作業完了
│   ├── status [name] <status>    # 状態変更
│   ├── show [name]               # 詳細表示
│   ├── task <task>               # タスク更新
│   └── error <message>           # エラー報告
├── msg
│   ├── send <to> <message>       # メッセージ送信
│   ├── broadcast <message>       # 全員に送信
│   ├── unread                    # 未読確認
│   └── read <id|all>             # 既読化
├── events                        # イベントログ
├── watch                         # 状態監視
└── version                       # バージョン表示
```

### 4.2 環境変数

| 変数名 | 説明 | 例 |
|--------|------|-----|
| DEVHIVE_PROJECT | プロジェクト名（DB分離用） | duel-log-app |
| DEVHIVE_WORKER | デフォルトのワーカー名 | security |

#### プロジェクト指定

プロジェクトを指定すると、DBが `~/.devhive/projects/<project>/state.db` に分離される:

```bash
# 方法1: 環境変数
export DEVHIVE_PROJECT=duel-log-app
devhive init sprint-01

# 方法2: --project フラグ（-P）
devhive --project duel-log-app init sprint-01
devhive -P duel-log-app status
```

#### ワーカー名省略

環境変数を設定すると、コマンドでワーカー名を省略できる:

```bash
export DEVHIVE_WORKER=security

# 以下は同等
devhive worker start
devhive worker start security
```

### 4.3 典型的なワークフロー

#### PM（プロジェクトマネージャー）

```bash
# 1. スプリント開始
devhive init sprint-05

# 2. ワーカー登録
devhive worker register security fix/security-auth --issue "#313"
devhive worker register quality fix/quality-check --issue "#314"

# 3. 状態監視
devhive status
devhive watch

# 4. 問題発生時の確認
devhive events --limit 20
devhive msg unread
```

#### ワーカー

```bash
# 環境変数設定（セッション開始時）
export DEVHIVE_WORKER=security

# 1. 作業開始
devhive worker start --task "認証APIの実装"

# 2. タスク更新
devhive worker task "トークン検証の実装中"

# 3. 他ワーカーへの連絡
devhive msg send quality "認証APIを変更しました"

# 4. 未読確認
devhive msg unread

# 5. 状態監視（バックグラウンド）
devhive watch --filter=message

# 6. 作業完了
devhive worker complete
```

## 5. イベント種別

全ての操作はeventsテーブルに記録される。

| イベント種別 | 説明 | dataの例 |
|-------------|------|----------|
| sprint_created | スプリント作成 | {"sprint_id": "sprint-05"} |
| sprint_completed | スプリント完了 | {"sprint_id": "sprint-05"} |
| worker_registered | ワーカー登録 | {"branch": "fix/xxx", "issue": "#123"} |
| worker_status_changed | ワーカー状態変更 | {"status": "working"} |
| worker_task_updated | タスク更新 | {"task": "認証API実装中"} |
| worker_error | エラー報告 | {"message": "ビルド失敗"} |
| message_sent | メッセージ送信 | {"to": "quality", "type": "info"} |

## 6. watchコマンド仕様

### 6.1 動作

eventsテーブルをポーリングし、前回以降の新規イベントを出力する。

### 6.2 フィルタオプション

```bash
devhive watch                    # 全変化を監視
devhive watch --filter=message   # メッセージのみ
devhive watch --filter=worker    # ワーカー状態変化のみ
```

### 6.3 出力例

```
[12:34:56] message: quality → you: "DuelTable.vue編集します"
[12:35:10] worker: quality → completed
[12:36:00] message: (broadcast) pm: "15分後にマージします"
```

## 7. 拡張ポイント

### 7.1 将来的な機能追加候補

1. **Web UI**: ブラウザから状態を確認
2. **Slack/Discord連携**: 通知の外部送信
3. **統計機能**: スプリントの振り返りデータ
4. **設定ファイル読み込み**: sprint.confの自動パース
5. **VS Code拡張**: エディタ統合

### 7.2 連携スクリプト

DevHiveは状態管理に専念し、実行環境との連携はオプションのスクリプトで提供:

```
scripts/
├── devhive-tmux.sh      # tmuxセッション起動・管理
├── devhive-worktree.sh  # Git Worktree作成補助
└── examples/
    └── sprint.conf.example
```

## 8. ファイル構成

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
├── scripts/                 # 連携スクリプト（オプション）
├── templates/               # CLAUDE.mdテンプレート
├── go.mod
├── go.sum
├── README.md
└── .gitignore
```

### 8.2 運用時のディレクトリ構成

プロジェクト固有の設定はホームディレクトリに配置:

```
~/.devhive/
├── state.db                 # グローバルDB（プロジェクト未指定時）
└── projects/
    └── <project-name>/
        ├── state.db         # プロジェクト専用DB
        ├── roles/           # ロール定義ファイル
        │   ├── frontend.md
        │   └── backend.md
        ├── templates/       # CLAUDE.mdテンプレート
        │   └── CLAUDE.md
        ├── sprints/         # スプリント設定
        │   └── sprint-01.conf
        └── worktrees/       # Git Worktree配置場所
```

### 8.3 推奨セットアップ手順

```bash
# 1. devhiveをインストール
go build -o devhive ./cmd/devhive
ln -s $(pwd)/devhive ~/bin/devhive
export PATH="$HOME/bin:$PATH"

# 2. プロジェクト設定ディレクトリを作成
mkdir -p ~/.devhive/projects/my-project/{roles,templates,sprints,worktrees}

# 3. 環境変数を設定（.bashrcなどに追加）
export DEVHIVE_PROJECT=my-project

# 4. スプリント開始
devhive init sprint-01
```

## 9. 技術スタック

| 項目 | 選定 | 理由 |
|------|------|------|
| 言語 | Go 1.21+ | 高速起動、単一バイナリ、クロスプラットフォーム |
| DB | SQLite3 | 組み込み、ファイルベース、トランザクション |
| CLI | spf13/cobra | Goの標準的CLIフレームワーク |
| SQLiteドライバ | mattn/go-sqlite3 | 成熟した実装、CGO依存 |
