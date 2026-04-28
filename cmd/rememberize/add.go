package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <content>",
	Short: "Create a new memory",
	Long:  "Create a new memory. Pass content as an argument, or use \"-\" to read from stdin.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runAdd,
}

var (
	addNamespace string
	addType      string
	addMeta      string
	addExpires   string
)

func init() {
	addCmd.Flags().StringVar(&addNamespace, "ns", "", "namespace (default: from config)")
	addCmd.Flags().StringVar(&addType, "type", "", "memory type: semantic, episodic, procedural (default: from config)")
	addCmd.Flags().StringVar(&addMeta, "meta", "", "metadata as JSON string")
	addCmd.Flags().StringVar(&addExpires, "expires", "", "expiration time (RFC3339)")
}

func runAdd(cmd *cobra.Command, args []string) error {
	cfg := loadConfig()
	client := NewClient()

	var content string
	if args[0] == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		content = strings.TrimSpace(string(data))
	} else {
		content = strings.Join(args, " ")
	}

	if content == "" {
		return fmt.Errorf("content cannot be empty")
	}

	ns := addNamespace
	if ns == "" {
		ns = cfg.Defaults.Namespace
	}
	mt := addType
	if mt == "" {
		mt = cfg.Defaults.Type
	}

	var metaPtr *string
	if addMeta != "" {
		metaPtr = &addMeta
	}
	var expiresPtr *string
	if addExpires != "" {
		expiresPtr = &addExpires
	}

	mem, err := client.CreateMemory(content, ns, mt, metaPtr, expiresPtr)
	if err != nil {
		return err
	}

	if jsonOutput {
		return printJSON(mem)
	}

	fmt.Fprintf(os.Stdout, "Created memory %s\n", mem.ID)
	fmt.Fprintf(os.Stdout, "  namespace: %s\n", mem.Namespace)
	fmt.Fprintf(os.Stdout, "  type:      %s\n", mem.Type)
	fmt.Fprintf(os.Stdout, "  content:   %s\n", truncate(mem.Content, 80))
	return nil
}
