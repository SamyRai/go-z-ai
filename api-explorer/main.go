package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	apiKey := os.Getenv("KEY")
	if apiKey == "" {
		apiKey = os.Getenv("ZAI_API_KEY")
	}

	// Try to load from .env file in parent directory
	if apiKey == "" {
		apiKey = loadAPIKeyFromEnv()
	}

	if apiKey == "" {
		fmt.Println("❌ No API key found in KEY, ZAI_API_KEY, or .env file")
		fmt.Println("💡 Set KEY or ZAI_API_KEY environment variable, or create .env file")
		return
	}

	fmt.Println("🔍 Z.AI API Explorer - Learning About Quotas & Limits")
	fmt.Println("=" + makeLine(70))
	fmt.Println()

	baseURL := "https://api.z.ai/api/paas/v4"

	// Test different endpoints to find quota/usage information
	endpoints := map[string]string{
		"Known Working - Models":    "/models",
		"Known Working - Chat":     "/chat/completions",
		"Monitor Usage Quota":      "/monitor/usage/quota/limit",
		"Usage Quota":              "/usage/quota",
		"Account Info":             "/account/info",
		"User Info":                "/user/info",
		"Billing Info":             "/billing/info",
		"Quota":                    "/quota",
		"Limits":                   "/limits",
		"Usage":                    "/usage",
		"Account Usage":            "/account/usage",
		"User Quota":               "/user/quota",
		"Monitor Usage":            "/monitor/usage",
		"Stats Quota":              "/stats/quota",
		"Me Usage":                 "/me/usage",
		"Embeddings":               "/embeddings",
		"Files":                    "/files",
		"Assistants":               "/assistants",
		"Threads":                  "/threads",
		"Batches":                  "/batches",
		"Finetuning":               "/fine_tuning/jobs",
		"Images":                   "/images/generations",
		"Audio":                    "/audio/transcriptions",
		"Moderations":              "/moderations",
	}

	for name, endpoint := range endpoints {
		fmt.Printf("🧪 Testing: %s (%s)\n", name, endpoint)
		testEndpoint(baseURL, endpoint, apiKey)
		fmt.Println()
	}

	// Test rate limit headers on a simple request
	fmt.Println("🧪 Testing Rate Limit Headers")
	testRateLimitHeaders(baseURL, apiKey)
}

func testEndpoint(baseURL, endpoint, apiKey string) {
	url := baseURL + endpoint

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("  ❌ Failed to create request: %v\n", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("  ❌ Request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("  ❌ Failed to read response: %v\n", err)
		return
	}

	// Parse JSON response
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		// Not JSON, show raw response
		fmt.Printf("  📄 Status: %d\n", resp.StatusCode)
		fmt.Printf("  📄 Response: %s\n", string(body))
		return
	}

	// Show formatted response
	switch resp.StatusCode {
	case 200:
		fmt.Printf("  ✅ SUCCESS (200)\n")
		printJSON(jsonData)
	case 404:
		fmt.Printf("  ⚠️  NOT FOUND (404) - Endpoint doesn't exist\n")
		printJSON(jsonData)
	case 401:
		fmt.Printf("  🔐 UNAUTHORIZED (401) - API key issue\n")
		printJSON(jsonData)
	case 429:
		fmt.Printf("  🚫 RATE LIMITED (429)\n")
		printJSON(jsonData)
	default:
		fmt.Printf("  ❓ Status: %d\n", resp.StatusCode)
		printJSON(jsonData)
	}

	// Show interesting headers
	showInterestingHeaders(resp.Header)
}

func testRateLimitHeaders(baseURL, apiKey string) {
	url := baseURL + "/chat/completions"

	payload := map[string]interface{}{
		"model":    "glm-4.5",
		"messages": []map[string]string{{"role": "user", "content": "test"}},
	}

	jsonData, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		fmt.Printf("  ❌ Failed to create request: %v\n", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("  ❌ Request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Show rate limit related headers
	fmt.Println("  📋 Rate Limit Headers:")
	for key, values := range resp.Header {
		if isRateLimitHeader(key) {
			for _, value := range values {
				fmt.Printf("     %s: %s\n", key, value)
			}
		}
	}

	fmt.Printf("  📄 Status: %d\n", resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	if len(body) > 0 {
		fmt.Printf("  📄 Response: %s\n", string(body[:min(200, len(body))]))
	}
}

func printJSON(data interface{}) {
	prettyJSON, _ := json.MarshalIndent(data, "     ", "  ")
	fmt.Printf("     %s\n", string(prettyJSON))
}

func showInterestingHeaders(header http.Header) {
	rateLimitHeaders := []string{
		"X-RateLimit-Limit",
		"X-RateLimit-Remaining",
		"X-RateLimit-Reset",
		"X-RateLimit-Used",
		"X-Quota-Limit",
		"X-Quota-Remaining",
		"X-Quota-Reset",
		"X-Usage-Limit",
		"X-Usage-Remaining",
		"X-RateLimit-Limit-Second",
		"X-RateLimit-Limit-Minute",
		"X-RateLimit-Limit-Hour",
		"X-RateLimit-Limit-Day",
		"Retry-After",
	}

	for _, key := range rateLimitHeaders {
		if values := header.Values(key); len(values) > 0 {
			for _, value := range values {
				fmt.Printf("  📋 %s: %s\n", key, value)
			}
		}
	}
}

func isRateLimitHeader(key string) bool {
	rateLimitKeywords := []string{
		"rate", "limit", "quota", "usage", "remaining", "reset", "retry",
		"X-Rate", "X-Quota", "X-Usage", "X-Limit",
	}

	lowerKey := toLower(key)
	for _, keyword := range rateLimitKeywords {
		if contains(lowerKey, toLower(keyword)) {
			return true
		}
	}
	return false
}

func makeLine(length int) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = '='
	}
	return string(result)
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			result[i] = s[i] + 32
		} else {
			result[i] = s[i]
		}
	}
	return string(result)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func loadAPIKeyFromEnv() string {
	// Try to read .env file from parent directory
	data, err := os.ReadFile("../.env")
	if err != nil {
		return ""
	}

	// Parse simple .env file (KEY=value or ZAI_API_KEY=value)
	lines := bytes.Split(data, []byte{'\n'})
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		parts := bytes.SplitN(line, []byte{'='}, 2)
		if len(parts) == 2 {
			key := string(bytes.TrimSpace(parts[0]))
			value := string(bytes.TrimSpace(parts[1]))

			if key == "KEY" || key == "ZAI_API_KEY" {
				return value
			}
		}
	}

	return ""
}