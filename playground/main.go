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

// Load API key from environment
func getAPIKey() string {
	apiKey := os.Getenv("ZAI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("KEY")
	}
	return apiKey
}

// Make authenticated API request
func makeRequest(endpoint string) (map[string]interface{}, error) {
	apiKey := getAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("no API key found")
	}

	baseURL := "https://api.z.ai/api/paas/v4"
	url := baseURL + endpoint

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Add status code to result
	result["_status_code"] = resp.StatusCode
	result["_endpoint"] = endpoint

	return result, nil
}

// Test multiple endpoints
func exploreEndpoints() {
	fmt.Println("🔍 Exploring Z.AI API endpoints for quota/limits information...\n")

	endpoints := []string{
		"/monitor/usage/quota/limit",
		"/usage/quota",
		"/quota",
		"/limits",
		"/account/quota",
		"/account/usage",
		"/account/info",
		"/account",
		"/billing/quota",
		"/billing/usage",
		"/billing",
		"/user/quota",
		"/user/usage",
		"/user/info",
		"/user",
		"/monitor/quota",
		"/monitor/usage",
		"/stats/quota",
		"/stats/usage",
		"/me/quota",
		"/me/usage",
	}

	for _, endpoint := range endpoints {
		fmt.Printf("Testing: %s\n", endpoint)
		result, err := makeRequest(endpoint)
		if err != nil {
			fmt.Printf("  ❌ Error: %v\n\n", err)
			continue
		}

		statusCode := int(result["_status_code"].(float64))
		delete(result, "_status_code")
		delete(result, "_endpoint")

		if statusCode == 200 {
			fmt.Printf("  ✅ SUCCESS (200)\n")
			prettyPrint(result)
		} else if statusCode == 404 {
			fmt.Printf("  ⚠️  NOT FOUND (404)\n")
			if errMsg, ok := result["error"].(map[string]interface{}); ok {
				fmt.Printf("     Message: %v\n", errMsg)
			}
		} else if statusCode == 401 {
			fmt.Printf("  🔐 UNAUTHORIZED (401)\n")
		} else if statusCode == 429 {
			fmt.Printf("  🚫 RATE LIMITED (429)\n")
			prettyPrint(result)
		} else {
			fmt.Printf("  ❓ STATUS %d\n", statusCode)
			prettyPrint(result)
		}
		fmt.Println()
	}
}

// Test chat completion to see rate limit headers
func testRateLimitHeaders() {
	fmt.Println("🔍 Testing rate limit headers...\n")

	apiKey := getAPIKey()
	if apiKey == "" {
		fmt.Println("No API key found")
		return
	}

	url := "https://api.z.ai/api/paas/v4/chat/completions"

	payload := map[string]interface{}{
		"model":    "glm-4.5",
		"messages": []map[string]string{{"role": "user", "content": "test"}},
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Response Headers:")
	for key, values := range resp.Header {
		for _, value := range values {
			if contains(key, "rate") || contains(key, "limit") || contains(key, "quota") || contains(key, "remaining") {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
	}

	fmt.Printf("\nStatus Code: %d\n", resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response: %s\n", string(body))
}

// Test account info endpoint
func testAccountInfo() {
	fmt.Println("🔍 Testing account information...\n")

	endpoints := []string{
		"/account/info",
		"/account",
		"/user/info",
		"/user",
		"/me",
	}

	for _, endpoint := range endpoints {
		fmt.Printf("Testing: %s\n", endpoint)
		result, err := makeRequest(endpoint)
		if err != nil {
			fmt.Printf("  ❌ Error: %v\n\n", err)
			continue
		}

		statusCode := int(result["_status_code"].(float64))
		delete(result, "_status_code")
		delete(result, "_endpoint")

		if statusCode == 200 {
			fmt.Printf("  ✅ SUCCESS (200)\n")
			prettyPrint(result)
		} else {
			fmt.Printf("  ❌ Status %d\n", statusCode)
			prettyPrint(result)
		}
		fmt.Println()
	}
}

func main() {
	fmt.Println("🎮 Z.AI API Playground - Exploring Quotas & Limits")
	fmt.Println("=" + string(makeTerm(70)))
	fmt.Println()

	// Test various endpoints for quota/usage information
	exploreEndpoints()

	// Test account info
	testAccountInfo()

	// Test rate limit headers
	testRateLimitHeaders()
}

func prettyPrint(data map[string]interface{}) {
	indent := "    "
	printMap(data, indent)
}

func printMap(data map[string]interface{}, indent string) {
	for key, value := range data {
		switch v := value.(type) {
		case map[string]interface{}:
			fmt.Printf("%s%s:\n", indent, key)
			printMap(v, indent+"  ")
		case []interface{}:
			fmt.Printf("%s%s: []interface{} with %d items\n", indent, key, len(v))
		case string:
			fmt.Printf("%s%s: %s\n", indent, key, v)
		case float64:
			fmt.Printf("%s%s: %.0f\n", indent, key, v)
		case bool:
			fmt.Printf("%s%s: %t\n", indent, key, v)
		default:
			fmt.Printf("%s%s: %v\n", indent, key, v)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func makeTerm(length int) []byte {
	result := make([]byte, length)
	for i := range result {
		result[i] = '='
	}
	return result
}