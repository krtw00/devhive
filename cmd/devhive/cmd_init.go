package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/iguchi/devhive/internal/db"
	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [project-name]",
		Short: "Initialize DevHive in current directory",
		Long: `Initialize DevHive project structure:
  - Creates .devhive/ directory with subdirectories
  - Initializes SQLite database
  - Optionally creates template .devhive.yaml

Examples:
  devhive init              # Auto-detect project name from directory
  devhive init myproject    # Specify project name
  devhive init --template   # Also create template .devhive.yaml`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			// Determine project name
			projectName := filepath.Base(cwd)
			if len(args) > 0 {
				projectName = args[0]
			}

			// Create .devhive directory structure
			devhiveDir := filepath.Join(cwd, ".devhive")
			subdirs := []string{
				"worktrees",
				"roles",
				"tasks",
				"workers",
			}

			fmt.Printf("Initializing DevHive for project: %s\n", projectName)

			// Create main directory
			if err := os.MkdirAll(devhiveDir, 0755); err != nil {
				return fmt.Errorf("failed to create .devhive directory: %w", err)
			}
			fmt.Println("  Created .devhive/")

			// Create subdirectories
			for _, subdir := range subdirs {
				path := filepath.Join(devhiveDir, subdir)
				if err := os.MkdirAll(path, 0755); err != nil {
					return fmt.Errorf("failed to create %s directory: %w", subdir, err)
				}
				fmt.Printf("  Created .devhive/%s/\n", subdir)
			}

			// Initialize database
			db.ProjectName = projectName
			database, err := db.Open("")
			if err != nil {
				return fmt.Errorf("failed to initialize database: %w", err)
			}
			defer database.Close()
			fmt.Println("  Initialized devhive.db")

			// Create template .devhive.yaml if requested
			createTemplate, _ := cmd.Flags().GetBool("template")
			if createTemplate {
				yamlPath := filepath.Join(cwd, ".devhive.yaml")
				if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
					if err := createTemplateYaml(yamlPath, projectName); err != nil {
						return fmt.Errorf("failed to create template: %w", err)
					}
					fmt.Println("  Created .devhive.yaml")
				} else {
					fmt.Println("  Skipped .devhive.yaml (already exists)")
				}
			}

			// Add .devhive/ to .gitignore if not already
			gitignorePath := filepath.Join(cwd, ".gitignore")
			if err := ensureGitignore(gitignorePath); err != nil {
				fmt.Printf("  Warning: could not update .gitignore: %v\n", err)
			} else {
				fmt.Println("  Updated .gitignore")
			}

			fmt.Println("\nDone! Next steps:")
			if !createTemplate {
				fmt.Println("  1. Create .devhive.yaml with your worker definitions")
				fmt.Println("  2. Run 'devhive up' to start workers")
			} else {
				fmt.Println("  1. Edit .devhive.yaml to define your workers")
				fmt.Println("  2. Run 'devhive up' to start workers")
			}

			return nil
		},
	}

	cmd.Flags().BoolP("template", "t", false, "Create template .devhive.yaml")

	return cmd
}

func createTemplateYaml(path, projectName string) error {
	template := fmt.Sprintf(`# DevHive Configuration
version: "1"
project: %s

# Default settings
defaults:
  base_branch: main

# Worker definitions
workers:
  frontend:
    branch: feat/frontend
    role: "@frontend"
    task: フロントエンド実装

  backend:
    branch: feat/backend
    role: "@backend"
    task: バックエンド実装

# Custom roles (optional)
# roles:
#   custom-role:
#     description: "Custom role description"
#     file: .devhive/roles/custom.md
`, projectName)

	return os.WriteFile(path, []byte(template), 0644)
}

func ensureGitignore(path string) error {
	entry := ".devhive/"

	// Read existing content
	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Check if already contains entry
	if contains(string(content), entry) {
		return nil
	}

	// Append entry
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Add newline if file doesn't end with one
	if len(content) > 0 && content[len(content)-1] != '\n' {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}

	_, err = f.WriteString(entry + "\n")
	return err
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsLine(s, substr))
}

func containsLine(s, line string) bool {
	lines := splitLines(s)
	for _, l := range lines {
		if l == line {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
