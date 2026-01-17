# Role コマンド

[← 目次に戻る](index.md)

## devhive role create

ロールを作成する。

```bash
devhive role create <name> [flags]
```

### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --description | -d | ロールの説明 |
| --file | -f | ロールファイルパス |
| --args | -a | 追加引数（Claude Code用など） |

### 例

```bash
# 基本的なロール作成
devhive role create frontend -d "Frontend developer"

# ロールファイルを指定
devhive role create security -f roles/security.md -d "Security specialist"

# 引数付きでロール作成
devhive role create backend -d "Backend developer" --args "--model opus"
```

---

## devhive role list

すべてのロールを一覧表示する。

```bash
devhive role list [flags]
```

### フラグ

| フラグ | 説明 |
|--------|------|
| --json | JSON形式で出力 |

### 例

```bash
devhive role list
devhive role list --json
```

### 出力例

```
NAME       DESCRIPTION          FILE               ARGS
----       -----------          ----               ----
frontend   Frontend developer   roles/frontend.md
backend    Backend developer    roles/backend.md   --model opus
security   Security specialist  roles/security.md  --allowedTools Edit,Read
```

---

## devhive role show

ロールの詳細情報を表示する。

```bash
devhive role show <name> [flags]
```

### フラグ

| フラグ | 説明 |
|--------|------|
| --json | JSON形式で出力 |

### 例

```bash
devhive role show frontend
devhive role show frontend --json
```

---

## devhive role update

ロールを更新する。

```bash
devhive role update <name> [flags]
```

### フラグ

| フラグ | 短縮 | 説明 |
|--------|------|------|
| --description | -d | ロールの説明 |
| --file | -f | ロールファイルパス |
| --args | -a | 追加引数 |

### 例

```bash
devhive role update frontend -d "Frontend & UI developer"
devhive role update backend --args "--model sonnet"
```

---

## devhive role delete

ロールを削除する。

```bash
devhive role delete <name>
```

### 例

```bash
devhive role delete frontend
```

**注意**: ロールを使用しているワーカーがいる場合、削除できません。
