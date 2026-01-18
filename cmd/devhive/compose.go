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
	Worktree string `yaml:"worktree"` // Override worktree path
	Disabled bool   `yaml:"disabled"` // Skip this worker
}

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
