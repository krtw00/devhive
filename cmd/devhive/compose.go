package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeConfig represents the .devhive.yaml configuration
type ComposeConfig struct {
	Version     string                   `yaml:"version"`
	Project     string                   `yaml:"project"`
	Roles       map[string]ComposeRole   `yaml:"roles"`
	Defaults    ComposeDefaults          `yaml:"defaults"`
	Workers     map[string]ComposeWorker `yaml:"workers"`
	WorkerOrder []string                 `yaml:"-"` // Preserves yaml definition order
}

// ComposeRole represents a role definition in compose config
type ComposeRole struct {
	Description string `yaml:"description"`
	File        string `yaml:"file"`    // External file reference
	Content     string `yaml:"content"` // Inline content
	Extends     string `yaml:"extends"` // Extend builtin role (e.g., @frontend)
	Args        string `yaml:"args"`    // Additional arguments
}

// ComposeDefaults represents default settings
type ComposeDefaults struct {
	CreateWorktree bool              `yaml:"create_worktree"`
	BaseBranch     string            `yaml:"base_branch"`
	Sprint         string            `yaml:"sprint"`           // Default sprint ID
	PromptTemplate string            `yaml:"prompt_template"`  // Custom prompt template for AI tools
	ToolArgs       map[string]string `yaml:"tool_args"`        // Default args per tool (e.g., claude: "--dangerously-skip-permissions")
	AutoPrompt     bool              `yaml:"auto_prompt"`      // Auto-generate initial prompt for AI tools
	GenerateEnvrc  *bool             `yaml:"generate_envrc"`   // Generate .envrc file (default: true)
	DirenvAllow    bool              `yaml:"direnv_allow"`     // Auto-run direnv allow after creating worktree
	AutoComplete   bool              `yaml:"auto_complete"`    // Auto-mark worker as completed when progress reaches 100%
}

// ComposeWorker represents a worker definition in compose config
type ComposeWorker struct {
	Branch   string `yaml:"branch"`
	Role     string `yaml:"role"`
	Task     string `yaml:"task"`
	Tool     string `yaml:"tool"`     // AI tool: claude, codex, gemini, generic (default: generic)
	Command  string `yaml:"command"`  // Command to execute (default: tool name, generic: $SHELL)
	Args     string `yaml:"args"`     // Arguments for the command
	Prompt   string `yaml:"prompt"`   // Initial prompt to pass to AI tool
	Worktree string `yaml:"worktree"` // Override worktree path
	Disabled bool   `yaml:"disabled"` // Skip this worker
}

// SupportedTools lists the supported AI tools
var SupportedTools = []string{"claude", "codex", "gemini", "generic"}

// DefaultComposeFiles are the filenames to search for
var DefaultComposeFiles = []string{
	".devhive.yaml",
	".devhive.yml",
	"devhive.yaml",
	"devhive.yml",
}

// FindComposeFile searches for a compose file in the current directory
func FindComposeFile() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for _, filename := range DefaultComposeFiles {
		path := filepath.Join(cwd, filename)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("compose file not found (tried: %s)", strings.Join(DefaultComposeFiles, ", "))
}

// LoadComposeFile loads and parses a compose configuration file
func LoadComposeFile(path string) (*ComposeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read compose file: %w", err)
	}

	var config ComposeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// Extract worker order from yaml using yaml.Node
	config.WorkerOrder = extractWorkerOrder(data)

	// Set defaults
	if config.Version == "" {
		config.Version = "1"
	}
	if config.Project == "" {
		// Use directory name as project name
		cwd, _ := os.Getwd()
		config.Project = filepath.Base(cwd)
	}

	return &config, nil
}

// extractWorkerOrder parses yaml to extract worker keys in definition order
func extractWorkerOrder(data []byte) []string {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil
	}

	// root.Content[0] is the document node
	if len(root.Content) == 0 {
		return nil
	}
	doc := root.Content[0]

	// Find "workers" key in the document
	for i := 0; i < len(doc.Content)-1; i += 2 {
		keyNode := doc.Content[i]
		if keyNode.Value == "workers" {
			valueNode := doc.Content[i+1]
			if valueNode.Kind == yaml.MappingNode {
				var order []string
				// MappingNode contains key-value pairs alternating
				for j := 0; j < len(valueNode.Content)-1; j += 2 {
					workerKeyNode := valueNode.Content[j]
					order = append(order, workerKeyNode.Value)
				}
				return order
			}
		}
	}
	return nil
}

// GetRoleContent returns the content of a role (from file or inline)
func (r *ComposeRole) GetRoleContent(basePath string) (string, error) {
	if r.Content != "" {
		return r.Content, nil
	}
	if r.File != "" {
		// Resolve relative path
		filePath := r.File
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(basePath, filePath)
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read role file %s: %w", filePath, err)
		}
		return string(data), nil
	}
	return "", nil
}

// GetEffectiveWorkers returns the list of workers to process
// If workerNames is empty, returns all non-disabled workers
func (c *ComposeConfig) GetEffectiveWorkers(workerNames []string) map[string]ComposeWorker {
	result := make(map[string]ComposeWorker)

	if len(workerNames) == 0 {
		// Return all non-disabled workers
		for name, worker := range c.Workers {
			if !worker.Disabled {
				result[name] = worker
			}
		}
	} else {
		// Return only specified workers
		for _, name := range workerNames {
			if worker, ok := c.Workers[name]; ok {
				result[name] = worker
			}
		}
	}

	return result
}

// GetOrderedWorkerNames returns worker names in yaml definition order
// If specific names are provided, returns them in the order they appear in yaml
// Falls back to alphabetical order if WorkerOrder is not available
func (c *ComposeConfig) GetOrderedWorkerNames(filterNames []string) []string {
	workers := c.GetEffectiveWorkers(filterNames)
	if len(workers) == 0 {
		return nil
	}

	// If WorkerOrder is available, use it
	if len(c.WorkerOrder) > 0 {
		var ordered []string
		for _, name := range c.WorkerOrder {
			if _, ok := workers[name]; ok {
				ordered = append(ordered, name)
			}
		}
		// Add any workers not in WorkerOrder (shouldn't happen normally)
		for name := range workers {
			found := false
			for _, n := range ordered {
				if n == name {
					found = true
					break
				}
			}
			if !found {
				ordered = append(ordered, name)
			}
		}
		return ordered
	}

	// Fallback to alphabetical order
	names := make([]string, 0, len(workers))
	for name := range workers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ResolveRole resolves a role name to its actual name
// Handles @builtin prefix and role definitions
func (c *ComposeConfig) ResolveRole(roleName string) string {
	if roleName == "" {
		return ""
	}

	// Check if it's a builtin role reference
	if strings.HasPrefix(roleName, "@") {
		return strings.TrimPrefix(roleName, "@")
	}

	// Check if it's defined in config
	if role, ok := c.Roles[roleName]; ok {
		// If it extends a builtin, use the builtin name
		if role.Extends != "" && strings.HasPrefix(role.Extends, "@") {
			return strings.TrimPrefix(role.Extends, "@")
		}
		return roleName
	}

	return roleName
}

// GenerateSprintID generates a sprint ID based on config or timestamp
func (c *ComposeConfig) GenerateSprintID() string {
	if c.Defaults.Sprint != "" {
		return c.Defaults.Sprint
	}
	// Use a simple incrementing name
	return "sprint"
}

// GetTaskContent returns the task content for a worker
// Priority: 1. inline task as file path, 2. .devhive/tasks/<name>.md, 3. inline task as content
func GetTaskContent(projectRoot, workerName, inlineTask string) string {
	// 1. If inline task looks like a file path, read it directly
	if inlineTask != "" && (strings.HasSuffix(inlineTask, ".md") || strings.Contains(inlineTask, "/")) {
		filePath := inlineTask
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(projectRoot, filePath)
		}
		if data, err := os.ReadFile(filePath); err == nil {
			return string(data)
		}
	}

	// 2. Try to read from .devhive/tasks/<worker>.md
	taskFile := filepath.Join(projectRoot, ".devhive", "tasks", workerName+".md")
	if data, err := os.ReadFile(taskFile); err == nil {
		return string(data)
	}

	// 3. Fall back to inline task as content
	return inlineTask
}

// GetRoleFile returns the role file path for a worker
// Returns .devhive/roles/<role>.md if exists, otherwise empty
func GetRoleFile(projectRoot, roleName string) string {
	// Skip builtin roles
	if strings.HasPrefix(roleName, "@") {
		return ""
	}

	roleFile := filepath.Join(projectRoot, ".devhive", "roles", roleName+".md")
	if _, err := os.Stat(roleFile); err == nil {
		return roleFile
	}
	return ""
}

// GetEffectiveTool returns the tool name, defaulting to "generic"
func (w *ComposeWorker) GetEffectiveTool() string {
	if w.Tool == "" {
		return "generic"
	}
	return w.Tool
}

// GetEffectiveCommand returns the command to execute
// If command is set, use it; otherwise use tool name; generic defaults to $SHELL
func (w *ComposeWorker) GetEffectiveCommand() string {
	if w.Command != "" {
		return w.Command
	}
	tool := w.GetEffectiveTool()
	if tool == "generic" {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
		return shell
	}
	return tool
}

// GetCommandWithArgs returns the full command string with arguments
func (w *ComposeWorker) GetCommandWithArgs() string {
	cmd := w.GetEffectiveCommand()
	if w.Args != "" {
		return cmd + " " + w.Args
	}
	return cmd
}

// GetFullCommand returns the complete command with default args and prompt
func (w *ComposeWorker) GetFullCommand(workerName string, config *ComposeConfig, projectRoot string) string {
	cmd := w.GetEffectiveCommand()
	tool := w.GetEffectiveTool()

	// Build args: default tool_args + worker-specific args
	var argParts []string

	// Add default tool args if defined
	if config.Defaults.ToolArgs != nil {
		if defaultArgs, ok := config.Defaults.ToolArgs[tool]; ok && defaultArgs != "" {
			argParts = append(argParts, defaultArgs)
		}
	}

	// Add worker-specific args
	if w.Args != "" {
		argParts = append(argParts, w.Args)
	}

	// Build the prompt
	prompt := w.getEffectivePrompt(workerName, config, projectRoot)

	// Combine command + args + prompt
	if len(argParts) > 0 {
		cmd = cmd + " " + strings.Join(argParts, " ")
	}

	if prompt != "" {
		// Escape double quotes in prompt and wrap in quotes
		escapedPrompt := strings.ReplaceAll(prompt, `"`, `\"`)
		escapedPrompt = strings.ReplaceAll(escapedPrompt, `$`, `\$`)
		escapedPrompt = strings.ReplaceAll(escapedPrompt, "`", "\\`")
		cmd = cmd + ` "` + escapedPrompt + `"`
	}

	return cmd
}

// getEffectivePrompt returns the prompt to use for the AI tool
func (w *ComposeWorker) getEffectivePrompt(workerName string, config *ComposeConfig, projectRoot string) string {
	// 1. Worker-specific prompt takes precedence
	if w.Prompt != "" {
		return w.Prompt
	}

	// 2. Auto-prompt if enabled
	if config.Defaults.AutoPrompt {
		tool := w.GetEffectiveTool()
		switch tool {
		case "claude":
			return fmt.Sprintf("CLAUDE.mdを読んでタスクを実行してください。進捗は devhive progress %s <0-100> で報告してください。", workerName)
		case "codex":
			return fmt.Sprintf("AGENTS.mdを読んでタスクを実行してください。進捗は devhive progress %s <0-100> で報告してください。", workerName)
		case "gemini":
			return fmt.Sprintf("GEMINI.mdを読んでタスクを実行してください。進捗は devhive progress %s <0-100> で報告してください。", workerName)
		}
	}

	return ""
}

// TemplateVars holds variables for prompt template rendering
type TemplateVars struct {
	WorkerName  string
	Branch      string
	Role        string
	Tool        string
	Project     string
	BaseBranch  string
	TaskContent string
	RoleContent string
}

// RenderPromptTemplate renders a prompt template with the given variables
func RenderPromptTemplate(template string, vars TemplateVars) string {
	result := template
	result = strings.ReplaceAll(result, "{{worker_name}}", vars.WorkerName)
	result = strings.ReplaceAll(result, "{{branch}}", vars.Branch)
	result = strings.ReplaceAll(result, "{{role}}", vars.Role)
	result = strings.ReplaceAll(result, "{{tool}}", vars.Tool)
	result = strings.ReplaceAll(result, "{{project}}", vars.Project)
	result = strings.ReplaceAll(result, "{{base_branch}}", vars.BaseBranch)
	result = strings.ReplaceAll(result, "{{task_content}}", vars.TaskContent)
	result = strings.ReplaceAll(result, "{{role_content}}", vars.RoleContent)
	return result
}

// GetPromptTemplate returns the prompt template content
// If prompt_template is a file path, reads the file; otherwise returns the string as-is
// If empty, returns the default template
func GetPromptTemplate(promptTemplate, projectRoot string) string {
	if promptTemplate == "" {
		return GetDefaultPromptTemplate()
	}

	// If it looks like a file path, read the file
	if strings.HasSuffix(promptTemplate, ".md") || strings.Contains(promptTemplate, "/") {
		filePath := promptTemplate
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(projectRoot, filePath)
		}
		if data, err := os.ReadFile(filePath); err == nil {
			return string(data)
		}
	}

	// Otherwise return as-is (inline template)
	return promptTemplate
}

// GetDefaultPromptTemplate returns the default prompt template
func GetDefaultPromptTemplate() string {
	return `あなたは **{{worker_name}}** ワーカーとして作業しています。

## プロジェクト
- プロジェクト: {{project}}
- ブランチ: {{branch}}
- ベースブランチ: {{base_branch}}

## ロール
{{role_content}}

## タスク
{{task_content}}

## 実行ルール
1. 上記タスクを順番に実行してください
2. 進捗に応じて ` + "`devhive progress {{worker_name}} <0-100>`" + ` を実行
3. 完了したら ` + "`devhive progress {{worker_name}} 100`" + ` を実行
4. コミットメッセージは Conventional Commits 形式で
5. 問題発生時は ` + "`devhive request help \"内容\"`" + ` でPMに連絡
6. レビュー準備完了時は ` + "`devhive request review \"内容\"`" + `

## 利用可能なコマンド
- ` + "`devhive progress {{worker_name}} <0-100>`" + ` - 進捗更新
- ` + "`devhive request help \"質問\"`" + ` - ヘルプ要求
- ` + "`devhive request review \"内容\"`" + ` - レビュー依頼
- ` + "`devhive request unblock \"理由\"`" + ` - ブロック解除
- ` + "`devhive report \"進捗報告\"`" + ` - 進捗報告
- ` + "`devhive msgs`" + ` - メッセージ確認
`
}

// GenerateContextFiles generates context files for a worker in the worktree
func GenerateContextFiles(worktreePath, workerName string, worker ComposeWorker, config *ComposeConfig, projectRoot string) error {
	// Always generate CONTEXT.md (generic)
	contextContent := generateContextContent(workerName, worker, config, projectRoot)
	contextPath := filepath.Join(worktreePath, "CONTEXT.md")
	if err := os.WriteFile(contextPath, []byte(contextContent), 0644); err != nil {
		return fmt.Errorf("failed to write CONTEXT.md: %w", err)
	}

	// Generate tool-specific file if needed
	tool := worker.GetEffectiveTool()
	switch tool {
	case "claude":
		claudeContent := generateClaudeContent(workerName, worker, config, projectRoot)
		claudePath := filepath.Join(worktreePath, "CLAUDE.md")
		if err := os.WriteFile(claudePath, []byte(claudeContent), 0644); err != nil {
			return fmt.Errorf("failed to write CLAUDE.md: %w", err)
		}
	case "codex":
		codexContent := generateCodexContent(workerName, worker, config, projectRoot)
		codexPath := filepath.Join(worktreePath, "AGENTS.md")
		if err := os.WriteFile(codexPath, []byte(codexContent), 0644); err != nil {
			return fmt.Errorf("failed to write AGENTS.md: %w", err)
		}
	case "gemini":
		geminiContent := generateGeminiContent(workerName, worker, config, projectRoot)
		geminiPath := filepath.Join(worktreePath, "GEMINI.md")
		if err := os.WriteFile(geminiPath, []byte(geminiContent), 0644); err != nil {
			return fmt.Errorf("failed to write GEMINI.md: %w", err)
		}
	}

	return nil
}

// generateContextContent creates the generic CONTEXT.md content
func generateContextContent(workerName string, worker ComposeWorker, config *ComposeConfig, projectRoot string) string {
	var sb strings.Builder

	sb.WriteString("# DevHive Worker Context\n\n")
	sb.WriteString("> このファイルは自動生成されています。編集しないでください。\n\n")

	// Worker info
	sb.WriteString("## Worker\n\n")
	sb.WriteString(fmt.Sprintf("- **Name**: %s\n", workerName))
	sb.WriteString(fmt.Sprintf("- **Branch**: %s\n", worker.Branch))
	sb.WriteString(fmt.Sprintf("- **Role**: %s\n", worker.Role))
	sb.WriteString(fmt.Sprintf("- **Tool**: %s\n", worker.GetEffectiveTool()))
	sb.WriteString("\n")

	// Project info
	sb.WriteString("## Project\n\n")
	sb.WriteString(fmt.Sprintf("- **Name**: %s\n", config.Project))
	sb.WriteString(fmt.Sprintf("- **Base Branch**: %s\n", config.Defaults.BaseBranch))
	sb.WriteString("\n")

	// Role description
	sb.WriteString("## Role\n\n")
	roleContent := getRoleContent(worker.Role, config, projectRoot)
	if roleContent != "" {
		sb.WriteString(roleContent)
	} else {
		sb.WriteString(fmt.Sprintf("Role: %s\n", worker.Role))
	}
	sb.WriteString("\n")

	// Task
	sb.WriteString("## Task\n\n")
	taskContent := GetTaskContent(projectRoot, workerName, worker.Task)
	if taskContent != "" {
		sb.WriteString(taskContent)
	} else {
		sb.WriteString("タスクが定義されていません。\n")
	}
	sb.WriteString("\n")

	// Communication
	sb.WriteString("## Communication\n\n")
	sb.WriteString("PMとの通信には以下のコマンドを使用:\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("devhive request help \"質問内容\"     # ヘルプ要求\n")
	sb.WriteString("devhive request review \"内容\"      # レビュー依頼\n")
	sb.WriteString("devhive request unblock \"理由\"     # ブロック解除\n")
	sb.WriteString("devhive report \"進捗報告\"          # 進捗報告\n")
	sb.WriteString("devhive progress 50                 # 進捗更新 (0-100)\n")
	sb.WriteString("devhive msgs                        # メッセージ確認\n")
	sb.WriteString("```\n")

	return sb.String()
}

// generateClaudeContent creates Claude-specific CLAUDE.md
func generateClaudeContent(workerName string, worker ComposeWorker, config *ComposeConfig, projectRoot string) string {
	roleContent := getRoleContent(worker.Role, config, projectRoot)
	taskContent := GetTaskContent(projectRoot, workerName, worker.Task)

	// Use custom template (supports file path or inline)
	template := GetPromptTemplate(config.Defaults.PromptTemplate, projectRoot)

	vars := TemplateVars{
		WorkerName:  workerName,
		Branch:      worker.Branch,
		Role:        worker.Role,
		Tool:        worker.GetEffectiveTool(),
		Project:     config.Project,
		BaseBranch:  config.Defaults.BaseBranch,
		TaskContent: taskContent,
		RoleContent: roleContent,
	}

	var sb strings.Builder
	sb.WriteString("# Claude Code Instructions\n\n")
	sb.WriteString(RenderPromptTemplate(template, vars))

	return sb.String()
}

// generateCodexContent creates Codex-specific AGENTS.md
func generateCodexContent(workerName string, worker ComposeWorker, config *ComposeConfig, projectRoot string) string {
	roleContent := getRoleContent(worker.Role, config, projectRoot)
	taskContent := GetTaskContent(projectRoot, workerName, worker.Task)

	// Use custom template (supports file path or inline)
	template := GetPromptTemplate(config.Defaults.PromptTemplate, projectRoot)

	vars := TemplateVars{
		WorkerName:  workerName,
		Branch:      worker.Branch,
		Role:        worker.Role,
		Tool:        worker.GetEffectiveTool(),
		Project:     config.Project,
		BaseBranch:  config.Defaults.BaseBranch,
		TaskContent: taskContent,
		RoleContent: roleContent,
	}

	var sb strings.Builder
	sb.WriteString("# Codex Agent Instructions\n\n")
	sb.WriteString(RenderPromptTemplate(template, vars))

	return sb.String()
}

// generateGeminiContent creates Gemini-specific GEMINI.md
func generateGeminiContent(workerName string, worker ComposeWorker, config *ComposeConfig, projectRoot string) string {
	roleContent := getRoleContent(worker.Role, config, projectRoot)
	taskContent := GetTaskContent(projectRoot, workerName, worker.Task)

	// Use custom template (supports file path or inline)
	template := GetPromptTemplate(config.Defaults.PromptTemplate, projectRoot)

	vars := TemplateVars{
		WorkerName:  workerName,
		Branch:      worker.Branch,
		Role:        worker.Role,
		Tool:        worker.GetEffectiveTool(),
		Project:     config.Project,
		BaseBranch:  config.Defaults.BaseBranch,
		TaskContent: taskContent,
		RoleContent: roleContent,
	}

	var sb strings.Builder
	sb.WriteString("# Gemini Instructions\n\n")
	sb.WriteString(RenderPromptTemplate(template, vars))

	return sb.String()
}

// getRoleContent returns role content from config or .devhive/roles/<name>.md
func getRoleContent(roleName string, config *ComposeConfig, projectRoot string) string {
	if roleName == "" {
		return ""
	}

	// 1. If role looks like a file path, read it directly
	if strings.HasSuffix(roleName, ".md") || strings.Contains(roleName, "/") {
		// Resolve relative path
		filePath := roleName
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(projectRoot, filePath)
		}
		if data, err := os.ReadFile(filePath); err == nil {
			return string(data)
		}
	}

	// Strip @ prefix if present
	cleanName := strings.TrimPrefix(roleName, "@")

	// 2. Check if role is defined in config
	if role, ok := config.Roles[cleanName]; ok {
		content, err := role.GetRoleContent(projectRoot)
		if err == nil && content != "" {
			return content
		}
		if role.Description != "" {
			return role.Description
		}
	}

	// 3. Check .devhive/roles/<name>.md
	roleFile := filepath.Join(projectRoot, ".devhive", "roles", cleanName+".md")
	if data, err := os.ReadFile(roleFile); err == nil {
		return string(data)
	}

	return ""
}
