package cli

import (
	"fmt"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var videoCmd = &cobra.Command{
	Use:   "video",
	Short: "Video generation",
	Long:  `Generate videos with Z.AI's cogvideox-3 / Vidu models. Always asynchronous — use 'video status' to poll.`,
}

var videoGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Submit a video generation task",
	Args:  cobra.NoArgs,
	RunE:  runWithClient(runVideoGenerate),
}

var videoStatusCmd = &cobra.Command{
	Use:   "status [id]",
	Short: "Check an async video generation task",
	Args:  cobra.ExactArgs(1),
	RunE:  runWithClient(runVideoStatus),
}

func init() {
	rootCmd.AddCommand(videoCmd)
	videoCmd.AddCommand(videoGenerateCmd, videoStatusCmd)

	videoGenerateCmd.Flags().String("model", "cogvideox-3", "Model: cogvideox-3, viduq1-text, viduq1-image, vidu2-image, viduq1-start-end, vidu2-start-end, vidu2-reference")
	videoGenerateCmd.Flags().String("prompt", "", "Text prompt (<=512 chars)")
	videoGenerateCmd.Flags().StringArray("image", nil, "Image URL/base64 (repeatable; count/meaning depends on --model)")
	videoGenerateCmd.Flags().String("size", "", "Resolution, e.g. 1920x1080 (model-dependent)")
	videoGenerateCmd.Flags().String("aspect-ratio", "", "16:9 | 9:16 | 1:1 (Vidu text/reference models)")
	videoGenerateCmd.Flags().Int("duration", 0, "Duration in seconds (valid values vary by model)")
	videoGenerateCmd.Flags().Int("fps", 0, "cogvideox-3 only: 30 or 60")
	videoGenerateCmd.Flags().String("style", "", "viduq1-text only: general or anime")
	videoGenerateCmd.Flags().String("quality", "", "cogvideox-3 only: speed or quality")
	videoGenerateCmd.Flags().String("movement", "", "Vidu models only: auto | small | medium | large")
	videoGenerateCmd.Flags().Bool("audio", false, "Generate with audio (model-dependent)")
	addFormatFlag("text", videoGenerateCmd, videoStatusCmd)
}

func runVideoGenerate(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	model, _ := cmd.Flags().GetString("model")
	prompt, _ := cmd.Flags().GetString("prompt")
	images, _ := cmd.Flags().GetStringArray("image")
	size, _ := cmd.Flags().GetString("size")
	aspectRatio, _ := cmd.Flags().GetString("aspect-ratio")
	duration, _ := cmd.Flags().GetInt("duration")
	fps, _ := cmd.Flags().GetInt("fps")
	style, _ := cmd.Flags().GetString("style")
	quality, _ := cmd.Flags().GetString("quality")
	movement, _ := cmd.Flags().GetString("movement")
	withAudio, _ := cmd.Flags().GetBool("audio")

	resp, err := apiClient.Videos().Generate(cmd.Context(), client.VideoGenerationRequest{
		Model:             model,
		Prompt:            prompt,
		ImageURL:          images,
		Size:              size,
		AspectRatio:       aspectRatio,
		Duration:          duration,
		FPS:               fps,
		Style:             style,
		Quality:           quality,
		MovementAmplitude: movement,
		WithAudio:         withAudio,
	})
	if err != nil {
		return fmt.Errorf("video generation failed: %w", err)
	}

	return emit(cmd, resp, func() error {
		fmt.Printf("⏳ Task submitted: %s (status: %s)\n", resp.ID, resp.TaskStatus)
		fmt.Printf("   Check with: go-z-ai video status %s\n", resp.ID)
		return nil
	})
}

func runVideoStatus(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	result, err := apiClient.GetAsyncResult(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("failed to check status: %w", err)
	}

	return emit(cmd, result, func() error {
		fmt.Printf("Status: %s\n", result.TaskStatus)
		for i, v := range result.VideoResult {
			fmt.Printf("Video %d: %s\n", i+1, v.URL)
			if v.CoverImageURL != "" {
				fmt.Printf("  Cover: %s\n", v.CoverImageURL)
			}
		}
		return nil
	})
}
