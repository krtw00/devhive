# DevHive コマンドリファレンス

## 目次

- [Compose コマンド](compose.md) - Docker風インターフェース
- [Hooks 連携](hooks.md) - Claude Code連携

## グローバルオプション

```
-h, --help    ヘルプを表示
```

## 環境変数

| 変数名 | 説明 | 設定方法 |
|--------|------|----------|
| DEVHIVE_WORKER | ワーカー名 | `devhive up` で自動生成される `.envrc` |

## クイックリファレンス

```bash
# 初期化
devhive init              # プロジェクト初期化
devhive init -t           # テンプレート付きで初期化

# 基本ワークフロー
devhive up                # 全て自動セットアップ
devhive ps                # ワーカー一覧
devhive logs -f           # ログをリアルタイム表示
devhive down              # ワーカー停止

# 個別操作
devhive start <worker>    # 特定ワーカー開始
devhive stop <worker>     # 特定ワーカー停止
devhive rm <worker>       # ワーカー削除
devhive exec <w> <cmd>    # worktreeでコマンド実行

# 情報
devhive roles -b          # 組み込みロール一覧
devhive config            # 設定表示
```
