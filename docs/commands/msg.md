# Message コマンド

[← 目次に戻る](index.md)

## devhive msg send

ワーカーにメッセージを送信する。

```bash
devhive msg send <to> <message>
```

### 例

```bash
devhive msg send frontend "APIの仕様が変更されました"
devhive msg send backend "認証トークンの形式を確認してください"
```

---

## devhive msg broadcast

全ワーカーにメッセージをブロードキャストする。

```bash
devhive msg broadcast <message>
```

### 例

```bash
devhive msg broadcast "15分後にミーティングがあります"
devhive msg broadcast "mainブランチが更新されました。リベースしてください"
```

---

## devhive msg unread

未読メッセージを表示する。

```bash
devhive msg unread [flags]
```

### フラグ

| フラグ | 説明 |
|--------|------|
| --json | JSON形式で出力 |

### 例

```bash
devhive msg unread
devhive msg unread --json
```

### 出力例

```
ID  FROM      TIME                 MESSAGE
--  ----      ----                 -------
5   pm        2025-01-18 10:30:00  APIの仕様が変更されました
3   backend   2025-01-18 10:15:00  認証部分の実装完了しました
```

---

## devhive msg read

メッセージを既読にする。

```bash
devhive msg read <id|all>
```

### 例

```bash
# 特定のメッセージを既読
devhive msg read 5

# すべてのメッセージを既読
devhive msg read all
```

---

## ワーカー間コミュニケーション

DevHiveのメッセージシステムは、ワーカー間およびPMとワーカー間のコミュニケーションを可能にします。

### 使用例

**PMからワーカーへ**:
```bash
devhive msg send frontend "優先度が変更されました"
devhive msg broadcast "スプリントの残り時間は2時間です"
```

**ワーカーからPM/他ワーカーへ**:
```bash
export DEVHIVE_WORKER=frontend
devhive msg send backend "APIエンドポイントの確認お願いします"
```

### メッセージの確認フロー

1. ワーカーは定期的に `devhive msg unread` で未読確認
2. 確認したら `devhive msg read <id>` または `devhive msg read all` で既読化
3. Claude Code Hooksと組み合わせることで自動チェックも可能
