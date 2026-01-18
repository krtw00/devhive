# Compose コマンド（Docker風インターフェース）

DevHiveはDocker Composeに似た操作感でワーカーを管理できます。

## プロジェクト構成

```
myapp/
├── .devhive.yaml    # 設定ファイル（git管理）
├── .devhive.db      # 状態DB（自動生成、gitignore）
└── .worktrees/      # Git Worktrees（自動生成、gitignore）
```

## 設定ファイル

プロジェクトルートに `.devhive.yaml` を配置します：

```yaml
version: "1"
project: my-project  # 省略時はディレクトリ名

defaults:
  base_branch: develop

workers:
  fe-auth:
    branch: feat/auth-ui
    role: "@frontend"          # 組み込みロール
    task: |
      認証UIの実装
      - ログインフォーム
      - 登録フォーム

  be-api:
    branch: feat/auth-api
    role: "@backend"
    task: 認証APIの実装

  docs:
    branch: docs/api
    role: "@docs"
    task: API仕様書の作成
    disabled: true             # 一時的に無効化
```

## コマンド一覧

| コマンド | 説明 | Docker相当 |
|----------|------|-----------|
| `devhive up` | ワーカーを起動 | `docker compose up` |
| `devhive down` | ワーカーを停止 | `docker compose down` |
| `devhive ps` | ワーカー一覧 | `docker compose ps` |
| `devhive start` | 特定ワーカーを起動 | `docker start` |
| `devhive stop` | 特定ワーカーを停止 | `docker stop` |
| `devhive logs` | イベントログ表示 | `docker logs` |
| `devhive rm` | ワーカーを削除 | `docker rm` |
| `devhive exec` | ワーカーのworktreeでコマンド実行 | `docker exec` |
| `devhive roles` | 利用可能なロール一覧 | `docker images` |
| `devhive config` | 設定内容を表示 | `docker compose config` |

---

## devhive up

`.devhive.yaml` からワーカーを起動（登録）します。

```bash
# 全ワーカーを起動（worktree自動作成）
devhive up

# 特定のワーカーのみ起動
devhive up fe-auth be-api

# worktreeを作成しない
devhive up --no-worktree
```

### オプション

| オプション | 説明 |
|-----------|------|
| `--no-worktree` | Git worktreeを作成しない |
| `--file <path>`, `-f` | 設定ファイルを指定 |
| `--dry-run` | 実行内容を表示（実行しない） |

### 処理内容

1. スプリントが存在しない場合は作成
2. 設定ファイルで定義されたロールをDBに登録
3. 各ワーカーを登録（`worker register` 相当）
4. `--worktree` 指定時はGit worktreeを作成

---

## devhive down

ワーカーを完了状態にします。

```bash
# 全ワーカーを完了
devhive down

# 特定のワーカーのみ
devhive down fe-auth
```

---

## devhive ps

ワーカーの状態を一覧表示します。

```bash
# 実行中のワーカーのみ
devhive ps

# 全てのワーカー（完了済み含む）
devhive ps -a

# 名前のみ表示
devhive ps -q
```

### オプション

| オプション | 短縮形 | 説明 |
|-----------|-------|------|
| `--all` | `-a` | 完了済みワーカーも表示 |
| `--quiet` | `-q` | 名前のみ表示 |

### 出力例

```
WORKER    BRANCH           ROLE       STATUS    PROGRESS
fe-auth   feat/auth-ui     frontend   working   [███░░] 60%
be-api    feat/auth-api    backend    working   [████░] 80%
docs      docs/api         @docs      idle      [░░░░░] 0%
```

---

## devhive start

停止中のワーカーを開始状態にします。

```bash
devhive start fe-auth
```

---

## devhive stop

ワーカーを一時停止状態にします。

```bash
devhive stop fe-auth
```

---

## devhive logs

イベントログを表示します。

```bash
# 最新のログを表示
devhive logs

# 特定ワーカーのログ
devhive logs fe-auth

# リアルタイムで追跡
devhive logs -f

# 表示件数を指定
devhive logs -n 50
```

### オプション

| オプション | 短縮形 | 説明 |
|-----------|-------|------|
| `--follow` | `-f` | リアルタイム追跡 |
| `--tail <n>` | `-n` | 表示件数（デフォルト: 20） |

---

## devhive rm

ワーカーをデータベースから削除します。

```bash
# 完了済みワーカーを削除
devhive rm fe-auth

# 強制削除（実行中でも）
devhive rm -f fe-auth
```

### オプション

| オプション | 短縮形 | 説明 |
|-----------|-------|------|
| `--force` | `-f` | 強制削除 |

---

## devhive exec

ワーカーのworktreeディレクトリでコマンドを実行します。

```bash
# gitステータスを確認
devhive exec fe-auth git status

# テストを実行
devhive exec fe-auth npm test

# 複数のコマンドを実行
devhive exec fe-auth -- bash -c "npm install && npm test"
```

---

## devhive roles

利用可能なロール一覧を表示します。

```bash
# 全ロール
devhive roles

# 組み込みロールのみ
devhive roles --builtin
devhive roles -b
```

### 組み込みロール

| ロール | 説明 |
|--------|------|
| `@frontend` | フロントエンド開発（React/Vue/TypeScript） |
| `@backend` | バックエンド開発（Python/Go/Node.js） |
| `@test` | テスト・QA（E2E、単体テスト） |
| `@docs` | ドキュメント作成 |
| `@security` | セキュリティレビュー |
| `@devops` | CI/CD・インフラ |

---

## devhive config

設定ファイルの内容を解析して表示します。

```bash
devhive config
```

### 出力例

```
=== DevHive Compose Configuration ===

Project: my-project
Version: 1

Defaults:
  Create Worktree: true
  Base Branch: develop
  Sprint: sprint-01

Roles:
  - frontend (file: roles/frontend.md)
  - backend (extends: @backend)

Workers:
  - fe-auth
    Branch: feat/auth-ui
    Role: frontend
    Task: 認証UIの実装...

  - be-api
    Branch: feat/auth-api
    Role: backend
    Task: 認証APIの実装
```

---

## 典型的なワークフロー

### 1. 新しいスプリントを開始

```bash
# 設定ファイルを作成
vim .devhive.yaml

# 全ワーカーを起動
devhive up --worktree

# 状態確認
devhive ps
```

### 2. 作業中の監視

```bash
# ログをリアルタイム監視
devhive logs -f

# 定期的に状態確認
devhive ps
```

### 3. 作業完了

```bash
# 全ワーカーを完了
devhive down

# ワーカーを削除
devhive rm fe-auth be-api docs
```

---

## 設定ファイルの検索順序

DevHiveは以下の順序で設定ファイルを検索します：

1. `.devhive.yaml`
2. `.devhive.yml`
3. `devhive.yaml`
4. `devhive.yml`

`-f` オプションで明示的に指定することも可能です。
