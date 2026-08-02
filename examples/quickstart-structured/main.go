// Command quickstart-structured shows structured (JSON Schema) output: ask
// the model for typed data, parse the response into a Go struct. Useful as
// the building block for extraction pipelines.
//
// Usage:
//
//	export ZAI_API_KEY=your_api_key_here
//	go run ./examples/quickstart-structured "Marie Curie, born 1867, won Nobel prizes in physics and chemistry"
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

// Person is the typed shape we want the model to fill. The JSON Schema we
// send to the API is hand-built to match; in a real app you'd generate it
// from the struct via reflection (an instructor-go-style helper is on the
// roadmap — see ops/objectives-and-opportunities.md §2.3).
type Person struct {
	Name       string `json:"name"`
	BirthYear  int    `json:"birth_year"`
	NotableFor string `json:"notable_for"`
	NobelYears []int  `json:"nobel_years,omitempty"`
}

func main() {
	subject := "Marie Curie, born 1867, won Nobel prizes in physics (1903) and chemistry (1911)"
	if len(os.Args) > 1 {
		subject = os.Args[1]
	}

	c, err := client.NewClientFromEnv()
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	resp, err := c.Chat().Create(context.Background(), client.ChatRequest{
		Model: "glm-5.2",
		Messages: []client.Message{
			{Role: "system", Content: "Extract a structured person record from the user's description."},
			{Role: "user", Content: subject},
		},
		ResponseFormat: client.NewJSONSchemaFormat(
			"person",
			json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"birth_year":{"type":"integer"},"notable_for":{"type":"string"},"nobel_years":{"type":"array","items":{"type":"integer"}}},"required":["name","birth_year","notable_for"]}`),
			true, // strict
		),
	})
	if err != nil {
		log.Fatalf("chat: %v", err)
	}

	var p Person
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &p); err != nil {
		log.Fatalf("parse structured response: %v", err)
	}
	out, _ := json.MarshalIndent(p, "", "  ")
	fmt.Println(string(out))
}
