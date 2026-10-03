package cli

import (
	"fmt"
	"strings"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var videoCmd = &cobra.Command{
	Use:   "video",
	Short: "Video generation",
	Long:  `Generate videos with Z.AI's CogVideoX / Vidu models. Always asynchronous — use 'video status' to poll.`,
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
	RunE:  runWithClient(runAsyncStatus),
}

func init() {
	rootCmd.AddCommand(videoCmd)
	videoCmd.AddCommand(videoGenerateCmd, videoStatusCmd)

	videoGenerateCmd.Flags().String("model", client.ModelCogVideoX3, "Model: "+strings.Join(client.VideoModels, ", "))
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
	videoGenerateCmd.Flags().Bool("off-peak", false, "Queue for off-peak processing at a lower price")
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
	offPeak, _ := cmd.Flags().GetBool("off-peak")

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
		OffPeak:           offPeak,
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
