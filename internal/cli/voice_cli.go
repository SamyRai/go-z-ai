package cli

import (
	"fmt"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var voiceCmd = &cobra.Command{
	Use:   "voice",
	Short: "Manage GLM-TTS voice clones",
	Long:  `Create, delete, and list custom voice clones for use with "zai-client audio speech --voice".`,
}

var voiceCloneCmd = &cobra.Command{
	Use:   "clone [voice-name] [sample-file-id] [preview-text]",
	Short: "Clone a voice from a sample audio file",
	Long:  `Clone a voice from a sample audio file already uploaded via "zai-client files upload --purpose voice-clone-input".`,
	Args:  cobra.ExactArgs(3),
	RunE:  runVoiceClone,
}

var voiceDeleteCmd = &cobra.Command{
	Use:   "delete [voice-id]",
	Short: "Delete a cloned voice",
	Args:  cobra.ExactArgs(1),
	RunE:  runVoiceDelete,
}

var voiceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available voices",
	Args:  cobra.NoArgs,
	RunE:  runVoiceList,
}

func init() {
	rootCmd.AddCommand(voiceCmd)
	voiceCmd.AddCommand(voiceCloneCmd, voiceDeleteCmd, voiceListCmd)

	voiceListCmd.Flags().String("name", "", "Filter by voice name (fuzzy match)")
	voiceListCmd.Flags().String("type", "", "Filter by type: OFFICIAL or PRIVATE")
}

func runVoiceClone(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	voiceName, fileID, previewText := args[0], args[1], args[2]

	resp, err := apiClient.Voice().Clone(cmd.Context(), client.VoiceCloneRequest{
		VoiceName: voiceName,
		FileID:    fileID,
		Input:     previewText,
	})
	if err != nil {
		return fmt.Errorf("voice clone failed: %w", err)
	}

	fmt.Printf("✅ Cloned voice: %s (preview file: %s)\n", resp.Voice, resp.FileID)
	return nil
}

func runVoiceDelete(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	resp, err := apiClient.Voice().Delete(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("voice delete failed: %w", err)
	}

	fmt.Printf("✅ Deleted: %s (at %s)\n", resp.Voice, resp.UpdateTime)
	return nil
}

func runVoiceList(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	voiceType, _ := cmd.Flags().GetString("type")

	voices, err := apiClient.Voice().List(cmd.Context(), name, voiceType)
	if err != nil {
		return fmt.Errorf("voice list failed: %w", err)
	}

	if len(voices) == 0 {
		fmt.Println("No voices found")
		return nil
	}
	for _, v := range voices {
		fmt.Printf("%s  %-10s  %-20s  %s\n", v.Voice, v.VoiceType, v.VoiceName, v.CreateTime)
	}
	return nil
}
