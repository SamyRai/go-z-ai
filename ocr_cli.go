package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"zai-api-client/pkg/client"
)

var ocrCmd = &cobra.Command{
	Use:   "ocr",
	Short: "Layout parsing (OCR)",
	Long:  `Parse an image or PDF into Markdown with Z.AI's glm-ocr model.`,
}

var ocrParseCmd = &cobra.Command{
	Use:   "parse [file-or-url]",
	Short: "Parse a local file or URL into Markdown",
	Args:  cobra.ExactArgs(1),
	RunE:  runOCRParse,
}

func init() {
	rootCmd.AddCommand(ocrCmd)
	ocrCmd.AddCommand(ocrParseCmd)

	ocrParseCmd.Flags().Int("start-page", 0, "First PDF page to parse (1-indexed; PDFs only)")
	ocrParseCmd.Flags().Int("end-page", 0, "Last PDF page to parse (1-indexed; PDFs only)")
}

func runOCRParse(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	target := args[0]
	file := target
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		data, err := os.ReadFile(target)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", target, err)
		}
		file = base64.StdEncoding.EncodeToString(data)
	}

	startPage, _ := cmd.Flags().GetInt("start-page")
	endPage, _ := cmd.Flags().GetInt("end-page")

	fmt.Println("📄 Parsing document...")
	resp, err := apiClient.Layout().Parse(cmd.Context(), client.LayoutParsingRequest{
		File:        file,
		StartPageID: startPage,
		EndPageID:   endPage,
	})
	if err != nil {
		return fmt.Errorf("layout parsing failed: %w", err)
	}

	fmt.Println(resp.MDResults)
	return nil
}
