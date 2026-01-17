# Events コマンド

[← 目次に戻る](index.md)

## devhive events

最近のイベントを表示する。

```bash
devhive events [flags]
```

### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --limit | -l | 表示件数（デフォルト: 20） |
| --type | -t | イベントタイプでフィルタ |
| --worker | -w | ワーカー名でフィルタ |
| --json | | JSON形式で出力 |

### イベントタイプ

| タイプ | 説明 |
|--------|------|
| sprint_started | スプリント開始 |
| sprint_completed | スプリント完了 |
| worker_registered | ワーカー登録 |
| worker_status_changed | ワーカーステータス変更 |
| worker_session_changed | セッション状態変更 |
| message_sent | メッセージ送信 |
| error_reported | エラー報告 |

### 例

```bash
# 最近のイベント20件
devhive events

# 直近50件
devhive events -l 50

# ステータス変更イベントのみ
devhive events -t worker_status_changed

# 特定ワーカーのイベント
devhive events -w frontend

# 複合フィルタ
devhive events -w frontend -t worker_status_changed -l 10
```

### 出力例

```
TIME                 TYPE                    WORKER    DETAILS
----                 ----                    ------    -------
10:30:05             worker_status_changed   frontend  pending → working
10:30:00             worker_session_changed  frontend  stopped → running
10:25:00             message_sent            pm        → frontend: タスク確認
10:20:00             worker_registered       frontend  branch: feat/ui
```

---

## devhive watch

イベントをリアルタイムで監視する。

```bash
devhive watch [flags]
```

### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --interval | -i | ポーリング間隔（秒、デフォルト: 2） |
| --type | -t | イベントタイプでフィルタ |
| --worker | -w | ワーカー名でフィルタ |

### 例

```bash
# すべてのイベントを監視
devhive watch

# 5秒間隔で監視
devhive watch -i 5

# 特定ワーカーのみ監視
devhive watch -w frontend

# エラーイベントのみ監視
devhive watch -t error_reported
```

### 出力例

```
Watching for events... (Ctrl+C to stop)
[10:30:05] worker_status_changed | frontend | pending → working
[10:30:10] worker_session_changed | frontend | idle → running
[10:31:00] message_sent | backend | → frontend: API ready
```

---

## イベント活用例

### PMダッシュボード

別ターミナルで `devhive watch` を実行し、全体の進捗をリアルタイム監視:

```bash
# ターミナル1: 全イベント監視
devhive watch

# ターミナル2: エラーのみ監視
devhive watch -t error_reported
```

### スクリプトからの利用

```bash
# 最新イベントをJSON取得してパース
devhive events -l 1 --json | jq '.events[0]'

# 特定ワーカーの最新ステータス変更を取得
devhive events -w frontend -t worker_status_changed -l 1 --json
```
