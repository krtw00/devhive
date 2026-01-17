# Worker コマンド

[← 目次に戻る](index.md)

## devhive worker register

ワーカーを登録する。

```bash
devhive worker register <name> <branch> [flags]
```

### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --role | -r | ロール名（事前に `devhive role create` で作成） |
| --worktree | -w | Worktreeパス |
| --create-worktree | -c | Git Worktreeを自動作成 |
| --repo | | Gitリポジトリのパス（デフォルト: cwd） |

### 例

```bash
# 基本的な登録
devhive worker register security fix/security-auth --role security

# Worktreeパスを指定
devhive worker register quality fix/quality-check -r quality -w /path/to/worktree

# Worktreeを自動作成
devhive worker register frontend feat/frontend --create-worktree --role frontend

# 別のリポジトリからWorktreeを作成
devhive worker register backend feat/backend -c --repo /path/to/repo
```

**Worktree自動作成について**:
- `--create-worktree`（`-c`）を指定すると、`~/.devhive/projects/<project>/worktrees/<worker-name>`にWorktreeが作成される
- ブランチが存在しない場合は新規作成される

---

## devhive worker start

ワーカーの作業を開始状態にする。

```bash
devhive worker start [name] [flags]
```

### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --task | -t | 現在のタスク説明 |

### 例

```bash
devhive worker start security
devhive worker start --task "認証APIの実装"   # DEVHIVE_WORKER使用
```

---

## devhive worker complete

ワーカーの作業を完了状態にする。

```bash
devhive worker complete [name]
```

### 例

```bash
devhive worker complete security
devhive worker complete   # DEVHIVE_WORKER使用
```

---

## devhive worker status

ワーカーの状態を手動で更新する。

```bash
devhive worker status [name] <status>
```

### 有効なステータス

- `pending` - 待機中
- `working` - 作業中
- `completed` - 完了
- `blocked` - ブロック中
- `error` - エラー

### 例

```bash
devhive worker status security blocked
devhive worker status blocked   # DEVHIVE_WORKER使用
```

---

## devhive worker session

AIセッションの状態を更新する。

```bash
devhive worker session <state>
```

### 有効なセッション状態

| 状態 | アイコン | 説明 |
|------|----------|------|
| running | ▶ | セッションがアクティブに実行中 |
| waiting_permission | ⏸ | ユーザーの権限確認待ち |
| idle | ○ | セッションは開いているが待機中 |
| stopped | ■ | セッション終了 |

### 例

```bash
devhive worker session running
devhive worker session waiting_permission
devhive worker session stopped
```

---

## devhive worker show

ワーカーの詳細情報を表示する。

```bash
devhive worker show [name] [flags]
```

### フラグ

| フラグ | 説明 |
|--------|------|
| --json | JSON形式で出力 |

### 例

```bash
devhive worker show security
devhive worker show --json
```

### 出力例

```
Worker: security
Role: security
Role File: roles/security.md
Branch: fix/security-auth
Worktree: /home/user/project-security
Status: working
Session: ▶ running
Task: 認証APIの実装
Last Commit: abc1234
Errors: 0
Updated: 2025-01-18 10:30:00
Unread Messages: 0
```

---

## devhive worker task

現在のタスク説明を更新する。

```bash
devhive worker task <task>
```

### 例

```bash
devhive worker task "トークン検証の実装中"
```

---

## devhive worker error

エラーを報告し、ワーカーをエラー状態にする。

```bash
devhive worker error <message>
```

### 例

```bash
devhive worker error "ビルドが失敗しました: missing dependency"
```
