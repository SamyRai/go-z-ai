package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"zai-api-client/pkg/client"
)

var audioCmd = &cobra.Command{
	Use:   "audio",
	Short: "Audio transcription",
	Long:  `Transcribe audio with Z.AI's glm-asr model.`,
}

var audioTranscribeCmd = &cobra.Command{
	Use:   "transcribe [file]",
	Short: "Transcribe a .wav or .mp3 file (<=25MB, <=30s)",
	Args:  cobra.ExactArgs(1),
	RunE:  runAudioTranscribe,
}

func init() {
	rootCmd.AddCommand(audioCmd)
	audioCmd.AddCommand(audioTranscribeCmd)

	audioTranscribeCmd.Flags().String("prompt", "", "Previous transcription context (recommended <8000 chars)")
	audioTranscribeCmd.Flags().StringArray("hotword", nil, "Domain-specific vocabulary word (repeatable, max 100)")
}

func runAudioTranscribe(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	path := args[0]
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	prompt, _ := cmd.Flags().GetString("prompt")
	hotwords, _ := cmd.Flags().GetStringArray("hotword")

	fmt.Printf("🎙️  Transcribing %s...\n", filepath.Base(path))
	resp, err := apiClient.Audio().Transcribe(cmd.Context(), client.AudioTranscriptionRequest{
		FileName: filepath.Base(path),
		FileData: data,
		Prompt:   prompt,
		Hotwords: hotwords,
	})
	if err != nil {
		return fmt.Errorf("transcription failed: %w", err)
	}

	fmt.Printf("✅ %s\n", resp.Text)
	return nil
}
