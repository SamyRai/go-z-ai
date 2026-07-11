# Z.AI API Discovery Results

## 🔍 Research Summary

Extensive API exploration revealed that **Z.AI does not provide usage/quota monitoring endpoints** in their main API (`/api/paas/v4`).

## ❌ Non-Existent Endpoints (404)

All tested quota/usage/account endpoints returned 404:
- `/monitor/usage/quota/limit`
- `/usage/quota`, `/quota`, `/usage`
- `/account/info`, `/account/usage`
- `/user/info`, `/user/quota`
- `/billing/info`
- `/limits`
- `/stats/quota`
- `/me/usage`

## ✅ Working Endpoints (200)

### Core AI Operations
- **`/models`** - List available models ✅
- **`/chat/completions`** - Chat completions (POST only) ✅
- **`/embeddings`** - Text embeddings (POST only) ✅
- **`/images/generations`** - Image generation (POST only) ✅
- **`/audio/transcriptions`** - Audio transcription (POST only) ✅
- **`/moderations`** - Content moderation (POST only) ✅

### Management Operations
- **`/files`** - File management ✅
- **`/fine_tuning/jobs`** - Fine-tuning jobs ✅
- **`/batches`** - Batch operations ✅

## 💰 Account Status

**Current Account State:**
```
Error Code: 1113
Message: "Insufficient balance or no resource package. Please recharge."
```

## 🔑 Key Learnings

### 1. **API Architecture**
- Z.AI separates **AI functionality** from **account management**
- Main API focuses purely on model operations
- Usage monitoring is NOT part of the core API

### 2. **Rate Limiting**
- **No rate limit headers** returned in responses
- Balance checking happens at request time
- **No traditional rate limiting** apparent

### 3. **Account Types**
- **Pay-as-you-go**: Requires balance pre-load
- **Subscription Plans**: Different access patterns
- **Enterprise**: Potentially different endpoints

## 📊 Usage Monitoring Alternatives

Since API endpoints don't exist, usage must be monitored via:

### 1. **Web Dashboard**
- Visit https://z.ai
- Manual quota/balance checking
- Real-time usage statistics

### 2. **Separate Management API**
- Potentially different API service
- Account management endpoints
- Billing integration points

### 3. **Client-Side Tracking**
- Implement local usage tracking
- Log all API requests
- Calculate costs based on model pricing

## 🚨 Client Application Updates Required

### Remove Non-Existent Services
```go
// ❌ These services call non-existent endpoints:
- usage.GetQuota()      // 404
- usage.GetAccountInfo() // 404
- usage.GetBillingInfo()  // 404
- usage.GetUsageSummary() // 404
```

### Replace With Working Alternatives
```go
// ✅ Keep these working services:
- models.List()    // ✅ Works
- chat.Create()    // ✅ Works
- files.List()     // ✅ Works
```

## 🎯 Recommended Next Steps

1. **Update Client Code**: Remove non-existent endpoints
2. **Implement Client-Side Usage Tracking**: Log requests and calculate costs
3. **Document Account Management**: Guide users to web dashboard
4. **Error Handling**: Properly handle 1113 (insufficient balance) errors
5. **Account Type Detection**: Different handling for subscription vs pay-as-you-go

## 🔧 Technical Implementation

### Client-Side Usage Tracking
```go
type UsageTracker struct {
    requests map[string]int
    tokens   map[string]int64
    costs    map[string]float64
}

func (t *UsageTracker) LogRequest(model string, promptTokens, completionTokens int) {
    // Calculate cost based on model pricing
    // Track local usage
    // Provide summary methods
}
```

### Balance Check Wrapper
```go
func CheckBalanceBeforeRequest(client *Client) error {
    // Make a minimal API call
    // If 1113 error, alert user
    // Provide guidance to recharge
}
```

## 📝 Conclusion

The Z.AI API is designed differently than expected:
- **Core API**: Pure AI model operations
- **Management**: Separate service/web dashboard
- **Usage Monitoring**: Not available via API

This requires a fundamental change in how the Go client handles usage monitoring and account management.