package cli

import (
	"fmt"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

// runAsyncStatus is the shared "chat async-result / image status / video
// status <id>" handler: one async-result endpoint serves every task type, and
// the result carries whichever of choices, images, or videos the task made.
// The status goes to stderr so stdout carries only the result.
func runAsyncStatus(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	result, err := apiClient.GetAsyncResult(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("failed to check task %s: %w", args[0], err)
	}
	return emit(cmd, result, func() error {
		progressf("status: %s\n", result.TaskStatus)
		if len(result.Choices) > 0 {
			printReply(result.Choices[0].Message, false)
		}
		for i, img := range result.ImageResult {
			fmt.Printf("Image %d: %s\n", i+1, img.URL)
		}
		for i, v := range result.VideoResult {
			fmt.Printf("Video %d: %s\n", i+1, v.URL)
			if v.CoverImageURL != "" {
				fmt.Printf("  Cover: %s\n", v.CoverImageURL)
			}
		}
		return nil
	})
}
