package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"zai-api-client/pkg/client"
)

var filesCmd = &cobra.Command{
	Use:   "files",
	Short: "File upload and management",
	Long:  `Upload files for use in other API calls (batch input, fine-tuning, retrieval, voice-clone input).`,
}

var filesUploadCmd = &cobra.Command{
	Use:   "upload [file]",
	Short: "Upload a file",
	Args:  cobra.ExactArgs(1),
	RunE:  runFilesUpload,
}

var filesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List uploaded files",
	Args:  cobra.NoArgs,
	RunE:  runFilesList,
}

var filesDeleteCmd = &cobra.Command{
	Use:   "delete [file-id]",
	Short: "Delete an uploaded file",
	Args:  cobra.ExactArgs(1),
	RunE:  runFilesDelete,
}

var filesDownloadCmd = &cobra.Command{
	Use:   "download [file-id] [output-path]",
	Short: "Download an uploaded file's content",
	Args:  cobra.ExactArgs(2),
	RunE:  runFilesDownload,
}

func init() {
	rootCmd.AddCommand(filesCmd)
	filesCmd.AddCommand(filesUploadCmd, filesListCmd, filesDeleteCmd, filesDownloadCmd)

	filesUploadCmd.Flags().String("purpose", "batch", "File purpose: fine-tune, retrieval, batch, or voice-clone-input")
	filesListCmd.Flags().String("purpose", "", "Filter by purpose (omit for all)")
}

func runFilesUpload(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	path := args[0]
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	purpose, _ := cmd.Flags().GetString("purpose")

	fmt.Printf("📤 Uploading %s...\n", filepath.Base(path))
	f, err := apiClient.Files().Upload(cmd.Context(), filepath.Base(path), data, client.FilePurpose(purpose))
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	fmt.Printf("✅ Uploaded: %s (%d bytes, status: %s)\n", f.ID, f.Bytes, f.Status)
	return nil
}

func runFilesList(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	purpose, _ := cmd.Flags().GetString("purpose")
	list, err := apiClient.Files().List(cmd.Context(), client.FilePurpose(purpose))
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	if len(list.Data) == 0 {
		fmt.Println("No files found")
		return nil
	}
	for _, f := range list.Data {
		fmt.Printf("%s  %-20s  %8d bytes  %-10s  %s\n", f.ID, f.Filename, f.Bytes, f.Purpose, f.Status)
	}
	return nil
}

func runFilesDelete(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	resp, err := apiClient.Files().Delete(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	if resp.Deleted {
		fmt.Printf("✅ Deleted: %s\n", resp.ID)
	} else {
		fmt.Printf("⚠️  Not deleted: %s\n", resp.ID)
	}
	return nil
}

func runFilesDownload(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	fileID, outPath := args[0], args[1]
	data, err := apiClient.Files().Content(cmd.Context(), fileID)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	if err := os.WriteFile(outPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write %s: %w", outPath, err)
	}

	fmt.Printf("✅ Downloaded %d bytes to %s\n", len(data), outPath)
	return nil
}
