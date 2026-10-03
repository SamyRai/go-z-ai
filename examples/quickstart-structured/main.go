// Command quickstart-structured shows structured output: JSON-object mode
// plus a JSON Schema in the system prompt (the API has no json_schema
// response format), parsed into a Go struct. Useful as the building block
// for extraction pipelines.
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

// Person is the typed shape we want the model to fill. personSchema is
// hand-built to match; in a real app you'd generate it from the struct.
type Person struct {
	Name       string `json:"name"`
	BirthYear  int    `json:"birth_year"`
	NotableFor string `json:"notable_for"`
	NobelYears []int  `json:"nobel_years,omitempty"`
}

const personSchema = `{"type":"object","properties":{"name":{"type":"string"},"birth_year":{"type":"integer"},"notable_for":{"type":"string"},"nobel_years":{"type":"array","items":{"type":"integer"}}},"required":["name","birth_year","notable_for"]}`

func main() {
	subject := "Marie Curie, born 1867, won Nobel prizes in physics (1903) and chemistry (1911)"
	if len(os.Args) > 1 {
		subject = os.Args[1]
	}

	c, err := client.NewClientFromEnv()
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	instruction, err := client.JSONSchemaPrompt(json.RawMessage(personSchema))
	if err != nil {
		log.Fatalf("schema: %v", err)
	}
	resp, err := c.Chat().Create(context.Background(), client.ChatRequest{
		Model: client.DefaultModel,
		Messages: []client.Message{
			{Role: "system", Content: "Extract a structured person record from the user's description.\n\n" + instruction},
			{Role: "user", Content: subject},
		},
		ResponseFormat: client.JSONObjectFormat(),
	})
	if err != nil {
		log.Fatalf("chat: %v", err)
	}

	var p Person
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &p); err != nil {
		log.Fatalf("parse structured response: %v", err)
	}
	out, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		log.Fatalf("marshal: %v", err)
	}
	fmt.Println(string(out))
}
