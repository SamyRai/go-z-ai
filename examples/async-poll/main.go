// Command async-poll demonstrates the async image flow: submit a request,
// receive a task id, then block on WaitForResult until the task is terminal.
//
// Usage:
//
//	export ZAI_API_KEY=your_api_key_here
//	go run ./examples/async-poll "a serene mountain lake at dawn"
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

func main() {
	prompt := "a serene mountain lake at dawn"
	if len(os.Args) > 1 {
		prompt = os.Args[1]
	}

	c, err := client.NewClientFromEnv()
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	ctx := context.Background()

	// Step 1: submit. Returns immediately with a task id; TaskStatus is
	// typically "PROCESSING" at this point.
	task, err := c.Images().GenerateAsync(ctx, client.ImageGenerationRequest{
		Model:  "cogview-4-250304",
		Prompt: prompt,
		Size:   "1280x1280",
	})
	if err != nil {
		log.Fatalf("submit: %v", err)
	}
	fmt.Fprintf(os.Stderr, "submitted task %s (status=%s), polling...\n", task.ID, task.TaskStatus)

	// Step 2: poll. WaitForResult blocks until TaskStatus is SUCCESS or FAIL,
	// or ctx is cancelled. interval <= 0 falls back to the 3s default.
	result, err := c.WaitForResult(ctx, task.ID, 3*time.Second)
	if err != nil {
		log.Fatalf("poll: %v", err)
	}
	if result.TaskStatus != client.TaskStatusSuccess {
		log.Fatalf("task ended in status %s", result.TaskStatus)
	}

	for i, img := range result.Data {
		fmt.Printf("image[%d]: %s\n", i, img.URL)
	}
}
