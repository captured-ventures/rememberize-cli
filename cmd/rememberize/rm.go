package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <id>",
	Short: "Delete a memory",
	Long:  "Delete a memory by ID. Prompts for confirmation unless --force is used.",
	Args:  cobra.ExactArgs(1),
	RunE:  runRm,
}

var rmForce bool

func init() {
	rmCmd.Flags().BoolVar(&rmForce, "force", false, "skip confirmation prompt")
}

func runRm(cmd *cobra.Command, args []string) error {
	id := args[0]

	if !rmForce {
		fmt.Fprintf(os.Stderr, "Delete memory %s? [y/N] ", id)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
	}

	client := NewClient()
	if err := client.DeleteMemory(id); err != nil {
		return err
	}

	if jsonOutput {
		return printJSON(map[string]string{"deleted": id})
	}

	fmt.Fprintf(os.Stdout, "Deleted memory %s\n", id)
	return nil
}
