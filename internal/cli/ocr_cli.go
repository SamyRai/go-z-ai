package cli

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
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
	RunE:  runWithClient(runOCRParse),
}

var ocrHandwritingCmd = &cobra.Command{
	Use:   "handwriting [file]",
	Short: "Recognize handwritten text in an image",
	Long: `Recognize handwritten text in an image, returning each recognized word
with its bounding box (and, with --probability, per-word confidence). Distinct
from "ocr parse": this targets short handwritten snippets, not full documents.`,
	Args: cobra.ExactArgs(1),
	RunE: runWithClient(runOCRHandwriting),
}

func init() {
	rootCmd.AddCommand(ocrCmd)
	ocrCmd.AddCommand(ocrParseCmd, ocrHandwritingCmd)

	ocrParseCmd.Flags().Int("start-page", 0, "First PDF page to parse (1-indexed; PDFs only)")
	ocrParseCmd.Flags().Int("end-page", 0, "Last PDF page to parse (1-indexed; PDFs only)")
	addFormatFlag("text", ocrParseCmd)

	ocrHandwritingCmd.Flags().String("language", "", "Language hint (optional)")
	ocrHandwritingCmd.Flags().Bool("probability", false, "Include per-word confidence statistics")
}

func runOCRParse(cmd *cobra.Command, args []string, apiClient *client.Client) error {
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

	fmt.Fprintln(os.Stderr, "📄 Parsing document...")
	resp, err := apiClient.Layout().Parse(cmd.Context(), client.LayoutParsingRequest{
		File:        file,
		StartPageID: startPage,
		EndPageID:   endPage,
	})
	if err != nil {
		return fmt.Errorf("layout parsing failed: %w", err)
	}

	return emit(cmd, resp, func() error {
		fmt.Println(resp.MDResults)
		return nil
	})
}

func runOCRHandwriting(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	path := args[0]
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	language, _ := cmd.Flags().GetString("language")
	probability, _ := cmd.Flags().GetBool("probability")

	fmt.Println("✍️  Recognizing handwriting...")
	resp, err := apiClient.Layout().HandwritingOCR(cmd.Context(), client.HandwritingOCRRequest{
		FileName:     filepath.Base(path),
		FileData:     data,
		LanguageType: language,
		Probability:  probability,
	})
	if err != nil {
		return fmt.Errorf("handwriting OCR failed: %w", err)
	}

	if resp.WordsResultNum == 0 {
		fmt.Println("No text recognized")
		return nil
	}
	for _, wr := range resp.WordsResult {
		if probability && wr.Probability != nil {
			fmt.Printf("%s  (confidence: %.2f)\n", wr.Words, wr.Probability.Average)
		} else {
			fmt.Println(wr.Words)
		}
	}
	return nil
}
