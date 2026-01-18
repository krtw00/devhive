# フィードバック: AIエージェント統合の課題 (v2)

## 概要

Katorin2プロジェクトでDevHive v2（tmux統合版）を使用した並列AI開発を試行した際に発見した問題点と改善提案。

## 試行内容

1. 4つのワーカー（series, team-ui, bracket, realtime）を`tool: claude`で定義
2. `devhive up` で CLAUDE.md が自動生成されることを確認
3. `devhive tmux` で4分割tmuxセッションを起動
4. 各ペインでClaude Codeが起動

## 改善された点

- `tool: claude` 指定で `CLAUDE.md` が自動生成される
- ロール内容がCLAUDE.mdに完全展開される
- タスク内容がCLAUDE.mdに完全展開される
- `devhive tmux` コマンドでtmux統合が簡単に
- `DEVHIVE_WORKER` 環境変数が自動設定される

## 残存する問題点

### 1. Claude Codeがタスクを自動実行しない

**現象**:
```
❯ export DEVHIVE_WORKER=bracket && claude

 ▐▛███▜▌   Claude Code v2.1.12
▝▜█████▛▘  Opus 4.5 · Claude Max

─────────────────────────────────────────────────────────────────────────────
❯ Try "refactor tournament.ts"    ← 入力待ち状態
```

**原因**:
- Claude Codeは`CLAUDE.md`を**参照用**として読み込むが、自動実行はしない
- 起動後にユーザー入力を待つ

**期待動作**:
- Claude Code起動時に`CLAUDE.md`のタスクを自動的に開始する

### 2. direnvエラー

**現象**:
```
direnv: error /home/.../worktrees/bracket/.envrc is blocked.
Run `direnv allow` to approve its content
```

**原因**:
- DevHiveが生成する`.envrc`ファイルがdirenvでブロックされる
- 各worktreeで手動で`direnv allow`が必要

**改善案**:
- `.envrc`を生成しないオプション
- または`devhive up`時に自動で`direnv allow`を実行

### 3. 初期プロンプトが渡されない

**現状**:
```bash
claude  # 引数なしで起動
```

**問題**:
- Claude Codeに初期タスクを渡す方法がない
- `claude "タスクを実行"` のような形式が必要

## 改善提案

### 提案1: 初期プロンプト付きで起動

```yaml
workers:
  series:
    branch: feat/series-complete
    tool: claude
    command: claude "CLAUDE.mdのタスクを実行してください"
    # または
    auto_prompt: true  # CLAUDE.md の内容を自動でプロンプトとして渡す
```

### 提案2: --dangerously-skip-permissions オプション

```yaml
defaults:
  claude_flags:
    - "--dangerously-skip-permissions"

workers:
  series:
    tool: claude
    # → claude --dangerously-skip-permissions で起動
```

### 提案3: カスタムコマンド

```yaml
workers:
  series:
    tool: claude
    command: |
      claude --dangerously-skip-permissions \
        "CLAUDE.mdを読んでタスクを実行してください。完了したらdevhive progressで報告してください。"
```

### 提案4: direnv自動許可

```yaml
defaults:
  direnv_allow: true  # devhive up 時に自動で direnv allow

# または .envrc を生成しない
defaults:
  generate_envrc: false
```

### 提案5: Claude Code専用起動スクリプト

DevHiveが各worktreeに起動スクリプトを生成:

```bash
# .devhive/worktrees/series/start.sh
#!/bin/bash
export DEVHIVE_WORKER=series
cd "$(dirname "$0")"

# CLAUDE.md の Task セクションを抽出してプロンプトに
TASK=$(sed -n '/^## タスク$/,/^## /p' CLAUDE.md | head -n -1)

claude --dangerously-skip-permissions "$TASK"
```

## 優先度

1. **高**: 初期プロンプト付きでClaude起動（提案1 or 3）
2. **高**: --dangerously-skip-permissions オプション対応（提案2）
3. **中**: direnv問題の解決（提案4）
4. **低**: 起動スクリプト生成（提案5）

## 補足: Claude Codeの自動実行について

Claude Codeは`CLAUDE.md`を**コンテキスト**として読み込むが、それだけでは自動実行されない。
自動実行には以下のいずれかが必要:

1. **引数でプロンプトを渡す**: `claude "タスクを実行"`
2. **--print オプション**: `claude --print "タスク"` （非対話モード）
3. **パイプ入力**: `echo "タスクを実行" | claude`

DevHiveのtmux統合では、これらの方法でClaude Codeに初期タスクを渡す仕組みが必要。

## 関連

- 試行プロジェクト: Katorin2（大会管理システム）
- 試行日: 2026-01-18
- DevHiveバージョン: tmux統合版
