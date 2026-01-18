# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.3] - 2025-01-18

### Added
- `devhive down --clean` オプション: ワーカー完了時にworktreeとブランチも一括削除

## [0.3.2] - 2025-01-18

### Fixed
- tmuxペインの作成順序がyaml定義順に従うように修正
- `devhive ps`で完了済みワーカーが非表示の場合、その旨を表示するように改善

## [0.3.1] - 2025-01-18

### Fixed
- tmuxペイン作成失敗時のペイン数表示が正確になった（実際に成功したペイン数を表示）
- `--no-worktree`で登録したワーカーもtmuxで起動可能に（プロジェクトルートで実行）
- split-windowエラー時に詳細なエラーメッセージとヒントを表示
- 各ペイン分割前にtiledレイアウトを適用し、多数ペイン作成の成功率を向上

## [0.3.0] - 2025-01-18

### Added
- `devhive tmux` - 全ワーカーをtmuxペインで一括起動
- `devhive tmux-kill` - tmuxセッションを終了
- `devhive tmux-list` - DevHive tmuxセッション一覧
- `devhive start <worker>` - 個別ワーカーをフォアグラウンドで起動
- AIツール指定機能（`tool: claude`, `tool: codex`, `tool: gemini`）
- コンテキストファイル自動生成（`.devhive/context/<worker>.md`）
- `--dangerously-skip-permissions`オプションのデフォルト有効化
- direnv allow実行の修正
- `auto_complete`機能の追加

## [0.2.0] - 2025-01-17

### Changed
- Docker風インターフェースに完全移行
- コマンド体系を刷新（`devhive up/down/ps/logs`等）

### Added
- `devhive init` - プロジェクト初期化コマンド
- `devhive progress` - 進捗更新
- `devhive merge` - ブランチマージ
- `devhive diff` - 変更差分表示
- `devhive note` - メモ追記
- `devhive clean` - 完了済みワーカー削除
- 通信機能（`request`, `report`, `reply`, `broadcast`, `inbox`, `msgs`）
- ヘルプ出力のカテゴリ別グループ化

## [0.1.0] - 2025-01-16

### Added
- 初期実装
- Git Worktree + 複数AIエージェント並列開発の状態管理
- `.devhive.yaml`設定ファイルサポート
- ロール機能によるワーカー管理
- プロジェクト単位でのDB分離
- direnv連携（.envrc自動生成）
- ユーザーガイドと仕様書

[Unreleased]: https://github.com/iguchi/devhive/compare/v0.3.1...HEAD
[0.3.1]: https://github.com/iguchi/devhive/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/iguchi/devhive/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/iguchi/devhive/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/iguchi/devhive/releases/tag/v0.1.0
