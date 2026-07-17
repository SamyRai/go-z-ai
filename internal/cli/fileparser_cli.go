package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var fileParserCmd = &cobra.Command{
	Use:   "parser",
	Short: "Parse documents into text (RAG/retrieval preprocessing)",
	Long:  `Parse PDF/Office/image documents into text or a downloadable result, for RAG/retrieval preprocessing.`,
}

var fileParserSyncCmd = &cobra.Command{
	Use:   "parse [file] [file-type]",
	Short: "Parse a document synchronously and print the result",
	Long:  `Parse a document synchronously (tool_type prime-sync). file-type is required (e.g. PDF, DOCX, PNG) — the API rejects requests without it despite documenting it as optional.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runFileParserSync,
}

var fileParserCreateCmd = &cobra.Command{
	Use:   "create [file] [tool-type] [file-type]",
	Short: "Submit an async document parse task",
	Long:  `Submit an async document parse task (tool-type: lite, expert, or prime). Poll the result with "fileparser result".`,
	Args:  cobra.ExactArgs(3),
	RunE:  runFileParserCreate,
}

var fileParserResultCmd = &cobra.Command{
	Use:   "result [task-id] [format]",
	Short: "Fetch the result of an async parse task",
	Long:  `Fetch the result of an async parse task submitted via "fileparser create". format is "text" or "download_link".`,
	Args:  cobra.ExactArgs(2),
	RunE:  runFileParserResult,
}

func init() {
	rootCmd.AddCommand(fileParserCmd)
	fileParserCmd.AddCommand(fileParserSyncCmd, fileParserCreateCmd, fileParserResultCmd)
}

func runFileParserSync(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	path, fileType := args[0], args[1]
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	fmt.Printf("📄 Parsing %s...\n", filepath.Base(path))
	resp, err := apiClient.FileParser().Sync(cmd.Context(), client.FileParserRequest{
		FileName: filepath.Base(path),
		FileData: data,
		ToolType: client.FileParserToolPrimeSync,
		FileType: fileType,
	})
	if err != nil {
		return fmt.Errorf("file parse failed: %w", err)
	}

	fmt.Println(resp.Content)
	return nil
}

func runFileParserCreate(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	path, toolType, fileType := args[0], args[1], args[2]
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	resp, err := apiClient.FileParser().Create(cmd.Context(), client.FileParserRequest{
		FileName: filepath.Base(path),
		FileData: data,
		ToolType: toolType,
		FileType: fileType,
	})
	if err != nil {
		return fmt.Errorf("failed to submit parse task: %w", err)
	}

	fmt.Printf("✅ Task submitted: %s\n", resp.TaskID)
	return nil
}

func runFileParserResult(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	resp, err := apiClient.FileParser().Result(cmd.Context(), args[0], args[1])
	if err != nil {
		return fmt.Errorf("failed to get parse result: %w", err)
	}

	switch resp.Status {
	case client.FileParserStatusProcessing:
		fmt.Println("⏳ Still processing, try again shortly")
	case client.FileParserStatusFailed:
		fmt.Printf("❌ Failed: %s\n", resp.Message)
	default:
		if resp.Content != "" {
			fmt.Println(resp.Content)
		}
		if resp.ParsingResultURL != "" {
			fmt.Println(resp.ParsingResultURL)
		}
	}
	return nil
}
