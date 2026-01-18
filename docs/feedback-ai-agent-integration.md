# フィードバック: AIエージェント統合の課題 (v3)

## 概要

Katorin2プロジェクトでDevHive（compose.md更新後）を使用した並列AI開発を試行。

## 試行結果

### ドキュメントに記載されているが未実装の機能

`docs/commands/compose.md` に以下の機能が記載されているが、実際には動作しない：

#### 1. `defaults.auto_prompt`

**ドキュメント記載**:
```yaml
defaults:
  auto_prompt: true  # AIツール起動時に初期プロンプトを自動生成
```

**期待動作**:
> `auto_prompt: true` を設定すると、`devhive tmux` 実行時にAIツールに初期プロンプトが渡されます：
> - **claude**: `"CLAUDE.mdを読んでタスクを実行してください。..."`

**実際の動作**:
- `devhive tmux --dry-run` では `claude` のみ表示（プロンプトなし）
- 起動後、Claudeは入力待ち状態

#### 2. `defaults.tool_args`

**ドキュメント記載**:
```yaml
defaults:
  tool_args:
    claude: "--dangerously-skip-permissions"
```

**期待動作**:
> ツール別のデフォルト引数を設定できます

**実際の動作**:
- `devhive tmux --dry-run` では引数なしで `claude` のみ
- 実行コマンド: `export DEVHIVE_WORKER=bracket && claude`（引数なし）

#### 3. `defaults.direnv_allow`

**ドキュメント記載**:
```yaml
defaults:
  direnv_allow: true  # devhive up時に自動でdirenv allow
```

**実際の動作**:
```
direnv: error /home/.../worktrees/bracket/.envrc is blocked.
Run `direnv allow` to approve its content
```

### 実装されている機能

以下は正常に動作：

- `tool: claude` で `CLAUDE.md` 生成
- `CLAUDE.md` にロール・タスク内容が展開される
- `devhive tmux` でtmuxセッション作成
- `DEVHIVE_WORKER` 環境変数の設定

## 実装が必要な箇所

### 1. tmuxコマンド生成部分

現在:
```go
// 推測: tmux.go内
command := toolName  // "claude"
```

必要な変更:
```go
command := toolName
if args := getToolArgs(toolName); args != "" {
    command += " " + args
}
if autoPrompt && prompt := generateAutoPrompt(toolName); prompt != "" {
    command += " " + shellescape(prompt)
}
```

### 2. devhive up時のdirenv処理

```go
// worktree作成後
if defaults.DirenvAllow {
    exec.Command("direnv", "allow", worktreePath).Run()
}
```

### 3. 設定ファイルパース

`defaults` セクションの新フィールドをパースするstructの更新が必要:

```go
type Defaults struct {
    BaseBranch     string            `yaml:"base_branch"`
    AutoPrompt     bool              `yaml:"auto_prompt"`
    ToolArgs       map[string]string `yaml:"tool_args"`
    DirenvAllow    bool              `yaml:"direnv_allow"`
    PromptTemplate string            `yaml:"prompt_template"`
}
```

## テスト用設定

```yaml
# .devhive.yaml
version: "1"
project: Katorin2

defaults:
  base_branch: main
  auto_prompt: true
  direnv_allow: true
  tool_args:
    claude: "--dangerously-skip-permissions"

workers:
  series:
    branch: feat/series-complete
    role: .devhive/roles/fullstack.md
    task: .devhive/tasks/series.md
    tool: claude
```

**期待されるtmuxコマンド**:
```bash
export DEVHIVE_WORKER=series && claude --dangerously-skip-permissions "CLAUDE.mdを読んでタスクを実行してください。進捗は devhive progress series <0-100> で報告してください。"
```

## 関連

- 試行プロジェクト: Katorin2
- 試行日: 2026-01-18
- compose.md: 機能ドキュメント（実装より先行）
