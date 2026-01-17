# Cleanup コマンド

[← 目次に戻る](index.md)

## devhive cleanup events

古いイベントを削除する。

```bash
devhive cleanup events [flags]
```

### フラグ

| フラグ | 説明 |
|--------|------|
| --days | 保持する日数（デフォルト: 30） |
| --dry-run | 削除せずに対象を表示 |

### 例

```bash
# 30日以上前のイベントを削除
devhive cleanup events

# 7日以上前のイベントを削除
devhive cleanup events --days 7

# 削除対象の確認のみ
devhive cleanup events --dry-run
```

---

## devhive cleanup messages

古い既読メッセージを削除する。

```bash
devhive cleanup messages [flags]
```

### フラグ

| フラグ | 説明 |
|--------|------|
| --days | 保持する日数（デフォルト: 30） |
| --dry-run | 削除せずに対象を表示 |

### 例

```bash
# 30日以上前の既読メッセージを削除
devhive cleanup messages

# 14日以上前の既読メッセージを削除
devhive cleanup messages --days 14

# 削除対象の確認のみ
devhive cleanup messages --dry-run
```

**注意**: 未読メッセージは削除されません。

---

## devhive cleanup worktrees

不要なWorktreeを削除する。

```bash
devhive cleanup worktrees [flags]
```

### フラグ

| フラグ | 説明 |
|--------|------|
| --dry-run | 削除せずに対象を表示（デフォルト: true） |
| --force | 完了済みワーカーのWorktreeも削除 |

### 例

```bash
# 削除対象の確認
devhive cleanup worktrees

# 実際に削除
devhive cleanup worktrees --dry-run=false

# 完了済みワーカーのWorktreeも含めて削除
devhive cleanup worktrees --dry-run=false --force
```

### 削除対象

- 登録されていないワーカーのWorktree
- `--force`指定時: 完了済みワーカーのWorktree

---

## devhive cleanup all

すべてのクリーンアップタスクを実行する。

```bash
devhive cleanup all [flags]
```

### フラグ

| フラグ | 説明 |
|--------|------|
| --days | 保持する日数（デフォルト: 30） |
| --dry-run | 削除せずに対象を表示 |

### 例

```bash
# すべてのクリーンアップを実行
devhive cleanup all

# 7日分のデータを保持
devhive cleanup all --days 7

# 削除対象の確認
devhive cleanup all --dry-run
```

### 出力例

```
Cleaning up events...
  ✓ Deleted 150 events
Cleaning up messages...
  ✓ Deleted 45 messages
```

---

## 定期クリーンアップ

cronやsystemdタイマーで定期実行することを推奨:

```bash
# crontab例: 毎日午前3時に実行
0 3 * * * devhive cleanup all --days 30
```

### スクリプト例

```bash
#!/bin/bash
# cleanup.sh - 週次クリーンアップスクリプト

echo "=== DevHive Weekly Cleanup ==="
date

# イベントとメッセージのクリーンアップ
devhive cleanup all --days 14

# 不要なWorktreeのクリーンアップ
devhive cleanup worktrees --dry-run=false

echo "=== Cleanup Complete ==="
```
