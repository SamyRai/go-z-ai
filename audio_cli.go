package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var audioCmd = &cobra.Command{
	Use:   "audio",
	Short: "Audio transcription and text-to-speech",
	Long:  `Transcribe audio with Z.AI's glm-asr model, or synthesize speech with GLM-TTS.`,
}

var audioTranscribeCmd = &cobra.Command{
	Use:   "transcribe [file]",
	Short: "Transcribe a .wav or .mp3 file (<=25MB, <=30s)",
	Args:  cobra.ExactArgs(1),
	RunE:  runAudioTranscribe,
}

var audioSpeechCmd = &cobra.Command{
	Use:   "speech [text] [output-path]",
	Short: "Synthesize speech from text (GLM-TTS)",
	Args:  cobra.ExactArgs(2),
	RunE:  runAudioSpeech,
}

func init() {
	rootCmd.AddCommand(audioCmd)
	audioCmd.AddCommand(audioTranscribeCmd, audioSpeechCmd)

	audioTranscribeCmd.Flags().String("prompt", "", "Previous transcription context (recommended <8000 chars)")
	audioTranscribeCmd.Flags().StringArray("hotword", nil, "Domain-specific vocabulary word (repeatable, max 100)")

	audioSpeechCmd.Flags().String("voice", client.VoiceTongtong, "Voice: tongtong, chuichui, xiaochen, jam, kazi, douji, luodo, or a cloned voice ID")
	audioSpeechCmd.Flags().String("format", "", "Output format: wav or pcm (API default: pcm)")
	audioSpeechCmd.Flags().Float64("speed", 0, "Speed 0.5-2 (API default 1.0)")
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

func runAudioSpeech(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	text, outPath := args[0], args[1]
	voice, _ := cmd.Flags().GetString("voice")
	format, _ := cmd.Flags().GetString("format")
	speed, _ := cmd.Flags().GetFloat64("speed")

	fmt.Printf("🔊 Synthesizing speech for: %s\n", text)
	data, err := apiClient.Audio().Speech(cmd.Context(), client.AudioSpeechRequest{
		Input:          text,
		Voice:          voice,
		ResponseFormat: format,
		Speed:          speed,
	})
	if err != nil {
		return fmt.Errorf("speech synthesis failed: %w", err)
	}

	if err := os.WriteFile(outPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write %s: %w", outPath, err)
	}

	fmt.Printf("✅ Wrote %d bytes to %s\n", len(data), outPath)
	return nil
}
