// Command embeddings-batch generates embeddings for a list of texts and
// prints cosine-similarity scores against a query. The building block of a
// semantic-search or RAG pipeline — embeddings rank candidate chunks, then
// rerank (see rerank-documents) sharpens the order.
//
// Usage:
//
//	export ZAI_API_KEY=your_api_key_here
//	go run ./examples/embeddings-batch "how do I stream tokens?"
package main

import (
	"context"
	"fmt"
	"log"
	"math"
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
		"The batch endpoint accepts JSONL and runs at roughly half the cost of realtime.",
	}

	// Embed query + candidates in one call (the API accepts a list of inputs).
	all := append([]string{query}, candidates...)
	resp, err := c.Embeddings().Create(context.Background(), client.EmbeddingsRequest{
		Model: "embedding-3",
		Input: all,
	})
	if err != nil {
		log.Fatalf("embeddings: %v", err)
	}
	if len(resp.Data) != len(all) {
		log.Fatalf("expected %d embeddings, got %d", len(all), len(resp.Data))
	}

	queryVec := resp.Data[0].Embedding
	type pair struct {
		text  string
		score float64
	}
	results := make([]pair, len(candidates))
	for i, cand := range resp.Data[1:] {
		results[i] = pair{text: candidates[i], score: cosine(queryVec, cand.Embedding)}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].score > results[j].score })

	fmt.Printf("Query: %s\n\n", query)
	for i, r := range results {
		fmt.Printf("%d. [%.3f] %s\n", i+1, r.score, r.text)
	}
}

// cosine is the standard cosine similarity between two equal-length vectors.
func cosine(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
