# DevHive

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

  backend:
    branch: feat/api
    role: "@backend"
    task: API実装
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

## コマンド一覧

| コマンド | 説明 |
|----------|------|
| `devhive init [-t]` | 初期化（-t: テンプレート作成） |
| `devhive up` | ワーカー起動 |
| `devhive down` | ワーカー停止 |
| `devhive ps` | ワーカー一覧 |
| `devhive logs [-f]` | ログ表示 |
| `devhive exec <w> <cmd>` | コマンド実行 |
| `devhive roles [-b]` | ロール一覧 |

## 組み込みロール

| ロール | 説明 |
|--------|------|
| `@frontend` | フロントエンド |
| `@backend` | バックエンド |
| `@test` | テスト・QA |
| `@docs` | ドキュメント |
| `@security` | セキュリティ |
| `@devops` | CI/CD |

## gitignore

```
.devhive/
```

## License

MIT
