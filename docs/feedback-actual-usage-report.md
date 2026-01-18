# DevHive 実使用レポート

## 概要

Katorin2プロジェクトで4つのAIワーカーを並列稼働させ、約11分で4つのIssueを実装完了。

## 試行環境

- プロジェクト: Katorin2（大会管理システム）
- ワーカー数: 4
- AIツール: Claude Code v2.1.12 (Opus 4.5)
- 所要時間: 約11分

## 成功した点

### 1. auto_prompt が効果的に機能

```yaml
defaults:
  auto_prompt: true
  tool_args:
    claude: "--dangerously-skip-permissions"
```

**結果**: Claude Codeが起動直後にCLAUDE.mdを読んでタスク実行を開始。手動介入なしで全自動稼働。

### 2. CLAUDE.md の内容展開

ロールとタスクがCLAUDE.mdに完全展開され、各ワーカーが明確な指示を受け取れた。

### 3. devhive progress による進捗報告

全ワーカーが `devhive progress <worker> <0-100>` を適切に使用し、PMからの監視が容易だった。

### 4. devhive request review の活用

全ワーカーがタスク完了時に `devhive request review` でPMに通知。`devhive inbox` で一覧確認可能。

### 5. tmux統合

`devhive tmux` で4分割画面を自動作成。`--attach=false` でバックグラウンド実行も可能。

### 6. セッション状態のリアルタイム追跡

`devhive ps` で各ワーカーの状態（running/waiting_permission/idle）と進捗が一目で分かった。

## 改善が望まれる点

### 1. direnv_allow が機能していない

```
direnv: error /home/.../worktrees/bracket/.envrc is blocked.
```

**影響**: エラーメッセージが表示されるが、動作には影響なし（Claude Codeは正常稼働）

**提案**: `devhive up` 時に自動で `direnv allow` を実行するか、`.envrc` 生成をオプション化

### 2. ワーカー完了後の自動処理がない

現状: ワーカーが100%になってもstatusは`pending`のまま

**提案**:
- `status: completed` への自動遷移
- または `auto_down: true` オプションで自動完了処理

### 3. npm/pnpm install の待機時間

複数ワーカーが同時に `npm install` を実行すると、`waiting_permission` 状態が長く続く。

**提案**:
- `devhive up` 時に共有の `node_modules` を事前準備するオプション
- または各worktreeで `npm install` を事前実行

### 4. コミットの重複可能性

各ワーカーが独立してコミットするため、同じファイルを編集した場合にコンフリクトの可能性。

**提案**:
- マージ前の自動コンフリクトチェック
- `devhive merge` 時の警告表示

### 5. ビルドエラーの伝播

1つのワーカーが作成したコードが他ワーカーの型チェックに影響することがあった。

**提案**:
- 各worktreeは独立したブランチなので問題ないが、ワーカー間の依存関係を定義できると良い

## 使用統計

| 指標 | 値 |
|------|-----|
| 総所要時間 | 約11分 |
| 生成コード行数 | 約2,000行（推定） |
| コミット数 | 4 |
| レビュー依頼数 | 4 |
| エラー発生 | 0（direnv警告のみ） |

## ワーカー別実績

| Worker | 進捗推移 | 完了時間 | 成果物 |
|--------|----------|----------|--------|
| series | 5→15→35→55→80→100% | 5分46秒 | points.ts, Server Action, UIボタン |
| realtime | 5→5→5→100% | 6分9秒 | realtime.ts, カスタムフック, RealtimeBracket更新 |
| team-ui | 5→15→40→70→100% | 7分50秒 | 3コンポーネント（card, modal, bracket） |
| bracket | 5→25→50→80→100% | 10分57秒 | ダブルエリミ, スイスドロー（2形式） |

## 総評

DevHiveは並列AI開発のオーケストレーションツールとして**実用レベル**に達している。

### 特に優れている点

1. **ゼロ介入での自動実行**: `auto_prompt` + `tool_args` の組み合わせで完全自動化
2. **進捗可視化**: `devhive ps` でリアルタイム監視
3. **PM-Worker通信**: `request/inbox/reply` のワークフローが自然

### 今後の期待

1. ワーカー間の依存関係定義
2. 自動マージ・PR作成
3. ビルド/テスト結果の集約
4. Web UI（ダッシュボード）

## 結論

**推奨**: 独立性の高いタスク（機能追加、バグ修正、リファクタリング）の並列実行に最適。

**注意**: 相互依存するタスクは逐次実行を推奨。

---

試行日: 2026-01-18
DevHive バージョン: v0.9.0
