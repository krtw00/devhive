# agentmine Integration Memo for DevHive

**目的**: agentmineで確立した設計思想と監視機能をDevHiveに統合するための実装メモ

**作成日**: 2026-01-20

---

## 基本思想（agentmineで確立）

### 1. DBマスター方式

**原則**: すべてのデータはDBで管理（単一真実源）

**DevHiveへの適用**:
```
現状: .devhive.yaml + SQLite（ローカル）
↓
統合: PostgreSQL/SQLite（チーム共有可能）

データ構造:
- tasks（タスク情報）
- sessions（実行記録）
- agents（Agent定義 - 旧roles）
- memories（Memory Bank - 新規）
- settings（設定）
```

**実装影響**:
- `.devhive.yaml` → DB移行ツール必要
- `devhive up` → DB読み込み → worktree作成
- Worker情報もDB管理（`.devhive/workers/*.md` → DB）

---

### 2. Observable & Deterministic

**原則**: ステータスはAIの報告ではなく客観的事実で判定

**DevHiveの課題**:
```go
// 現状（気分依存）
devhive report "50%完了"     // AIが忘れる可能性
devhive request help "詰まった" // AIが報告しない可能性
```

**統合後（客観事実）**:
```go
// Git状態監視
commits := git.Log("main..HEAD")
diff := git.Diff("main")

// ログ解析
errors := analyzeLog(sessionLogPath)

// 自動判断
if errors > 5 {
  notifyOrchestrator("エラー多発")
}
```

**実装必要**:
- `devhive status` → Git状態取得
- `devhive monitor <w>` → ログ解析 + Git進捗表示
- 通信機能を「Observable Facts」ベースに再設計

---

### 3. Fail Fast

**原則**: エラーは即座に失敗、リカバリーは上位層の責務

**DevHiveへの適用**:
- Worker内でのエラーリトライ削除
- exit codeで成否判定
- Orchestrator（PM）がリトライ判断

**実装影響**:
```go
// Worker起動
cmd := exec.Command("claude-code", args...)
if err := cmd.Run(); err != nil {
  // リトライせず即座に失敗記録
  db.UpdateSession(workerID, Session{
    ExitCode: cmd.ProcessState.ExitCode(),
    Status: "failed",
  })
  return err
}
```

---

### 4. スコープ制御（案2 - デフォルトreadable）

**原則**: excludeを除く全ファイルが読み取り可能（read省略可）

**DevHiveへの適用**:
```yaml
# 現状: DevHiveにスコープ制御なし
workers:
  frontend:
    branch: feat/ui
    role: "@frontend"

# 統合後: Agent定義にスコープ追加
agents:
  frontend:
    scope:
      exclude:              # 物理的に除外
        - "**/*.env"
        - "**/secrets/**"
        - "packages/backend/**"

      # read: 省略可能（excludeを除く全ファイル）

      write:                # 編集可能
        - "packages/web/**"
        - "tests/web/**"
```

**実装必要**:
- sparse-checkout適用（exclude）
- chmod適用（write制御）
- worktree作成時にスコープ適用

---

## 監視機能（Phase 1実装）

### 問題の整理

**exit codeだけでは不十分**:
```
Worker実行中（exit code = null）
├─ 順調に進んでる？ ← わからない
├─ 詰まってる？ ← 検出できない
├─ エラー出てる？ ← 見えない
└─ 完全に停止してる？ ← 判断不能
```

**必要な情報**:
1. ログ解析（エラー・警告検出）
2. Git進捗推測（commits数、差分統計）
3. 最終活動時刻（停滞検出）

---

### 実装方針：オンデマンド取得

**案1（推奨）**: `devhive status`実行時のみ取得

```go
// services/monitor.go
type WorkerMonitor struct {
  db *sql.DB
}

// ログ解析（オンデマンド）
func (m *WorkerMonitor) AnalyzeSession(workerID string) (*SessionAnalysis, error) {
  logPath := fmt.Sprintf(".devhive/logs/session-%s.log", workerID)
  content, err := os.ReadFile(logPath)
  if err != nil {
    return nil, err
  }

  return &SessionAnalysis{
    Errors:   countMatches(content, "ERROR"),
    Warnings: countMatches(content, "WARN"),
    LastActivity: getFileModTime(logPath),
  }, nil
}

// Git進捗推測（オンデマンド）
func (m *WorkerMonitor) EstimateProgress(workerID string) (*ProgressEstimate, error) {
  worktreePath := fmt.Sprintf(".devhive/worktrees/%s", workerID)

  // Git状態取得
  commits, _ := execGit(worktreePath, "rev-list", "--count", "main..HEAD")
  stat, _ := execGit(worktreePath, "diff", "--shortstat", "main")

  return &ProgressEstimate{
    Commits: parseInt(commits),
    Insertions: parseStatInsertions(stat),
    Deletions: parseStatDeletions(stat),
  }, nil
}

// 統合ステータス
func (m *WorkerMonitor) GetWorkerStatus(workerID string) (*WorkerStatus, error) {
  session, _ := m.AnalyzeSession(workerID)
  progress, _ := m.EstimateProgress(workerID)

  return &WorkerStatus{
    Session: session,
    Progress: progress,
    Timestamp: time.Now(),
  }, nil
}
```

**実行タイミング**:
- `devhive status` 実行時
- `devhive monitor <w>` 実行時
- `devhive ps` 実行時（オプション）

**負荷**: ほぼゼロ（コマンド実行時のみ）

---

### CLI実装例

```go
// cmd/devhive/status.go
var statusCmd = &cobra.Command{
  Use: "status [worker]",
  Run: func(cmd *cobra.Command, args []string) {
    workerID := args[0]

    // オンデマンド取得
    status, err := monitor.GetWorkerStatus(workerID)
    if err != nil {
      log.Fatal(err)
    }

    // 表示
    fmt.Printf("Worker: %s\n", workerID)
    fmt.Printf("Status: %s\n", status.Status)
    fmt.Printf("Progress:\n")
    fmt.Printf("  Commits: %d\n", status.Progress.Commits)
    fmt.Printf("  Changes: +%d -%d\n",
      status.Progress.Insertions,
      status.Progress.Deletions)
    fmt.Printf("Session:\n")
    fmt.Printf("  Errors: %d\n", status.Session.Errors)
    fmt.Printf("  Warnings: %d\n", status.Session.Warnings)
    fmt.Printf("  Last Activity: %s\n",
      status.Session.LastActivity.Format(time.RFC3339))
  },
}

// cmd/devhive/monitor.go
var monitorCmd = &cobra.Command{
  Use: "monitor <worker>",
  Run: func(cmd *cobra.Command, args []string) {
    workerID := args[0]
    interval, _ := cmd.Flags().GetInt("interval")

    // ポーリング表示（CLIプロセス内）
    ticker := time.NewTicker(time.Duration(interval) * time.Second)
    defer ticker.Stop()

    for range ticker.C {
      status, _ := monitor.GetWorkerStatus(workerID)
      displayStatus(status)
    }
  },
}
```

---

### Phase 4: 自動介入（オプション）

```go
// Orchestratorが自動判断（将来機能）
func (o *Orchestrator) AutoIntervene(workerID string) error {
  status, _ := monitor.GetWorkerStatus(workerID)

  // エラー多発検出
  if status.Session.Errors > 5 {
    return o.HintWorker(workerID, "エラーが多発しています。ログを確認してください。")
  }

  // 停滞検出
  if time.Since(status.Session.LastActivity) > 30*time.Minute && status.Progress.Commits == 0 {
    return o.PauseWorker(workerID)
  }

  return nil
}
```

---

## 通信機能の再設計

### 現状の課題

```go
// 現状（気分依存）
devhive request help "詰まった"  // AIが報告を忘れる
devhive report "進捗50%"         // 不正確・主観的
```

### 統合後（Observable Facts）

**案1**: 通信機能を監視結果ベースに自動化

```go
// Orchestratorが自動検出
func (o *Orchestrator) CheckWorkers() {
  for _, workerID := range o.activeWorkers {
    status, _ := monitor.GetWorkerStatus(workerID)

    // 自動介入判断
    if needsHelp(status) {
      o.NotifyHuman(workerID, status)
    }
  }
}

func needsHelp(status *WorkerStatus) bool {
  // エラー多発
  if status.Session.Errors > 5 {
    return true
  }

  // 停滞
  if time.Since(status.Session.LastActivity) > 30*time.Minute {
    return true
  }

  return false
}
```

**案2**: 通信コマンドを残すが、Observable Facts併用

```bash
# 手動通信（残す）
devhive request help "特定の質問"

# 自動検出（追加）
devhive monitor --auto-intervene
# → エラー・停滞を自動検出してPMに通知
```

---

## 実装優先度

### Phase 1: 基盤実装（即座）

1. **監視機能**:
   - `services/monitor.go` 実装
   - ログ解析（エラー・警告カウント）
   - Git進捗推測（commits、diff統計）

2. **CLI拡張**:
   - `devhive status <w>` → 監視情報表示
   - `devhive monitor <w>` → ポーリング表示

**実装コスト**: 1-2日

---

### Phase 2: スコープ制御（中期）

1. **Agent定義拡張**:
   - `.devhive.yaml` に `scope` フィールド追加
   - DB移行（agents テーブル）

2. **Worktree作成時適用**:
   - sparse-checkout（exclude）
   - chmod（write制御）

**実装コスト**: 2-3日

---

### Phase 3: DBマスター移行（長期）

1. **DB統合**:
   - PostgreSQL対応
   - `.devhive.yaml` → DB移行ツール

2. **Web UI（オプション）**:
   - agentmine Web UIとの統合

**実装コスト**: 5-7日

---

### Phase 4: 自動介入（将来）

1. **Observable Facts応用**:
   - 自動介入判断
   - 通知システム

**実装コスト**: 3-5日

---

## 実装チェックリスト

### Phase 1（監視機能）

- [ ] `services/monitor.go` 作成
  - [ ] `AnalyzeSession()` - ログ解析
  - [ ] `EstimateProgress()` - Git進捗推測
  - [ ] `GetWorkerStatus()` - 統合ステータス

- [ ] CLI拡張
  - [ ] `devhive status <w>` - 監視情報表示
  - [ ] `devhive monitor <w> [--interval N]` - ポーリング

- [ ] テスト
  - [ ] ログ解析の正確性
  - [ ] Git進捗推測の精度
  - [ ] パフォーマンス（負荷測定）

### Phase 2（スコープ制御）

- [ ] Agent定義拡張
  - [ ] `.devhive.yaml` スキーマ更新
  - [ ] DB migration（agents.scope追加）

- [ ] Worktree作成時適用
  - [ ] sparse-checkout実装
  - [ ] chmod実装
  - [ ] エラーハンドリング

### Phase 3（DBマスター）

- [ ] PostgreSQL対応
  - [ ] Connection pool
  - [ ] Migration system

- [ ] 移行ツール
  - [ ] `.devhive.yaml` → DB
  - [ ] `.devhive/tasks/` → memories
  - [ ] `.devhive/roles/` → agents

---

## 参考資料

- **agentmine**: `/home/iguchi/work/agentmine/`
  - `docs/02-architecture/architecture.md` - アーキテクチャ
  - `docs/03-core-concepts/scope-control.md` - スコープ制御
  - `docs/07-runtime/worker-lifecycle.md` - Worker実行フロー
  - `docs/01-introduction/devhive-migration.md` - 統合計画

- **DevHive**: `/home/iguchi/work/devhive/`
  - `README.md` - 現状機能
  - `cmd/devhive/` - CLI実装
  - `.devhive.yaml` - 設定仕様

---

## 実装時の注意点

1. **後方互換性**: `.devhive.yaml` は当面サポート継続
2. **段階的移行**: 既存機能を壊さず、徐々に統合
3. **テスト**: 監視機能の精度検証を十分に
4. **ドキュメント**: README更新、Migration Guide作成

---

**Next Action**: Phase 1（監視機能）から着手
