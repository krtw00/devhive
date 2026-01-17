# Sprint コマンド

[← 目次に戻る](index.md)

## devhive init

スプリントを初期化する。

```bash
devhive init <sprint-id> [flags]
```

### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --config | -c | 設定ファイルパス |
| --project | -p | プロジェクトパス |

### 例

```bash
devhive init sprint-05
devhive init sprint-05 --project /home/user/myproject
```

---

## devhive status

現在のスプリント状態を表示する。

```bash
devhive status [flags]
```

### フラグ

| フラグ | 説明 |
|--------|------|
| --json | JSON形式で出力 |

### 出力例

```
Project: myproject
Sprint: sprint-05 (started: 2025-01-18 10:00)

WORKER    ROLE      BRANCH             STATUS      SESSION    TASK                MSGS
------    ----      ------             ------      -------    ----                ----
security  security  fix/security-auth  🔧 working  ▶ running  認証APIの実装       0
quality   quality   fix/quality-check  ⏳ pending  ⏸ waiting                      2
```

**ステータスアイコン**:
- `⏳` pending, `🔧` working, `✓` completed, `🚫` blocked, `❌` error

**セッションアイコン**:
- `▶` running, `⏸` waiting_permission, `○` idle, `■` stopped

---

## devhive sprint complete

アクティブなスプリントを完了状態にする。

```bash
devhive sprint complete
```

### 例

```bash
devhive sprint complete
# ✓ Sprint 'sprint-05' completed
```

---

## devhive sprint setup

設定ファイルからワーカーを一括登録する。

```bash
devhive sprint setup <config-file> [flags]
```

### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --create-worktrees | -c | 各ワーカーのWorktreeを自動作成 |
| --repo | | Gitリポジトリのパス（デフォルト: cwd） |

### 設定ファイル形式

**JSON形式**:
```json
{
  "workers": [
    {"name": "frontend", "branch": "feat/ui", "role": "frontend"},
    {"name": "backend", "branch": "feat/api", "role": "backend"}
  ]
}
```

**シンプル形式**（テキスト）:
```
# コメント
frontend feat/ui frontend
backend feat/api backend
```

### 例

```bash
# JSON設定からワーカーを登録
devhive sprint setup workers.json

# Worktreeも同時に作成
devhive sprint setup workers.json --create-worktrees

# 別のリポジトリを指定
devhive sprint setup workers.json -c --repo /path/to/repo
```

---

## devhive sprint report

スプリントレポートを生成する。

```bash
devhive sprint report [flags]
```

### フラグ

| フラグ | 説明 |
|--------|------|
| --json | JSON形式で出力 |

### 例

```bash
devhive sprint report
devhive sprint report --json > report.json
```

### 出力例

```
═══════════════════════════════════════════════════════════
  Sprint Report: sprint-05
═══════════════════════════════════════════════════════════
Started: 2025-01-18 10:00:00
Status: active

Workers:
───────────────────────────────────────────────────────────
  Total: 3  |  Completed: 1  |  Working: 2  |  Pending: 0  |  Error: 0

  frontend (frontend)
    Branch: feat/ui
    Status: ✅ done | Session: ■ stopped

  backend (backend)
    Branch: feat/api
    Status: 🔨 working | Session: ▶ running
    Task: API実装中

Recent Activity:
───────────────────────────────────────────────────────────
  worker_status_changed: 5
  message_sent: 3
  worker_registered: 3
```

---

## devhive projects

全プロジェクトの状態を横断的に表示する。

```bash
devhive projects [flags]
```

### フラグ

| フラグ | 説明 |
|--------|------|
| --json | JSON形式で出力 |

### 例

```bash
devhive projects
devhive projects --json
```

### 出力例

```
PROJECT       SPRINT       STATUS  WORKERS
-------       ------       ------  -------
duel-log-app  sprint-05    active  frontend[▶] backend[⏸] test[■]
my-api        sprint-02    active  auth[▶] db[○]
```

**アイコンの意味**:
- `▶` running - 実行中
- `⏸` waiting_permission - 権限待ち
- `○` idle - 待機中
- `■` stopped - 停止
