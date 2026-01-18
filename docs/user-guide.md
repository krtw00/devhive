# DevHive ユーザーガイド

DevHiveは、複数のAIワーカーによる並列開発を調整するためのCLIツールです。

## 目次

1. [概念](#概念)
2. [インストール](#インストール)
3. [クイックスタート](#クイックスタート)
4. [PMワークフロー](#pmワークフロー)
5. [ワーカーワークフロー](#ワーカーワークフロー)
6. [スプリント設定](#スプリント設定)
7. [Claude Code連携](#claude-code連携)
8. [トラブルシューティング](#トラブルシューティング)

---

## 概念

### 用語

| 用語 | 説明 |
|------|------|
| **プロジェクト** | 開発対象のリポジトリ。プロジェクトごとにDBが分離される |
| **スプリント** | 開発期間の単位。ワーカーの集合を定義 |
| **ロール** | ワーカーの役割テンプレート（frontend, backend等）。共通ルール・技術スタックを定義 |
| **ワーカー** | 実際に作業を行う単位。ロールを割り当て、固有のタスクを持つ |
| **タスク** | 各ワーカーへの具体的な作業指示 |

### アーキテクチャ

```
プロジェクト
  └── スプリント（sprint-01, sprint-02, ...）
        └── ワーカー（frontend-auth, backend-api, ...）
              ├── ロール（frontend, backend, ...）
              ├── ブランチ（feat/auth-ui）
              ├── Worktree（~/.devhive/projects/<project>/worktrees/<worker>）
              └── タスク（具体的な作業内容）
```

### 同じロールで複数ワーカー

```
Sprint: sprint-01
├── frontend-auth      (Role: frontend) → Task: 認証UIの実装
├── frontend-dashboard (Role: frontend) → Task: ダッシュボード実装
├── backend-auth       (Role: backend)  → Task: 認証APIの実装
└── test-e2e           (Role: test)     → Task: E2Eテスト作成
```

---

## インストール

### 1. DevHiveのビルドとインストール

```bash
cd /path/to/devhive
go build -o ~/bin/devhive ./cmd/devhive

# PATHに追加（~/.zshrc or ~/.bashrc）
export PATH="$HOME/bin:$PATH"
```

### 2. 依存ツールのインストール

```bash
# direnv（環境変数の自動設定）
curl -sfL https://direnv.net/install.sh | bash
echo 'eval "$(direnv hook zsh)"' >> ~/.zshrc

# yq（YAML解析、オプション）
# macOS
brew install yq
# Linux
sudo wget -qO /usr/local/bin/yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64
sudo chmod +x /usr/local/bin/yq
```

### 3. Claude Code Hooksの設定

```bash
# ~/.claude/settings.json に追加
```

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash|Edit|Write|NotebookEdit",
        "hooks": [
          {
            "type": "command",
            "command": "devhive worker session waiting_permission 2>/dev/null || true"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Bash|Edit|Write|NotebookEdit",
        "hooks": [
          {
            "type": "command",
            "command": "devhive worker session running 2>/dev/null || true"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "devhive worker session idle 2>/dev/null || true"
          }
        ]
      }
    ]
  }
}
```

---

## クイックスタート

### 1. プロジェクトの設定

```bash
cd /path/to/your-project

# プロジェクト識別ファイルを作成
echo "your-project-name" > .devhive
```

### 2. スプリントの初期化

```bash
devhive init sprint-01
```

### 3. ロールの登録

```bash
devhive role create frontend -d "フロントエンド担当" -f roles/frontend.md
devhive role create backend -d "バックエンド担当" -f roles/backend.md
```

### 4. ワーカーの登録

```bash
# Worktree自動作成付き
devhive worker register frontend-auth feat/auth-ui --role frontend --create-worktree
devhive worker register backend-api feat/api --role backend --create-worktree
```

### 5. ステータス確認

```bash
devhive status
```

---

## PMワークフロー

PMはスプリント全体を管理し、ワーカーの状態を監視します。

### スプリントの準備

```bash
# 1. スプリント初期化
devhive init sprint-01

# 2. ロール登録（初回のみ）
devhive role create frontend -d "フロントエンド" -f roles/frontend.md
devhive role create backend -d "バックエンド" -f roles/backend.md

# 3. ワーカー登録
devhive worker register fe-auth feat/auth-ui --role frontend -c
devhive worker register be-api feat/api --role backend -c
```

### 監視

```bash
# リアルタイム監視
devhive watch

# ステータス確認
devhive status

# レポート生成
devhive sprint report
```

### コミュニケーション

```bash
# 特定ワーカーへメッセージ
devhive msg send fe-auth "APIの仕様が変更されました"

# 全員へブロードキャスト
devhive msg broadcast "15分後にマージします"
```

### スプリント完了

```bash
devhive sprint complete
```

---

## ワーカーワークフロー

各ワーカーは自分のWorktreeで作業します。

### 環境設定

```bash
# Worktreeに移動（direnvで自動設定される）
cd ~/.devhive/projects/your-project/worktrees/fe-auth

# 初回のみdirenv許可
direnv allow
```

### 作業開始

```bash
# 環境確認
devhive worker show
devhive msg unread

# 作業開始
devhive worker start --task "認証UIの実装"
```

### 作業中

```bash
# タスク進捗更新
devhive worker task "ログインフォーム実装中"

# エラー報告
devhive worker error "ビルドエラー: missing dependency"

# メッセージ確認
devhive msg unread
devhive msg read all
```

### 作業完了

```bash
devhive worker complete
```

---

## スプリント設定

### YAML形式（推奨）

`scripts/devhive/sprints/sprint-XX.yaml`:

```yaml
# Sprint 06: パフォーマンス改善
# Issues: #312, #34

workers:
  - name: perf-be
    branch: perf/api-optimize
    role: backend
    task: |
      Issue #312: バックエンドAPIのパフォーマンス改善

      目標:
      - 統計APIのレスポンス時間改善
      - N+1クエリの修正

      作業内容:
      1. 現状のAPIレスポンス時間を計測
      2. SQLAlchemyクエリの最適化

  - name: perf-fe
    branch: perf/frontend-optimize
    role: frontend
    task: |
      Issue #312: フロントエンドのパフォーマンス改善

      目標:
      - 初期ロード時間の短縮

      作業内容:
      1. Lighthouse監査
      2. 不要な再レンダリングの修正
```

### ロールファイル

`scripts/devhive/roles/frontend.md`:

```markdown
# フロントエンドワーカー

## 基本ルール
- 日本語で思考・出力
- frontend/ 配下のファイルのみ変更

## 技術スタック
- Vue 3 (Composition API)
- TypeScript
- Vuetify 3

## コマンド
- テスト: `cd frontend && npm run test:unit`
- リント: `cd frontend && npm run lint`
```

---

## Claude Code連携

### セッション状態の自動追跡

Claude Code Hooksを設定すると、セッション状態が自動更新されます:

| タイミング | 状態 | アイコン |
|-----------|------|----------|
| ツール実行前 | waiting_permission | ⏸ |
| ツール実行後 | running | ▶ |
| セッション終了 | idle | ○ |

### tmux並列起動

`sprint-start.sh`でtmux上に複数ワーカーを一括起動:

```bash
# 基本起動
./scripts/devhive/sprint-start.sh

# claude自動起動
./scripts/devhive/sprint-start.sh -c

# claude + タスク自動送信
./scripts/devhive/sprint-start.sh -i
```

**作成されるウィンドウ:**
- `workers` - 全ワーカーのペイン（タイル配置）
- `monitor` - `devhive watch` 実行中
- `status` - `devhive status` 自動更新

---

## トラブルシューティング

### devhiveコマンドが見つからない

```bash
# PATHを確認
echo $PATH | tr ':' '\n' | grep bin

# 追加
export PATH="$HOME/bin:$PATH"
```

### プロジェクトが検出されない

```bash
# .devhiveファイルを確認
cat .devhive

# または環境変数で指定
export DEVHIVE_PROJECT=your-project
```

### ワーカー名が設定されない

```bash
# Worktree内の.envrcを確認
cat .envrc

# direnv許可
direnv allow
```

### Hooksが動作しない

```bash
# 手動でテスト
devhive worker session running

# 設定確認
cat ~/.claude/settings.json | jq '.hooks'
```

### Worktreeの作成に失敗

```bash
# ブランチが既に存在する場合
git branch -a | grep <branch-name>

# 手動作成
git worktree add ~/.devhive/projects/<project>/worktrees/<worker> <branch>
```
