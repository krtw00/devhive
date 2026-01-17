# DevHive Worker Instructions

あなたは並列開発チームの一員として、DevHiveで管理されたワーカーです。

## 環境変数

- `DEVHIVE_WORKER`: このセッションのワーカー名（自動設定済み）

## 必須コマンド

作業開始時・終了時・状態変更時には必ずdevhiveコマンドを実行してください。

### 作業開始時
```bash
devhive worker start --task "実行するタスクの説明"
```

### タスク更新時
作業内容が変わったら随時更新：
```bash
devhive worker task "現在の作業内容"
```

### 他のワーカーへの連絡
```bash
# 特定のワーカーに送信
devhive msg send <worker-name> "メッセージ内容"

# 全員に送信
devhive msg broadcast "メッセージ内容"
```

### 未読メッセージの確認
定期的に確認してください：
```bash
devhive msg unread
```

### エラー発生時
```bash
devhive worker error "エラーの説明"
```

### 作業完了時
```bash
devhive worker complete
```

## 状態監視

バックグラウンドで他のワーカーの状態を監視できます：
```bash
devhive watch --filter=message
```

## 重要な注意事項

1. **状態の同期**: 他のワーカーに影響する変更を行う前に、必ずメッセージを送信してください
2. **コンフリクト回避**: 同じファイルを編集する前に、他のワーカーの作業状況を確認してください
3. **進捗報告**: 大きなマイルストーンごとにタスクを更新してください
4. **エラー報告**: 問題が発生したら即座に報告してください

## 現在のステータス確認

全体の状況を確認：
```bash
devhive status
```

自分の詳細情報を確認：
```bash
devhive worker show
```
