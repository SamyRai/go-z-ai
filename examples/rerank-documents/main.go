// Command rerank-documents ranks candidate documents against a query using
// GLM's rerank API — typically the second stage of a RAG pipeline after
// embedding-based retrieval (see embeddings-batch). Returns documents in
// descending relevance with scores.
//
// Usage:
//
//	export ZAI_API_KEY=your_api_key_here
//	go run ./examples/rerank-documents "how do I stream tokens?"
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

func main() {
	query := "how do I stream tokens?"
	if len(os.Args) > 1 {
		query = os.Args[1]
	}

	c, err := client.NewClientFromEnv()
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	candidates := []string{
		"The Chat().Stream method returns an iter.Seq2 you range over for token-by-token output.",
		"Voice cloning uses a zero-shot model to copy a speaker's voice from a short sample.",
		"Embeddings map text into a vector space where semantic similarity is cosine distance.",
		"Set Config.MaxRetries to -1 to disable retry entirely on transient failures.",
	}

	resp, err := c.Rerank().Create(context.Background(), client.RerankRequest{
		Model:           "rerank",
		Query:           query,
		Documents:       candidates,
		ReturnDocuments: true,
	})
	if err != nil {
		log.Fatalf("rerank: %v", err)
	}

	// Results come back with index + relevance_score, referencing the input
	// slice by position. Sort descending by score for display.
	results := append([]client.RerankResult(nil), resp.Results...)
	sort.Slice(results, func(i, j int) bool { return results[i].RelevanceScore > results[j].RelevanceScore })

	fmt.Printf("Query: %s\n\n", query)
	for i, r := range results {
		doc := r.Document
		if doc == "" && r.Index >= 0 && r.Index < len(candidates) {
			doc = candidates[r.Index]
		}
		fmt.Printf("%d. [%.3f] %s\n", i+1, r.RelevanceScore, doc)
	}
}
