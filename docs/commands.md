# DevHive コマンドリファレンス

## グローバルオプション

```
-h, --help    ヘルプを表示
--json        JSON形式で出力（対応コマンドのみ）
```

## 環境変数

| 変数名 | 説明 | 例 |
|--------|------|-----|
| DEVHIVE_WORKER | デフォルトのワーカー名 | security |

```bash
export DEVHIVE_WORKER=security
```

---

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
Sprint: sprint-05 (started: 2025-01-18 10:00)

WORKER    BRANCH             ISSUE  STATUS      TASK                MSGS
------    ------             -----  ------      ----                ----
security  fix/security-auth  #313   working     認証APIの実装       0
quality   fix/quality-check  #314   pending                         2
```

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

## devhive worker

ワーカー管理コマンド群。

### devhive worker register

ワーカーを登録する。

```bash
devhive worker register <name> <branch> [flags]
```

#### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --issue | -i | Issue番号 |
| --worktree | -w | Worktreeパス |

#### 例

```bash
devhive worker register security fix/security-auth --issue "#313"
devhive worker register quality fix/quality-check -i "#314" -w /path/to/worktree
```

### devhive worker start

ワーカーの作業を開始状態にする。

```bash
devhive worker start [name] [flags]
```

#### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --task | -t | 現在のタスク説明 |

#### 例

```bash
devhive worker start security
devhive worker start --task "認証APIの実装"   # DEVHIVE_WORKER使用
```

### devhive worker complete

ワーカーの作業を完了状態にする。

```bash
devhive worker complete [name]
```

#### 例

```bash
devhive worker complete security
devhive worker complete   # DEVHIVE_WORKER使用
```

### devhive worker status

ワーカーの状態を手動で更新する。

```bash
devhive worker status [name] <status>
```

#### 有効なステータス

- `pending` - 待機中
- `working` - 作業中
- `completed` - 完了
- `blocked` - ブロック中
- `error` - エラー

#### 例

```bash
devhive worker status security blocked
devhive worker status blocked   # DEVHIVE_WORKER使用
```

### devhive worker show

ワーカーの詳細情報を表示する。

```bash
devhive worker show [name] [flags]
```

#### フラグ

| フラグ | 説明 |
|--------|------|
| --json | JSON形式で出力 |

#### 例

```bash
devhive worker show security
devhive worker show --json
```

#### 出力例

```
Worker: security
Branch: fix/security-auth
Issue: #313
Worktree: /home/user/project-security
Status: working
Task: 認証APIの実装
Last Commit: abc1234
Errors: 0
Updated: 2025-01-18 10:30:00
```

### devhive worker task

現在のタスク説明を更新する。

```bash
devhive worker task <task>
```

#### 例

```bash
devhive worker task "トークン検証の実装中"
```

### devhive worker error

エラーを報告し、ワーカーをエラー状態にする。

```bash
devhive worker error <message>
```

#### 例

```bash
devhive worker error "ビルドが失敗しました: missing dependency"
```

---

## devhive msg

メッセージ管理コマンド群。

### devhive msg send

特定のワーカーにメッセージを送信する。

```bash
devhive msg send <to> <message> [flags]
```

#### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --type | -t | メッセージ種別（デフォルト: info） |
| --subject | -s | 件名 |

#### メッセージ種別

- `info` - 一般情報
- `warning` - 警告
- `question` - 質問
- `answer` - 回答
- `system` - システム通知

#### 例

```bash
devhive msg send quality "認証APIを変更しました"
devhive msg send quality "APIが変わります" --type warning --subject "API変更通知"
```

### devhive msg broadcast

全ワーカーにメッセージをブロードキャストする。

```bash
devhive msg broadcast <message> [flags]
```

#### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --type | -t | メッセージ種別（デフォルト: info） |
| --subject | -s | 件名 |

#### 例

```bash
devhive msg broadcast "15分後にマージします"
devhive msg broadcast "API仕様変更" --type warning
```

### devhive msg unread

未読メッセージを表示する。

```bash
devhive msg unread [flags]
```

#### フラグ

| フラグ | 説明 |
|--------|------|
| --json | JSON形式で出力 |

#### 例

```bash
devhive msg unread
devhive msg unread --json
```

#### 出力例

```
[1] quality → you (10:30)
    認証APIを変更しました

[2] (broadcast) pm (10:45)
    Subject: 進捗確認
    各自の進捗を報告してください
```

### devhive msg read

メッセージを既読にする。

```bash
devhive msg read <id|all>
```

#### 例

```bash
devhive msg read 5          # ID=5のメッセージを既読に
devhive msg read all        # 全メッセージを既読に
```

---

## devhive events

イベントログを表示する。

```bash
devhive events [flags]
```

### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --limit | -l | 表示件数（デフォルト: 50） |
| --type | -t | イベント種別でフィルタ |
| --worker | -w | ワーカーでフィルタ |
| --json | | JSON形式で出力 |

### 例

```bash
devhive events
devhive events --limit 20
devhive events --type worker_status_changed
devhive events --worker security --limit 10
```

### 出力例

```
10:45:30 message_sent [security] {to:quality,type:info}
10:44:15 worker_status_changed [security] {status:working}
10:43:00 worker_registered [security] {branch:fix/security-auth,issue:#313}
10:42:00 sprint_created {sprint_id:sprint-05}
```

---

## devhive watch

状態変化をリアルタイムで監視する。

```bash
devhive watch [flags]
```

### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --filter | -f | 監視対象のフィルタ |

### フィルタ値

- `message` - メッセージのみ
- `worker` - ワーカー状態変化のみ
- (なし) - 全ての変化

### 例

```bash
devhive watch                    # 全変化を監視
devhive watch --filter=message   # メッセージのみ
devhive watch --filter=worker    # ワーカー状態変化のみ
```

### 出力例

```
[12:34:56] message: quality → you: "DuelTable.vue編集します"
[12:35:10] worker: quality → completed
[12:36:00] message: (broadcast) pm: "15分後にマージします"
```

Ctrl+Cで終了。

---

## devhive version

バージョン情報を表示する。

```bash
devhive version
```

### 出力例

```
devhive v0.2.0
```
