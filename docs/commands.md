# DevHive コマンドリファレンス

## グローバルオプション

```
-h, --help    ヘルプを表示
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
devhive init sprint-05 --config ./sprint-05.conf --project /home/user/myproject
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
| --json | JSON形式で出力（未実装） |

### 出力例

```
Sprint: sprint-05 (started: 2026-01-18 10:00)

WORKER    BRANCH             ISSUE  STATUS     COMMIT   REVIEWS  MSGS
------    ------             -----  ------     ------   -------  ----
security  fix/security-auth  #313   🔨 working abc1234  1        0
quality   fix/quality-check  #314   ⏳ pending          0        2

Pending Reviews: 1
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
| --pane | -p | tmuxペインID |
| --worktree | -w | Worktreeパス |

#### 例

```bash
devhive worker register security fix/security-auth -i "#313" -p 1
devhive worker register quality fix/quality-check -i "#314" -p 2 -w /path/to/worktree
```

### devhive worker start

ワーカーの作業を開始状態にする。

```bash
devhive worker start <name> [flags]
```

#### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --task | -t | 現在のタスク説明 |

#### 例

```bash
devhive worker start security
devhive worker start security -t "認証APIの実装"
```

### devhive worker complete

ワーカーの作業を完了状態にする。

```bash
devhive worker complete <name>
```

#### 例

```bash
devhive worker complete security
```

### devhive worker status

ワーカーの状態を手動で更新する。

```bash
devhive worker status <name> <status>
```

#### 有効なステータス

- `pending` - 待機中
- `working` - 作業中
- `review_pending` - レビュー待ち
- `completed` - 完了
- `blocked` - ブロック中
- `error` - エラー

#### 例

```bash
devhive worker status security blocked
```

---

## devhive review

レビュー管理コマンド群。

### devhive review request

レビューを依頼する。

```bash
devhive review request <commit> [flags]
```

#### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --worker | -w | ワーカー名（必須） |
| --desc | -d | 変更内容の説明 |

#### 例

```bash
devhive review request abc1234 -w security -d "認証機能の追加"
```

### devhive review list

未処理のレビュー一覧を表示する。

```bash
devhive review list
```

#### 出力例

```
ID  WORKER    COMMIT   BRANCH             ISSUE  DESCRIPTION     CREATED
--  ------    ------   ------             -----  -----------     -------
1   security  abc1234  fix/security-auth  #313   認証機能の追加  10:30
2   quality   def5678  fix/quality-check  #314   品質チェック追加 10:45
```

### devhive review ok

レビューを承認する。

```bash
devhive review ok <id> [comment] [flags]
```

#### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --reviewer | -r | レビュアー名（デフォルト: senior） |

#### 例

```bash
devhive review ok 1
devhive review ok 1 "問題なし"
devhive review ok 1 "LGTM" -r pm
```

### devhive review fix

レビューで修正を依頼する。

```bash
devhive review fix <id> <comment> [flags]
```

#### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --reviewer | -r | レビュアー名（デフォルト: senior） |

#### 例

```bash
devhive review fix 1 "エラーハンドリングを追加してください"
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
| --from | -f | 送信者名（デフォルト: pm） |
| --type | -t | メッセージ種別（デフォルト: info） |
| --subject | -s | 件名 |

#### メッセージ種別

- `info` - 一般情報
- `warning` - 警告
- `conflict` - 競合通知
- `question` - 質問
- `answer` - 回答
- `system` - システム通知

#### 例

```bash
devhive msg send quality "DuelTable.vueを編集します" -f security
devhive msg send mobile-layout "APIを変更しました" -f backend -t warning -s "API変更通知"
```

### devhive msg broadcast

全ワーカーにメッセージをブロードキャストする。

```bash
devhive msg broadcast <message> [flags]
```

#### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --from | -f | 送信者名（デフォルト: pm） |
| --type | -t | メッセージ種別（デフォルト: info） |
| --subject | -s | 件名 |

#### 例

```bash
devhive msg broadcast "15分後にマージします" -f pm
devhive msg broadcast "API仕様が変わりました" -f backend -t warning
```

### devhive msg unread

未読メッセージを表示する。

```bash
devhive msg unread [worker]
```

#### 例

```bash
devhive msg unread           # 全ての未読メッセージ
devhive msg unread security  # securityワーカー宛の未読メッセージ
```

### devhive msg read

メッセージを既読にする。

```bash
devhive msg read <id|all> [flags]
```

#### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --worker | -w | ワーカー名（allの場合は必須） |

#### 例

```bash
devhive msg read 5                    # ID=5のメッセージを既読に
devhive msg read all -w security      # securityの全メッセージを既読に
```

---

## devhive lock

ファイルロック管理コマンド群。

### devhive lock acquire

ファイルをロックする。

```bash
devhive lock acquire <file> [flags]
```

#### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --worker | -w | ワーカー名（必須） |
| --reason | -r | ロック理由 |

#### 例

```bash
devhive lock acquire src/components/DuelTable.vue -w security
devhive lock acquire src/auth.py -w security -r "認証ロジック変更"
```

### devhive lock release

ファイルのロックを解除する。

```bash
devhive lock release <file> [flags]
```

#### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --worker | -w | ワーカー名（必須） |

#### 例

```bash
devhive lock release src/components/DuelTable.vue -w security
```

### devhive lock list

現在のロック一覧を表示する。

```bash
devhive lock list
```

#### 出力例

```
FILE                              LOCKED BY  REASON        SINCE
----                              ---------  ------        -----
src/components/DuelTable.vue      security   編集中         5m30s
src/auth.py                       backend    認証ロジック   2m10s
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

### 例

```bash
devhive events
devhive events --limit 20
devhive events -t review_requested
devhive events -w security --limit 10
```

### 出力例

```
10:45:30 file_locked [security] {file:src/auth.py}
10:44:15 review_requested [security] {commit:abc1234}
10:43:00 worker_status_changed [security] {status:working}
10:42:30 worker_registered [security] {branch:fix/security-auth,issue:#313}
10:42:00 sprint_created {sprint_id:sprint-05}
```

---

## devhive version

バージョン情報を表示する。

```bash
devhive version
```

### 出力例

```
devhive v0.1.0
```
