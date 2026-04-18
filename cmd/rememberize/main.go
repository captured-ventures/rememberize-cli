package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/captured-ventures/rememberize-cli/internal/transfer"
	"github.com/spf13/cobra"
)

// version is the CLI version string. Defaults to "dev" for local builds;
// release builds inject the tag via -ldflags "-X main.version=v0.1.0".
var version = "dev"

// Global flags
var jsonOutput bool

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// ---------------------------------------------------------------------------
// Root command
// ---------------------------------------------------------------------------

var rootCmd = &cobra.Command{
	Use:           "rememberize",
	Short:         "CLI client for the rememberize memory system",
	Long:          "A portable, multi-directional memory system for AI.\nManage memories, search, and configure connections from the command line.",
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output raw JSON (for scripting/piping)")

	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(recallCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(rmCmd)
	rootCmd.AddCommand(namespacesCmd)
	rootCmd.AddCommand(connectionsCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(keysCmd)
	rootCmd.AddCommand(pairCmd)
}

// ---------------------------------------------------------------------------
// add command
// ---------------------------------------------------------------------------

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

	// Determine content.
	var content string
	if args[0] == "-" {
		// Read from stdin.
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

	// Apply defaults from config.
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

// ---------------------------------------------------------------------------
// recall command (semantic search)
// ---------------------------------------------------------------------------

var recallCmd = &cobra.Command{
	Use:   "recall <query>",
	Short: "Semantic search for memories",
	Long:  "Search memories using vector/semantic similarity.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runRecall,
}

var (
	recallNamespace string
	recallLimit     int
)

func init() {
	recallCmd.Flags().StringVar(&recallNamespace, "ns", "", "filter by namespace")
	recallCmd.Flags().IntVar(&recallLimit, "limit", 10, "max results to return")
}

func runRecall(cmd *cobra.Command, args []string) error {
	client := NewClient()
	query := strings.Join(args, " ")

	results, err := client.SearchMemories(query, recallNamespace, "", recallLimit, true)
	if err != nil {
		return err
	}

	if jsonOutput {
		return printJSON(results)
	}

	if len(results) == 0 {
		fmt.Fprintln(os.Stdout, "No results found.")
		return nil
	}

	printSearchResults(results)
	return nil
}

// ---------------------------------------------------------------------------
// search command (FTS)
// ---------------------------------------------------------------------------

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Full-text search for memories",
	Long:  "Search memories using full-text search.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearch,
}

var (
	searchNamespace string
	searchType      string
	searchLimit     int
)

func init() {
	searchCmd.Flags().StringVar(&searchNamespace, "ns", "", "filter by namespace")
	searchCmd.Flags().StringVar(&searchType, "type", "", "filter by type")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 10, "max results to return")
}

func runSearch(cmd *cobra.Command, args []string) error {
	client := NewClient()
	query := strings.Join(args, " ")

	results, err := client.SearchMemories(query, searchNamespace, searchType, searchLimit, false)
	if err != nil {
		return err
	}

	if jsonOutput {
		return printJSON(results)
	}

	if len(results) == 0 {
		fmt.Fprintln(os.Stdout, "No results found.")
		return nil
	}

	printSearchResults(results)
	return nil
}

// ---------------------------------------------------------------------------
// list command
// ---------------------------------------------------------------------------

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List memories",
	Long:  "List memories with optional namespace and type filters.",
	RunE:  runList,
}

var (
	listNamespace string
	listType      string
	listLimit     int
	listOffset    int
)

func init() {
	listCmd.Flags().StringVar(&listNamespace, "ns", "", "filter by namespace")
	listCmd.Flags().StringVar(&listType, "type", "", "filter by type")
	listCmd.Flags().IntVar(&listLimit, "limit", 20, "max results to return")
	listCmd.Flags().IntVar(&listOffset, "offset", 0, "offset for pagination")
}

func runList(cmd *cobra.Command, args []string) error {
	client := NewClient()

	memories, err := client.ListMemories(listNamespace, listType, listLimit, listOffset)
	if err != nil {
		return err
	}

	if jsonOutput {
		return printJSON(memories)
	}

	if len(memories) == 0 {
		fmt.Fprintln(os.Stdout, "No memories found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAMESPACE\tTYPE\tCONTENT\tCREATED")
	for _, m := range memories {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			m.ID,
			m.Namespace,
			m.Type,
			truncate(m.Content, 60),
			m.CreatedAt,
		)
	}
	w.Flush()
	return nil
}

// ---------------------------------------------------------------------------
// get command
// ---------------------------------------------------------------------------

var getCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a single memory by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runGet,
}

func runGet(cmd *cobra.Command, args []string) error {
	client := NewClient()

	mem, err := client.GetMemory(args[0])
	if err != nil {
		return err
	}

	if jsonOutput {
		return printJSON(mem)
	}

	fmt.Fprintf(os.Stdout, "ID:        %s\n", mem.ID)
	fmt.Fprintf(os.Stdout, "Namespace: %s\n", mem.Namespace)
	fmt.Fprintf(os.Stdout, "Type:      %s\n", mem.Type)
	fmt.Fprintf(os.Stdout, "Version:   %d\n", mem.Version)
	fmt.Fprintf(os.Stdout, "Created:   %s\n", mem.CreatedAt)
	fmt.Fprintf(os.Stdout, "Updated:   %s\n", mem.UpdatedAt)
	if mem.ExpiresAt != nil {
		fmt.Fprintf(os.Stdout, "Expires:   %s\n", *mem.ExpiresAt)
	}
	if mem.Metadata != nil {
		fmt.Fprintf(os.Stdout, "Metadata:  %s\n", *mem.Metadata)
	}
	if mem.SourceID != nil {
		fmt.Fprintf(os.Stdout, "Source ID: %s\n", *mem.SourceID)
	}
	fmt.Fprintf(os.Stdout, "\n%s\n", mem.Content)
	return nil
}

// ---------------------------------------------------------------------------
// rm command
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// namespaces command
// ---------------------------------------------------------------------------

var namespacesCmd = &cobra.Command{
	Use:   "namespaces",
	Short: "List namespaces",
	RunE:  runNamespaces,
}

func runNamespaces(cmd *cobra.Command, args []string) error {
	client := NewClient()

	namespaces, err := client.ListNamespaces()
	if err != nil {
		return err
	}

	if jsonOutput {
		return printJSON(namespaces)
	}

	if len(namespaces) == 0 {
		fmt.Fprintln(os.Stdout, "No namespaces found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDESCRIPTION\tCREATED")
	for _, ns := range namespaces {
		fmt.Fprintf(w, "%s\t%s\t%s\n", ns.Name, ns.Description, ns.CreatedAt)
	}
	w.Flush()
	return nil
}

// ---------------------------------------------------------------------------
// connections command
// ---------------------------------------------------------------------------

var connectionsCmd = &cobra.Command{
	Use:   "connections",
	Short: "List connections",
	RunE:  runConnections,
}

func runConnections(cmd *cobra.Command, args []string) error {
	client := NewClient()

	connections, err := client.ListConnections()
	if err != nil {
		return err
	}

	if jsonOutput {
		return printJSON(connections)
	}

	if len(connections) == 0 {
		fmt.Fprintln(os.Stdout, "No connections found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tTYPE\tACTIVE\tLAST SEEN\tCREATED")
	for _, c := range connections {
		active := "no"
		if c.IsActive {
			active = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			c.ID, c.Name, c.Type, active, c.LastSeen, c.CreatedAt,
		)
	}
	w.Flush()
	return nil
}

// ---------------------------------------------------------------------------
// audit command
// ---------------------------------------------------------------------------

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "View audit log",
	RunE:  runAudit,
}

var (
	auditLimit        int
	auditAction       string
	auditConnectionID string
)

func init() {
	auditCmd.Flags().IntVar(&auditLimit, "limit", 20, "max entries to return")
	auditCmd.Flags().StringVar(&auditAction, "action", "", "filter by action")
	auditCmd.Flags().StringVar(&auditConnectionID, "connection-id", "", "filter by connection ID")
}

func runAudit(cmd *cobra.Command, args []string) error {
	client := NewClient()

	entries, err := client.ListAudit(auditLimit, auditAction, auditConnectionID)
	if err != nil {
		return err
	}

	if jsonOutput {
		return printJSON(entries)
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stdout, "No audit entries found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tACTION\tCONNECTION\tMEMORY\tCREATED")
	for _, e := range entries {
		memID := "-"
		if e.MemoryID != nil {
			memID = *e.MemoryID
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			e.ID, e.Action, e.ConnectionID, memID, e.CreatedAt,
		)
	}
	w.Flush()
	return nil
}

// ---------------------------------------------------------------------------
// config command
// ---------------------------------------------------------------------------

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or update CLI configuration",
	Long:  "Show current configuration, or use 'config set <key> <value>' to update.",
	RunE:  runConfigShow,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Long:  "Set a config value. Keys: auth.api_url, auth.api_key, defaults.namespace, defaults.type, defaults.format",
	Args:  cobra.ExactArgs(2),
	RunE:  runConfigSet,
}

func init() {
	configCmd.AddCommand(configSetCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg := loadConfig()

	if jsonOutput {
		return printJSON(cfg)
	}

	fmt.Fprintf(os.Stdout, "Config file: %s\n\n", configPath())
	fmt.Fprintln(os.Stdout, "[auth]")
	fmt.Fprintf(os.Stdout, "  api_url = %s\n", cfg.Auth.APIURL)
	if cfg.Auth.APIKey != "" {
		// Mask the API key, showing only last 4 chars.
		masked := strings.Repeat("*", max(0, len(cfg.Auth.APIKey)-4)) + cfg.Auth.APIKey[max(0, len(cfg.Auth.APIKey)-4):]
		fmt.Fprintf(os.Stdout, "  api_key = %s\n", masked)
	} else {
		fmt.Fprintln(os.Stdout, "  api_key = (not set)")
	}
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "[defaults]")
	fmt.Fprintf(os.Stdout, "  namespace = %s\n", cfg.Defaults.Namespace)
	fmt.Fprintf(os.Stdout, "  type      = %s\n", cfg.Defaults.Type)
	fmt.Fprintf(os.Stdout, "  format    = %s\n", cfg.Defaults.Format)

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	cfg := loadConfig()

	key, value := args[0], args[1]
	if err := setConfigValue(cfg, key, value); err != nil {
		return err
	}

	if err := saveConfig(cfg); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Set %s = %s\n", key, value)
	return nil
}

// ---------------------------------------------------------------------------
// import command
// ---------------------------------------------------------------------------

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import memories from a file",
	Long:  "Import memories from Claude MEMORY.md, ChatGPT JSON, or CSV files",
	Args:  cobra.ExactArgs(1),
	RunE:  runImport,
}

var (
	importNamespace string
	importFormat    string
	importDryRun    bool
)

func init() {
	importCmd.Flags().StringVar(&importNamespace, "namespace", "", "Override namespace for all imported memories")
	importCmd.Flags().StringVar(&importFormat, "format", "", "Force format (memory-md, chatgpt, csv, json)")
	importCmd.Flags().BoolVar(&importDryRun, "dry-run", false, "Parse and show count without importing")
}

func runImport(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	// Read file (or stdin if "-")
	var data []byte
	var filename string
	var err error
	if filePath == "-" {
		data, err = io.ReadAll(os.Stdin)
		filename = "stdin"
	} else {
		data, err = os.ReadFile(filePath)
		filename = filepath.Base(filePath)
	}
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	if importDryRun {
		// Just parse and show count
		var format transfer.Format
		if importFormat != "" {
			format = transfer.Format(importFormat)
		} else {
			format = transfer.DetectFormat(filename, data)
		}
		parser := transfer.GetParser(format)
		if parser == nil {
			return fmt.Errorf("unsupported format: %s", format)
		}
		memories, err := parser.Parse(data)
		if err != nil {
			return fmt.Errorf("parse: %w", err)
		}
		fmt.Printf("Dry run: would import %d memories (format: %s)\n", len(memories), format)
		return nil
	}

	// Build multipart request
	client := NewClient()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	part.Write(data)
	if importNamespace != "" {
		writer.WriteField("namespace", importNamespace)
	}
	if importFormat != "" {
		writer.WriteField("format", importFormat)
	}
	writer.Close()

	req, err := client.newRequest("POST", "/api/import", body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("import request: %w", err)
	}
	defer resp.Body.Close()

	var result transfer.ImportResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if jsonOutput {
		return printJSON(result)
	}

	fmt.Printf("Imported: %d created, %d skipped\n", result.Created, result.Skipped)
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Printf("  error: %s\n", e)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// export command
// ---------------------------------------------------------------------------

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export memories to a file",
	Long:  "Export memories as Claude MEMORY.md, JSON, or CSV",
	RunE:  runExport,
}

var (
	exportFormat    string
	exportNamespace string
	exportOutput    string
)

func init() {
	exportCmd.Flags().StringVar(&exportFormat, "format", "json", "Export format (memory-md, json, csv)")
	exportCmd.Flags().StringVar(&exportNamespace, "namespace", "", "Export only this namespace")
	exportCmd.Flags().StringVar(&exportOutput, "output", "", "Output file path (defaults to stdout)")
}

func runExport(cmd *cobra.Command, args []string) error {
	client := NewClient()

	url := "/api/export?format=" + exportFormat
	if exportNamespace != "" {
		url += "&namespace=" + exportNamespace
	}

	req, err := client.newRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := client.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("export request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("export failed (HTTP %d): %s", resp.StatusCode, respBody)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if exportOutput != "" {
		if err := os.WriteFile(exportOutput, data, 0644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
		fmt.Printf("Exported to %s\n", exportOutput)
	} else {
		os.Stdout.Write(data)
	}
	return nil
}

// ---------------------------------------------------------------------------
// keys command
// ---------------------------------------------------------------------------

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage API keys",
	Long:  "Create, list, and revoke API keys for connecting external tools",
}

var keysCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runKeysCreate,
}

var (
	keysCreateType string
	keysCreatePerm string
)

func init() {
	keysCreateCmd.Flags().StringVar(&keysCreateType, "type", "api", "Connection type (api, mcp, cli)")
	keysCreateCmd.Flags().StringVar(&keysCreatePerm, "permissions", "read,write", "Permissions (read,write,delete,admin)")
	keysCmd.AddCommand(keysCreateCmd, keysListCmd, keysRevokeCmd)
}

func runKeysCreate(cmd *cobra.Command, args []string) error {
	client := NewClient()
	name := args[0]

	body := map[string]string{
		"name":        name,
		"type":        keysCreateType,
		"permissions": keysCreatePerm,
	}

	resp, _, err := client.do("POST", "/api/keys", body)
	if err != nil {
		return err
	}

	if jsonOutput {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		ID     string `json:"id"`
		APIKey string `json:"api_key"`
		Name   string `json:"name"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return err
	}

	fmt.Printf("Created API key: %s\n", result.Name)
	fmt.Printf("Key: %s\n", result.APIKey)
	fmt.Printf("ID:  %s\n", result.ID)
	fmt.Println("\nSave this key now -- it will not be shown again.")
	return nil
}

var keysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List API keys",
	RunE:  runKeysList,
}

func runKeysList(cmd *cobra.Command, args []string) error {
	client := NewClient()

	resp, _, err := client.do("GET", "/api/keys", nil)
	if err != nil {
		return err
	}

	if jsonOutput {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		Keys []Connection `json:"keys"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return err
	}

	if len(result.Keys) == 0 {
		fmt.Println("No API keys found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tTYPE\tPERMISSIONS\tCREATED")
	for _, k := range result.Keys {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", k.ID, k.Name, k.Type, k.Permissions, k.CreatedAt)
	}
	w.Flush()
	return nil
}

var keysRevokeCmd = &cobra.Command{
	Use:   "revoke <id>",
	Short: "Revoke an API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runKeysRevoke,
}

func runKeysRevoke(cmd *cobra.Command, args []string) error {
	client := NewClient()

	_, _, err := client.do("DELETE", "/api/keys/"+args[0], nil)
	if err != nil {
		return err
	}

	fmt.Println("Key revoked.")
	return nil
}

// ---------------------------------------------------------------------------
// Output helpers
// ---------------------------------------------------------------------------

// printJSON marshals v as indented JSON and writes to stdout.
func printJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	fmt.Fprintln(os.Stdout, string(data))
	return nil
}

// truncate shortens s to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	// Replace newlines with spaces for table display.
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// printSearchResults renders search results as a table.
func printSearchResults(results []SearchResult) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	// Check if any result has vector scores to show the expanded header.
	hasVec := false
	for _, r := range results {
		if r.VectorScore > 0 || r.FTSScore > 0 {
			hasVec = true
			break
		}
	}
	if hasVec {
		fmt.Fprintln(w, "SCORE\tFTS\tVEC\tID\tNAMESPACE\tTYPE\tCONTENT")
		for _, r := range results {
			fmt.Fprintf(w, "%.4f\t%.4f\t%.4f\t%s\t%s\t%s\t%s\n",
				r.Score, r.FTSScore, r.VectorScore,
				r.Memory.ID,
				r.Memory.Namespace,
				r.Memory.Type,
				truncate(r.Memory.Content, 60),
			)
		}
	} else {
		fmt.Fprintln(w, "SCORE\tID\tNAMESPACE\tTYPE\tCONTENT")
		for _, r := range results {
			fmt.Fprintf(w, "%.4f\t%s\t%s\t%s\t%s\n",
				r.Score,
				r.Memory.ID,
				r.Memory.Namespace,
				r.Memory.Type,
				truncate(r.Memory.Content, 60),
			)
		}
	}
	w.Flush()
}
