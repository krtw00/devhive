package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeConfig represents the .devhive.yaml configuration
type ComposeConfig struct {
	Version  string                  `yaml:"version"`
	Project  string                  `yaml:"project"`
	Roles    map[string]ComposeRole  `yaml:"roles"`
	Defaults ComposeDefaults         `yaml:"defaults"`
	Workers  map[string]ComposeWorker `yaml:"workers"`
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
	CreateWorktree bool   `yaml:"create_worktree"`
	BaseBranch     string `yaml:"base_branch"`
	Sprint         string `yaml:"sprint"` // Default sprint ID
}

// ComposeWorker represents a worker definition in compose config
type ComposeWorker struct {
	Branch   string `yaml:"branch"`
	Role     string `yaml:"role"`
	Task     string `yaml:"task"`
	Tool     string `yaml:"tool"`     // AI tool: claude, codex, gemini, generic (default: generic)
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
// Checks .devhive/tasks/<name>.md first, then inline task field
func GetTaskContent(projectRoot, workerName, inlineTask string) string {
	// Try to read from .devhive/tasks/<worker>.md
	taskFile := filepath.Join(projectRoot, ".devhive", "tasks", workerName+".md")
	if data, err := os.ReadFile(taskFile); err == nil {
		return string(data)
	}

	// Fall back to inline task
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
	var sb strings.Builder

	sb.WriteString("# Claude Code Instructions\n\n")
	sb.WriteString(fmt.Sprintf("あなたは **%s** ワーカーとして作業しています。\n\n", workerName))

	// Include CONTEXT.md reference
	sb.WriteString("## Context\n\n")
	sb.WriteString("詳細なコンテキストは [CONTEXT.md](./CONTEXT.md) を参照してください。\n\n")

	// Role
	sb.WriteString("## Role\n\n")
	roleContent := getRoleContent(worker.Role, config, projectRoot)
	if roleContent != "" {
		sb.WriteString(roleContent)
	}
	sb.WriteString("\n")

	// Task
	sb.WriteString("## Task\n\n")
	taskContent := GetTaskContent(projectRoot, workerName, worker.Task)
	if taskContent != "" {
		sb.WriteString(taskContent)
	}
	sb.WriteString("\n")

	// Instructions
	sb.WriteString("## Instructions\n\n")
	sb.WriteString("- タスク完了時は `devhive progress 100` で進捗を更新\n")
	sb.WriteString("- 問題発生時は `devhive request help \"内容\"` でPMに連絡\n")
	sb.WriteString("- コードレビュー準備完了時は `devhive request review \"内容\"`\n")
	sb.WriteString("- ブロックされた場合は `devhive request unblock \"理由\"`\n")

	return sb.String()
}

// generateCodexContent creates Codex-specific AGENTS.md
func generateCodexContent(workerName string, worker ComposeWorker, config *ComposeConfig, projectRoot string) string {
	var sb strings.Builder

	sb.WriteString("# Codex Agent Instructions\n\n")
	sb.WriteString(fmt.Sprintf("Worker: %s\n", workerName))
	sb.WriteString(fmt.Sprintf("Branch: %s\n", worker.Branch))
	sb.WriteString(fmt.Sprintf("Role: %s\n\n", worker.Role))

	sb.WriteString("## Task\n\n")
	taskContent := GetTaskContent(projectRoot, workerName, worker.Task)
	if taskContent != "" {
		sb.WriteString(taskContent)
	}
	sb.WriteString("\n")

	sb.WriteString("## Context\n\n")
	sb.WriteString("See CONTEXT.md for full context.\n")

	return sb.String()
}

// generateGeminiContent creates Gemini-specific GEMINI.md
func generateGeminiContent(workerName string, worker ComposeWorker, config *ComposeConfig, projectRoot string) string {
	var sb strings.Builder

	sb.WriteString("# Gemini Instructions\n\n")
	sb.WriteString(fmt.Sprintf("Worker: %s\n", workerName))
	sb.WriteString(fmt.Sprintf("Branch: %s\n", worker.Branch))
	sb.WriteString(fmt.Sprintf("Role: %s\n\n", worker.Role))

	sb.WriteString("## Task\n\n")
	taskContent := GetTaskContent(projectRoot, workerName, worker.Task)
	if taskContent != "" {
		sb.WriteString(taskContent)
	}
	sb.WriteString("\n")

	sb.WriteString("## Context\n\n")
	sb.WriteString("See CONTEXT.md for full context.\n")

	return sb.String()
}

// getRoleContent returns role content from config or .devhive/roles/<name>.md
func getRoleContent(roleName string, config *ComposeConfig, projectRoot string) string {
	// Strip @ prefix if present
	cleanName := strings.TrimPrefix(roleName, "@")

	// 1. Check if role is defined in config
	if role, ok := config.Roles[cleanName]; ok {
		content, err := role.GetRoleContent(projectRoot)
		if err == nil && content != "" {
			return content
		}
		if role.Description != "" {
			return role.Description
		}
	}

	// 2. Check .devhive/roles/<name>.md
	roleFile := filepath.Join(projectRoot, ".devhive", "roles", cleanName+".md")
	if data, err := os.ReadFile(roleFile); err == nil {
		return string(data)
	}

	return ""
}
