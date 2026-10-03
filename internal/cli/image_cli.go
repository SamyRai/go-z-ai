package cli

import (
	"fmt"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Image generation",
	Long:  "Generate images with Z.AI's " + client.ModelGLMImage + " / " + client.ModelCogView4 + " models.",
}

var imageGenerateCmd = &cobra.Command{
	Use:   "generate [prompt]",
	Short: "Generate an image from a text prompt",
	Args:  cobra.ExactArgs(1),
	RunE:  runWithClient(runImageGenerate),
}

var imageStatusCmd = &cobra.Command{
	Use:   "status [id]",
	Short: "Check an async image generation task",
	Args:  cobra.ExactArgs(1),
	RunE:  runWithClient(runAsyncStatus),
}

func init() {
	rootCmd.AddCommand(imageCmd)
	imageCmd.AddCommand(imageGenerateCmd, imageStatusCmd)

	imageGenerateCmd.Flags().String("model", client.ModelGLMImage, "Model: "+client.ModelGLMImage+" or "+client.ModelCogView4)
	imageGenerateCmd.Flags().String("size", "", "Image size, e.g. 1280x1280 (default 1280x1280)")
	imageGenerateCmd.Flags().String("quality", "", "Quality: hd (default, ~20s) or standard (~5-10s)")
	imageGenerateCmd.Flags().Bool("async", false, "Submit as an async task instead of waiting (use 'image status' to poll)")
	addFormatFlag("text", imageGenerateCmd, imageStatusCmd)
}

func runImageGenerate(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	model, _ := cmd.Flags().GetString("model")
	size, _ := cmd.Flags().GetString("size")
	quality, _ := cmd.Flags().GetString("quality")
	async, _ := cmd.Flags().GetBool("async")

	req := client.ImageGenerationRequest{
		Model:   model,
		Prompt:  args[0],
		Size:    size,
		Quality: quality,
	}

	if async {
		resp, err := apiClient.Images().GenerateAsync(cmd.Context(), req)
		if err != nil {
			return fmt.Errorf("image generation failed: %w", err)
		}
		return emit(cmd, resp, func() error {
			fmt.Printf("⏳ Task submitted: %s (status: %s)\n", resp.ID, resp.TaskStatus)
			fmt.Printf("   Check with: go-z-ai image status %s\n", resp.ID)
			return nil
		})
	}

	progressf("🎨 Generating image...\n")
	resp, err := apiClient.Images().Generate(cmd.Context(), req)
	if err != nil {
		return fmt.Errorf("image generation failed: %w", err)
	}
	return emit(cmd, resp, func() error {
		if len(resp.Data) == 0 {
			fmt.Println("No image returned")
			return nil
		}
		for i, d := range resp.Data {
			fmt.Printf("✅ Image %d: %s\n", i+1, d.URL)
		}
		fmt.Println("   (URL expires after 30 days)")
		return nil
	})
}
