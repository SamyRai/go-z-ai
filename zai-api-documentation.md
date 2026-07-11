# Z.AI (Zhipu AI) API Documentation

> Complete API reference for Z.AI (Zhipu AI) platform - Last updated: July 2026

## Table of Contents
- [Overview](#overview)
- [Authentication](#authentication)
- [API Endpoints](#api-endpoints)
- [Available Models](#available-models)
- [Chat Completion API](#chat-completion-api)
- [Pricing](#pricing)
- [Usage Limits & Quotas](#usage-limits--quotas)
- [SDKs & Integration](#sdks--integration)
- [Account Management](#account-management)
- [Error Codes](#error-codes)
- [Resources](#resources)

---

## Overview

Z.AI (Zhipu AI) provides a comprehensive AI platform featuring the GLM (General Language Model) series, with GLM-5.2 as their flagship model. The platform offers RESTful APIs that support multiple programming languages and development environments.

### Key Features
- **Large Context Windows**: Up to 1M+ tokens depending on model
- **Multimodal Support**: Text, images, audio, video, and file inputs
- **Function Calling**: Tool use and agent capabilities
- **Streaming Support**: Real-time response generation
- **OpenAI-Compatible**: Can use OpenAI SDKs with base URL change

### Official Documentation
- **Developer Docs**: [https://docs.z.ai](https://docs.z.ai)
- **API Platform**: [https://z.ai/model-api](https://z.ai/model-api)
- **GitHub Repository**: [https://github.com/zai-org/z-ai-sdk-python](https://github.com/zai-org/z-ai-sdk-python)

---

## Authentication

### API Key Authentication

Z.AI uses standard HTTP Bearer authentication. You need an API key, which can be created and managed in the [API Keys Page](https://z.ai/manage-apikey).

**Header Format:**
```http
Authorization: Bearer YOUR_API_KEY
```

**Example Request:**
```bash
curl -X POST "https://api.z.ai/api/paas/v4/chat/completions" \
  -H "Content-Type: application/json" \
  -H "Accept-Language: en-US,en" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "glm-5.2",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'
```

### API Key Management
- Create and manage keys at: [https://z.ai/manage-apikey](https://z.ai/manage-apikey)
- Rate limits information: [https://z.ai/manage-apikey/rate-limits](https://z.ai/manage-apikey/rate-limits)
- Keys are scoped to specific product types/scenarios
- Different keys may be required for different service tiers

---

## API Endpoints

### 1. General API Endpoint (Pay-as-you-go)
```
https://api.z.ai/api/paas/v4
```
**Usage**: Standard API access with pay-per-use pricing

### 2. GLM Coding Plan Endpoint
```
https://api.z.ai/api/coding/paas/v4
```
**Usage**: Dedicated endpoint for GLM Coding Plan subscribers
**Note**: Requires specific base URL configuration

### 3. Main API Endpoints

#### Chat Completions
```
POST /api/paas/v4/chat/completions
```
Creates AI replies for conversation messages with multimodal support.

#### Usage Monitoring
```
GET /api/monitor/usage/quota/limit
```
Check current token quota and usage statistics.

---

## Available Models

### Text Models (Flagship)

| Model | Input | Cached Input | Cached Storage | Output | Max Output |
|-------|-------|--------------|----------------|--------|------------|
| **GLM-5.2** | $1.40/M | $0.26/M | Limited-time Free | $4.40/M | 128K tokens |
| **GLM-5.1** | $1.40/M | $0.26/M | Limited-time Free | $4.40/M | 128K tokens |
| **GLM-5** | $1.00/M | $0.20/M | Limited-time Free | $3.20/M | 128K tokens |
| **GLM-5-Turbo** | $1.20/M | $0.24/M | Limited-time Free | $4.00/M | 128K tokens |

### Text Models (Standard)

| Model | Input | Cached Input | Cached Storage | Output | Max Output |
|-------|-------|--------------|----------------|--------|------------|
| **GLM-4.7** | $0.60/M | $0.11/M | Limited-time Free | $2.20/M | 128K tokens |
| **GLM-4.6** | $0.60/M | $0.11/M | Limited-time Free | $2.20/M | 128K tokens |
| **GLM-4.5** | $0.60/M | $0.11/M | Limited-time Free | $2.20/M | 96K tokens |
| **GLM-4.5-X** | $2.20/M | $0.45/M | Limited-time Free | $8.90/M | 96K tokens |
| **GLM-4.5-Air** | $0.20/M | $0.03/M | Limited-time Free | $1.10/M | 96K tokens |
| **GLM-4.5-AirX** | $1.10/M | $0.22/M | Limited-time Free | $4.50/M | 96K tokens |

### Flash Models (Free/High-Speed)

| Model | Input | Output | Notes |
|-------|-------|--------|-------|
| **GLM-4.7-Flash** | Free | Free | Limited-time free |
| **GLM-4.7-FlashX** | $0.07/M | $0.40/M | FlashX variant |
| **GLM-4.5-Flash** | Free | Free | Limited-time free |

### Vision Models

| Model | Input | Cached Input | Cached Storage | Output | Max Output |
|-------|-------|--------------|----------------|--------|------------|
| **GLM-5V-Turbo** | $1.20/M | $0.24/M | Limited-time Free | $4.00/M | 32K tokens |
| **GLM-4.6V** | $0.30/M | $0.05/M | Limited-time Free | $0.90/M | 32K tokens |
| **GLM-4.6V-FlashX** | $0.04/M | $0.004/M | Limited-time Free | $0.40/M | 32K tokens |
| **GLM-4.6V-Flash** | Free | Free | Free | Free | 32K tokens |
| **GLM-4.5V** | $0.60/M | $0.11/M | Limited-time Free | $1.80/M | 16K tokens |
| **GLM-OCR** | $0.03/M | - | - | $0.03/M | Specialized |

### Specialized Models

**Image Generation:**
- **GLM-Image**: $0.015/image
- **CogView-4**: $0.01/image

**Video Generation:**
- **CogVideoX-3**: $0.2/video
- **ViduQ1-Text**: $0.4/video
- **ViduQ1-Image**: $0.4/video
- **Vidu2-Image**: $0.2/video
- **Vidu2-Start-End**: $0.2/video
- **Vidu2-Reference**: $0.4/video

**Audio Models:**
- **GLM-ASR-2512**: $0.03/MTok (~$0.0024/minute)

**Agents:**
- **GLM Slide/Poster Agent (beta)**: $0.7/MTok
- **General-Purpose Translation**: $3/MTok

---

## Chat Completion API

### Endpoint
```
POST https://api.z.ai/api/paas/v4/chat/completions
```

### Request Parameters

#### Core Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `model` | string | Yes | `glm-5.2` | Model code to use |
| `messages` | array | Yes | - | Conversation messages |
| `temperature` | number | No | 1.0 | Sampling temperature [0.0-1.0] |
| `top_p` | number | No | 0.95 | Nucleus sampling [0.01-1.0] |
| `max_tokens` | integer | No | Varies | Maximum output tokens |
| `stream` | boolean | No | false | Enable streaming |
| `do_sample` | boolean | No | true | Enable sampling strategy |

#### Message Format

```json
{
  "role": "user|system|assistant|tool",
  "content": "message content"
}
```

**Message Types:**
- `system`: System instructions
- `user`: User messages
- `assistant`: AI responses
- `tool`: Tool call results

#### Advanced Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `thinking` | object | Control chain-of-thought reasoning |
| `thinking.type` | string | `enabled`/`disabled` - controls CoT |
| `thinking.preserved` | boolean | Keep reasoning across turns |
| `thinking.effort` | string | `max`/`high`/`medium`/`low`/`minimal`/`none` |
| `tools` | array | List of available tools/functions |
| `tool_choice` | string | Tool selection strategy |
| `stop` | array | Stop sequences (max 4) |
| `response_format` | object | Output format control |

### Response Format

```json
{
  "id": "chat-123456789",
  "created": 1234567890,
  "model": "glm-5.2",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Response text",
        "reasoning_content": "Optional reasoning",
        "tool_calls": []
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 20,
    "total_tokens": 30,
    "prompt_tokens_details": {
      "cached_tokens": 5
    }
  }
}
```

### Finish Reasons
- `stop`: Normal completion
- `tool_calls`: Tool function call generated
- `length`: Max tokens reached
- `sensitive`: Content filter triggered
- `model_context_window_exceeded`: Context overflow
- `network_error`: Network issue

---

## Pricing

### Pricing Structure (per 1M tokens)

**Flagship Models:**
- GLM-5.2: Input $1.40, Cached $0.26, Output $4.40
- GLM-5.1: Input $1.40, Cached $0.26, Output $4.40
- GLM-5: Input $1.00, Cached $0.20, Output $3.20
- GLM-5-Turbo: Input $1.20, Cached $0.24, Output $4.00

**Standard Models:**
- GLM-4.7/4.6/4.5: Input $0.60, Cached $0.11, Output $2.20
- GLM-4.5-Air: Input $0.20, Cached $0.03, Output $1.10
- GLM-4.5-X: Input $2.20, Cached $0.45, Output $8.90

**Flash Models:**
- GLM-4.7-Flash: FREE
- GLM-4.5-Flash: FREE
- GLM-4.7-FlashX: Input $0.07, Output $0.40

### Built-in Tools Pricing
- **Web Search**: $0.01/use

### GLM Coding Plans (Subscription)

| Plan | Monthly | Yearly (2nd+) | Features |
|------|---------|---------------|----------|
| **Lite** | $12.60 ($18) | $151.20 | Basic coding |
| **Pro** | $50.40 ($72) | $604.80 | Popular choice |
| **Max** | $112.00 ($160) | - | Maximum usage |

**Note**: 30% promotional pricing available through September 2026

---

## Usage Limits & Quotas

### Quota System

Z.AI implements usage limits on two levels:

1. **5-Hour Rolling Window**: Short-term usage limits
2. **Weekly Quota**: Long-term usage caps starting from order date

### GLM Coding Plan Limits

**Reported Usage Multipliers:**
- Peak hours: 3× quota consumption
- Off-peak hours: 2× quota consumption

**Typical Limits (by plan):**
- Lite: ~120 prompts per 5-hour cycle
- Pro: ~600 prompts per 5-hour cycle
- Max: Higher limits (specific amount varies)

### Quota Monitoring

**API Endpoint:**
```
GET /api/monitor/usage/quota/limit
```

**Third-Party Tools:**
- [VS Code Usage Tracker](https://github.com/melon-hub/zai-usage-tracker)
- [OpenCode Quota Monitor](https://github.com/guyinwonder168/opencode-glm-quota)

### Rate Limiting

- Configured per API key
- Different limits for different plan tiers
- Check specific limits at: [https://z.ai/manage-apikey/rate-limits](https://z.ai/manage-apikey/rate-limits)

---

## SDKs & Integration

### Official Python SDK

**Installation:**
```bash
pip install zai-sdk
# Or specific version
pip install zai-sdk==0.2.3
```

**Usage:**
```python
from zai import ZaiClient

client = ZaiClient(api_key="YOUR_API_KEY")

response = client.chat.completions.create(
    model="glm-5.2",
    messages=[
        {"role": "system", "content": "You are a helpful AI assistant."},
        {"role": "user", "content": "Hello!"}
    ]
)

print(response.choices[0].message.content)
```

### Official Java SDK

**Maven:**
```xml
<dependency>
    <groupId>ai.z.openapi</groupId>
    <artifactId>zai-sdk</artifactId>
    <version>0.3.5</version>
</dependency>
```

**Gradle:**
```groovy
implementation 'ai.z.openapi:zai-sdk:0.3.5'
```

### OpenAI-Compatible SDKs

**Python:**
```python
from openai import OpenAI

client = OpenAI(
    api_key="your-Z.AI-api-key",
    base_url="https://api.z.ai/api/paas/v4/"
)

completion = client.chat.completions.create(
    model="glm-5.2",
    messages=[
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "Hello!"}
    ]
)
```

**Node.js:**
```javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: "your-Z.AI-api-key",
  baseURL: "https://api.z.ai/api/paas/v4/"
});

const completion = await client.chat.completions.create({
  model: "glm-5.2",
  messages: [
    { role: "system", content: "You are a helpful assistant." },
    { role: "user", "content: "Hello!" }
  ]
});
```

---

## Account Management

### API Key Management

**Create/Manage Keys:**
[https://z.ai/manage-apikey](https://z.ai/manage-apikey)

**Key Types:**
- General API keys (pay-as-you-go)
- GLM Coding Plan keys (subscription)
- Enterprise keys (custom limits)

### Subscription Management

**Official Subscription Page:**
[https://z.ai/subscribe](https://z.ai/subscribe)

**Plan Comparison:**
- Lite: Basic usage, budget-friendly
- Pro: Balanced usage, popular choice
- Max: Heavy usage, maximum limits

### Usage Tracking

**Tools:**
1. Official dashboard at z.ai
2. [VS Code Extension](https://github.com/melon-hub/zai-usage-tracker)
3. [OpenCode Monitor](https://github.com/guyinwonder168/opencode-glm-quota)
4. API endpoint: `/api/monitor/usage/quota/limit`

---

## Error Codes

### Common Error Responses

**Finish Reasons:**
- `stop`: Normal completion
- `tool_calls`: Function call required
- `length`: Token limit exceeded
- `sensitive`: Content policy violation
- `model_context_window_exceeded`: Input too long
- `network_error`: Connection issues

### HTTP Status Codes

- `200`: Success
- `400`: Bad request (invalid parameters)
- `401`: Unauthorized (invalid API key)
- `403`: Forbidden (insufficient permissions)
- `429`: Rate limit exceeded
- `500`: Internal server error

**Error Documentation:**
[https://docs.z.ai/api-reference/api-code](https://docs.z.ai/api-reference/api-code)

---

## Resources

### Official Documentation
- **API Reference**: [https://docs.z.ai/api-reference/introduction](https://docs.z.ai/api-reference/introduction)
- **Quick Start**: [https://docs.z.ai/guides/overview/quick-start](https://docs.z.ai/guides/overview/quick-start)
- **HTTP API Guide**: [https://docs.z.ai/guides/develop/http/introduction](https://docs.z.ai/guides/develop/http/introduction)
- **Pricing**: [https://docs.z.ai/guides/overview/pricing](https://docs.z.ai/guides/overview/pricing)
- **FAQ**: [https://docs.z.ai/devpack/faq](https://docs.z.ai/devpack/faq)
- **GLM Coding Plan**: [https://docs.z.ai/devpack/overview](https://docs.z.ai/devpack/overview)

### Community Resources
- **Python SDK**: [https://github.com/zai-org/z-ai-sdk-python](https://github.com/zai-org/z-ai-sdk-python)
- **Reddit Community**: [https://www.reddit.com/r/ZaiGLM](https://www.reddit.com/r/ZaiGLM)
- **VS Code Usage Tracker**: [https://github.com/melon-hub/zai-usage-tracker](https://github.com/melon-hub/zai-usage-tracker)

### Third-Party Providers
- **OpenRouter**: [https://openrouter.ai/z-ai](https://openrouter.ai/z-ai)
- **Together AI**: GLM-5.2 available
- **LiteLLM Integration**: [https://docs.litellm.ai/docs/providers/zai](https://docs.litellm.ai/docs/providers/zai)

---

## OpenAPI Schema

⚠️ **Note**: As of the documentation date, Z.AI does not provide a publicly accessible OpenAPI/JSON schema specification for download.

**Potential OpenAPI endpoints to try:**
- `https://api.z.ai/openapi.json`
- `https://api.z.ai/swagger.json`
- `https://openapi.z.ai/openapi.json`

**Recommendations:**
1. Use the official API documentation at docs.z.ai
2. Contact Z.AI support for OpenAPI specification
3. Create custom OpenAPI spec from documentation if needed
4. Monitor official GitHub repository for schema updates

---

## Best Practices

### Optimization
1. **Use Cached Inputs**: Enable prompt caching for repeated contexts
2. **Model Selection**: Use GLM-4.7 for routine tasks to save quota
3. **Streaming**: Enable streaming for better user experience
4. **Batch Requests**: Combine multiple requests when possible

### Cost Management
1. **Monitor Usage**: Track quota regularly
2. **Choose Right Plan**: Select subscription based on usage patterns
3. **Use Flash Models**: Leverage free flash models when appropriate
4. **Optimize Prompts**: Reduce token usage through efficient prompting

### Integration Tips
1. **Error Handling**: Implement proper error handling and retry logic
2. **Rate Limiting**: Respect rate limits to avoid account restrictions
3. **API Key Security**: Never expose API keys in client-side code
4. **Testing**: Use free flash models for development and testing

---

## Changelog

### 2026 Updates
- **December 2, 2026**: Usage limits updated for coding plans
- **2026**: GLM-5.2 released with enhanced capabilities
- **2026**: Cached input storage limited-time free promotion

### Recent Model Releases
- GLM-5.2 (flagship model)
- GLM-4.7 series
- Flash model variants
- Enhanced vision models

---

## Support

### Official Support Channels
- **Documentation**: [https://docs.z.ai](https://docs.z.ai)
- **API Platform**: [https://z.ai/model-api](https://z.ai/model-api)
- **Status Page**: Check official status for service updates

### Community Support
- **Reddit**: [r/ZaiGLM](https://www.reddit.com/r/ZaiGLM)
- **GitHub Issues**: [https://github.com/zai-org/z-ai-sdk-python/issues](https://github.com/zai-org/z-ai-sdk-python/issues)

---

**Disclaimer**: This documentation is compiled from publicly available information as of July 2026. Prices, limits, and features are subject to change. Always refer to official Z.AI documentation for the most current information.

**Sources**:
- [Z.AI Official Documentation](https://docs.z.ai/api-reference/introduction)
- [Z.AI Pricing Guide](https://docs.z.ai/guides/overview/pricing)
- [Z.AI Quick Start](https://docs.z.ai/guides/overview/quick-start)
- [Z.AI FAQ](https://docs.z.ai/devpack/faq)
- [Z.ai Subscription Page](https://z.ai/subscribe)