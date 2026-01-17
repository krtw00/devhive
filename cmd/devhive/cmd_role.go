package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func roleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "role",
		Short: "Role management",
	}

	// role create
	createCmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new role",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			description, _ := cmd.Flags().GetString("description")
			roleFile, _ := cmd.Flags().GetString("file")
			roleArgs, _ := cmd.Flags().GetString("args")

			err := database.CreateRole(args[0], description, roleFile, roleArgs)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Role '%s' created\n", args[0])
			return nil
		},
	}
	createCmd.Flags().StringP("description", "d", "", "Role description")
	createCmd.Flags().StringP("file", "f", "", "Role definition file path")
	createCmd.Flags().StringP("args", "a", "", "AI tool arguments (e.g., --model sonnet)")

	// role list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all roles",
		RunE: func(cmd *cobra.Command, args []string) error {
			roles, err := database.GetAllRoles()
			if err != nil {
				return err
			}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			if jsonOutput {
				b, _ := json.MarshalIndent(roles, "", "  ")
				fmt.Println(string(b))
				return nil
			}

			if len(roles) == 0 {
				fmt.Println("No roles defined")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tDESCRIPTION\tFILE")
			fmt.Fprintln(w, "----\t-----------\t----")
			for _, role := range roles {
				desc := role.Description
				if len(desc) > 30 {
					desc = desc[:27] + "..."
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", role.Name, desc, role.RoleFile)
			}
			w.Flush()
			return nil
		},
	}
	listCmd.Flags().Bool("json", false, "Output as JSON")

	// role show
	showCmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Show role details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			role, err := database.GetRole(args[0])
			if err != nil {
				return err
			}
			if role == nil {
				return fmt.Errorf("role not found: %s", args[0])
			}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			if jsonOutput {
				b, _ := json.MarshalIndent(role, "", "  ")
				fmt.Println(string(b))
				return nil
			}

			fmt.Printf("Name: %s\n", role.Name)
			if role.Description != "" {
				fmt.Printf("Description: %s\n", role.Description)
			}
			if role.RoleFile != "" {
				fmt.Printf("File: %s\n", role.RoleFile)
			}
			if role.Args != "" {
				fmt.Printf("Args: %s\n", role.Args)
			}
			fmt.Printf("Created: %s\n", role.CreatedAt.Format("2006-01-02 15:04:05"))
			return nil
		},
	}
	showCmd.Flags().Bool("json", false, "Output as JSON")

	// role update
	updateCmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Update a role",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get current role first
			role, err := database.GetRole(args[0])
			if err != nil {
				return err
			}
			if role == nil {
				return fmt.Errorf("role not found: %s", args[0])
			}

			description := role.Description
			roleFile := role.RoleFile
			roleArgs := role.Args

			if cmd.Flags().Changed("description") {
				description, _ = cmd.Flags().GetString("description")
			}
			if cmd.Flags().Changed("file") {
				roleFile, _ = cmd.Flags().GetString("file")
			}
			if cmd.Flags().Changed("args") {
				roleArgs, _ = cmd.Flags().GetString("args")
			}

			err = database.UpdateRole(args[0], description, roleFile, roleArgs)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Role '%s' updated\n", args[0])
			return nil
		},
	}
	updateCmd.Flags().StringP("description", "d", "", "Role description")
	updateCmd.Flags().StringP("file", "f", "", "Role definition file path")
	updateCmd.Flags().StringP("args", "a", "", "AI tool arguments (e.g., --model sonnet)")

	// role delete
	deleteCmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a role",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := database.DeleteRole(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("✓ Role '%s' deleted\n", args[0])
			return nil
		},
	}

	cmd.AddCommand(createCmd, listCmd, showCmd, updateCmd, deleteCmd)
	return cmd
}
