package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/iguchi/devhive/internal/db"
	"github.com/spf13/cobra"
)

func statusCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current status",
		RunE: func(cmd *cobra.Command, args []string) error {
			sprint, err := database.GetActiveSprint()
			if err != nil {
				return err
			}
			if sprint == nil {
				if jsonOutput {
					fmt.Println("{}")
				} else {
					fmt.Println("No active sprint")
				}
				return nil
			}

			workers, err := database.GetAllWorkers()
			if err != nil {
				return err
			}

			if jsonOutput {
				output := map[string]interface{}{
					"project": db.GetProjectName(),
					"sprint":  sprint,
					"workers": workers,
				}
				b, _ := json.MarshalIndent(output, "", "  ")
				fmt.Println(string(b))
				return nil
			}

			// Show project name if set
			if project := db.GetProjectName(); project != "" {
				fmt.Printf("Project: %s\n", project)
			}
			fmt.Printf("Sprint: %s (started: %s)\n\n", sprint.ID, sprint.StartedAt.Format("2006-01-02 15:04"))

			if len(workers) == 0 {
				fmt.Println("No workers registered")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "WORKER\tROLE\tBRANCH\tSTATUS\tSESSION\tTASK\tMSGS")
			fmt.Fprintln(w, "------\t----\t------\t------\t-------\t----\t----")
			for _, worker := range workers {
				task := worker.CurrentTask
				if len(task) > 20 {
					task = task[:17] + "..."
				}
				sessionStr := fmt.Sprintf("%s %s", sessionIcon(worker.SessionState), worker.SessionState)
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%d\n",
					worker.Name, worker.RoleName, worker.Branch, statusIcon(worker.Status),
					sessionStr, task, worker.UnreadMessages)
			}
			w.Flush()

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

// ProjectSummary holds aggregated project info
type ProjectSummary struct {
	Name           string
	SprintID       string
	SprintStatus   string
	WorkerCount    int
	ActiveWorkers  int
	WaitingWorkers int
	Workers        []WorkerSummary
}

// WorkerSummary holds minimal worker info for project listing
type WorkerSummary struct {
	Name         string
	SessionState string
}

func projectsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List all projects and their status",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, _ := os.UserHomeDir()
			projectsDir := filepath.Join(home, ".devhive", "projects")

			// Check if projects directory exists
			if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
				fmt.Println("No projects found")
				return nil
			}

			// Scan for project directories
			entries, err := os.ReadDir(projectsDir)
			if err != nil {
				return err
			}

			var summaries []ProjectSummary
			var attentionNeeded []string

			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}

				projectName := entry.Name()
				dbPath := filepath.Join(projectsDir, projectName, "state.db")

				// Skip if no state.db
				if _, err := os.Stat(dbPath); os.IsNotExist(err) {
					continue
				}

				// Open DB read-only
				projectDB, err := db.Open(dbPath)
				if err != nil {
					continue
				}

				summary := ProjectSummary{Name: projectName}

				// Get sprint info
				sprint, _ := projectDB.GetActiveSprint()
				if sprint != nil {
					summary.SprintID = sprint.ID
					summary.SprintStatus = sprint.Status

					// Get workers
					workers, _ := projectDB.GetAllWorkers()
					summary.WorkerCount = len(workers)

					for _, w := range workers {
						ws := WorkerSummary{Name: w.Name, SessionState: w.SessionState}
						summary.Workers = append(summary.Workers, ws)

						if w.SessionState == "running" {
							summary.ActiveWorkers++
						}
						if w.SessionState == "waiting_permission" {
							summary.WaitingWorkers++
							attentionNeeded = append(attentionNeeded,
								fmt.Sprintf("%s/%s: waiting_permission", projectName, w.Name))
						}
					}
				}

				projectDB.Close()
				summaries = append(summaries, summary)
			}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			if jsonOutput {
				b, _ := json.MarshalIndent(summaries, "", "  ")
				fmt.Println(string(b))
				return nil
			}

			if len(summaries) == 0 {
				fmt.Println("No projects found")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "PROJECT\tSPRINT\tSTATUS\tWORKERS")
			fmt.Fprintln(w, "-------\t------\t------\t-------")

			for _, s := range summaries {
				sprintID := s.SprintID
				if sprintID == "" {
					sprintID = "-"
				}
				status := s.SprintStatus
				if status == "" {
					status = "-"
				}

				// Build workers string
				var workerStrs []string
				for _, ws := range s.Workers {
					workerStrs = append(workerStrs, fmt.Sprintf("%s[%s]", ws.Name, sessionIcon(ws.SessionState)))
				}
				workersStr := strings.Join(workerStrs, " ")
				if workersStr == "" {
					workersStr = "-"
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, sprintID, status, workersStr)
			}
			w.Flush()

			// Show attention needed
			if len(attentionNeeded) > 0 {
				fmt.Println()
				fmt.Println("⚠ Attention needed:")
				for _, item := range attentionNeeded {
					fmt.Printf("  %s\n", item)
				}
			}

			return nil
		},
	}

	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}
