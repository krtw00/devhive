# DevHive コマンドリファレンス

## 目次

- [グローバルオプション](#グローバルオプション)
- [Sprint コマンド](sprint.md) - スプリント管理
- [Worker コマンド](worker.md) - ワーカー管理
- [Role コマンド](role.md) - ロール管理
- [Message コマンド](msg.md) - メッセージ管理
- [Events コマンド](events.md) - イベント・監視
- [Cleanup コマンド](cleanup.md) - クリーンアップ
- [Hooks 連携](hooks.md) - Claude Code連携

## グローバルオプション

```
-h, --help              ヘルプを表示
-P, --project <name>    プロジェクト名を指定
--json                  JSON形式で出力（対応コマンドのみ）
```

## 環境変数

| 変数名 | 説明 | 例 |
|--------|------|-----|
| DEVHIVE_WORKER | デフォルトのワーカー名 | security |
| DEVHIVE_PROJECT | プロジェクト名（自動検出の最低優先度） | duel-log-app |

```bash
export DEVHIVE_WORKER=security
export DEVHIVE_PROJECT=duel-log-app
```

## プロジェクト自動検出

DevHiveはプロジェクトを以下の優先順位で自動検出する:

1. **`--project` / `-P` フラグ**（最優先）
2. **`.devhive` ファイル** - cwdから上位へ検索
3. **パス検出** - `~/.devhive/projects/<name>/...` 配下にいる場合
4. **`DEVHIVE_PROJECT` 環境変数**（最低優先度）

```bash
# フラグで指定
devhive -P myproject status

# .devhive ファイルを作成してプロジェクトルートに配置
echo "myproject" > /path/to/project/.devhive

# ~/.devhive/projects/ 配下で作業すると自動検出
cd ~/.devhive/projects/myproject/worktrees
devhive status
```

## クイックリファレンス

### PM（プロジェクトマネージャー）

```bash
devhive init sprint-01                              # スプリント開始
devhive sprint setup workers.json --create-worktrees # 一括ワーカー登録
devhive status                                      # 状態確認
devhive projects                                    # 全プロジェクト一覧
devhive sprint report                               # レポート生成
devhive sprint complete                             # スプリント完了
```

### ワーカー

```bash
export DEVHIVE_WORKER=frontend
devhive worker start --task "UI実装"   # 作業開始
devhive worker session running          # セッション状態更新
devhive msg unread                       # 未読確認
devhive worker task "ボタン実装中"      # タスク更新
devhive worker complete                  # 作業完了
```
