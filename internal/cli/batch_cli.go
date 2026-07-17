package cli

import (
	"fmt"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Async bulk-processing jobs",
	Long: `Submit and manage batch jobs: many chat-completion or embedding requests
processed asynchronously from a JSONL input file. Upload the input file first
with "zai-client files upload --purpose batch <file.jsonl>", then create the
batch with the resulting file ID.`,
}

var batchCreateCmd = &cobra.Command{
	Use:   "create [input-file-id]",
	Short: "Submit a new batch job",
	Args:  cobra.ExactArgs(1),
	RunE:  runWithClient(runBatchCreate),
}

var batchStatusCmd = &cobra.Command{
	Use:   "status [batch-id]",
	Short: "Check a batch job's status",
	Args:  cobra.ExactArgs(1),
	RunE:  runWithClient(runBatchStatus),
}

var batchListCmd = &cobra.Command{
	Use:   "list",
	Short: "List batch jobs",
	Args:  cobra.NoArgs,
	RunE:  runWithClient(runBatchList),
}

var batchCancelCmd = &cobra.Command{
	Use:   "cancel [batch-id]",
	Short: "Cancel an in-progress batch job",
	Args:  cobra.ExactArgs(1),
	RunE:  runWithClient(runBatchCancel),
}

func init() {
	rootCmd.AddCommand(batchCmd)
	batchCmd.AddCommand(batchCreateCmd, batchStatusCmd, batchListCmd, batchCancelCmd)

	batchCreateCmd.Flags().String("endpoint", string(client.BatchEndpointChatCompletions), "Target endpoint (currently the API's only supported value)")
	batchListCmd.Flags().String("after", "", "Cursor: list batches after this batch ID")
	batchListCmd.Flags().Int("limit", 0, "Max batches to return (0 = server default)")
}

func runBatchCreate(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	endpoint, _ := cmd.Flags().GetString("endpoint")

	b, err := apiClient.Batch().Create(cmd.Context(), client.BatchCreateRequest{
		InputFileID: args[0],
		Endpoint:    client.BatchEndpoint(endpoint),
	})
	if err != nil {
		return fmt.Errorf("failed to create batch: %w", err)
	}

	fmt.Printf("⏳ Batch submitted: %s (status: %s)\n", b.ID, b.Status)
	fmt.Printf("   Check with: zai-client batch status %s\n", b.ID)
	return nil
}

func runBatchStatus(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	b, err := apiClient.Batch().Retrieve(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("failed to check batch status: %w", err)
	}

	fmt.Printf("Status: %s\n", b.Status)
	if b.RequestCounts != nil {
		fmt.Printf("Requests: %d/%d completed, %d failed\n", b.RequestCounts.Completed, b.RequestCounts.Total, b.RequestCounts.Failed)
	}
	if b.OutputFileID != "" {
		fmt.Printf("Output file: %s (download with: zai-client files download %s <path>)\n", b.OutputFileID, b.OutputFileID)
	}
	if b.ErrorFileID != "" {
		fmt.Printf("Error file: %s\n", b.ErrorFileID)
	}
	return nil
}

func runBatchList(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	after, _ := cmd.Flags().GetString("after")
	limit, _ := cmd.Flags().GetInt("limit")

	list, err := apiClient.Batch().List(cmd.Context(), after, limit)
	if err != nil {
		return fmt.Errorf("failed to list batches: %w", err)
	}

	if len(list.Data) == 0 {
		fmt.Println("No batches found")
		return nil
	}
	for _, b := range list.Data {
		fmt.Printf("%s  %-12s  %-24s\n", b.ID, b.Status, b.Endpoint)
	}
	if list.HasMore {
		fmt.Printf("(more results available; pass --after %s)\n", list.Data[len(list.Data)-1].ID)
	}
	return nil
}

func runBatchCancel(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	b, err := apiClient.Batch().Cancel(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("failed to cancel batch: %w", err)
	}

	fmt.Printf("Status: %s\n", b.Status)
	return nil
}
