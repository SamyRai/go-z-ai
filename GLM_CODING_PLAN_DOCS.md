# Z.AI API Discovery - CORRECTED

## 🔍 Real GLM Coding Plan Documentation

### **✅ CORRECT Endpoints for Different Account Types:**

| Account Type | API Endpoint | Purpose |
|--------------|--------------|---------|
| **GLM Coding Plan (Yearly Subscription)** | `https://api.z.ai/api/coding/paas/v4` | Subscription access |
| **Pay-as-you-go** | `https://api.z.ai/api/paas/v4` | Token-based billing |
| **Claude Code & Goose (Coding Plan)** | `https://api.z.ai/api/anthropic` | Anthropic protocol |

### **📊 GLM Coding Plan Usage Limits:**

#### **5-Hour Rolling Window:**
| Plan | Prompts per 5 Hours | Weekly Limit |
|------|-------------------|--------------|
| **Lite** | ~80 prompts | ~400 prompts |
| **Pro** | ~400 prompts | ~2,000 prompts |
| **Max** | ~1,600 prompts | ~8,000 prompts |

#### **Model Usage Multipliers:**
- **GLM-5.2 & GLM-5-Turbo**: 3× (peak), 2× (off-peak)
- **GLM-4.7**: 1× (standard rate)

#### **MCP Tool Quotas:**
| Plan | Web Search/Reader | Vision Understanding |
|------|------------------|-------------------|
| **Lite** | 100/month | 5-hour pool |
| **Pro** | 1,000/month | 5-hour pool |
| **Max** | 4,000/month | 5-hour pool |

### **🎯 Key Requirements for GLM Coding Plan:**

1. **Correct Base URL**: `https://api.z.ai/api/coding/paas/v4`
2. **Supported Models Only**: GLM-5.2, GLM-5-Turbo, GLM-4.7
3. **Official Tools Only**: Claude Code, Cline, OpenCode, etc.
4. **No Balance Deduction**: Uses quota, not account balance

### **🔍 Usage Monitoring:**

#### **Official Methods:**
- **Usage Query Plugin**: Built-in Claude Code extension
- **Community Tools**: 
  - [opencode-glm-quota](https://github.com/guyinwonder168/opencode-glm-quota)
  - [zai-usage-tracker](https://github.com/melon-hub/zai-usage-tracker)

#### **Potential Usage Endpoint:**
- **`https://api.z.ai/api/monitor/usage`** (mentioned by community)

## 🚀 Implementation Requirements:

### **Multi-Endpoint Support:**
1. **Auto-detect account type** based on API key
2. **Fallback between endpoints** if one fails
3. **Proper quota tracking** for subscription plans
4. **Model-specific usage calculation** with multipliers

### **Account Type Detection:**
```go
// Try Coding Plan endpoint first
// If successful → Subscription account
// If 401/1113 → Try pay-as-you-go endpoint
// If both fail → Invalid API key
```

### **Usage Tracking for Subscriptions:**
- **5-hour window tracking**
- **Model usage multipliers** (3× for GLM-5.2, 1× for GLM-4.7)
- **Weekly quota monitoring**
- **MCP tool usage limits**

## 📝 Sources:
- [Z.AI DevPack Overview](https://docs.z.ai/devpack/overview)
- [Z.AI FAQ](https://docs.z.ai/devpack/faq)
- [Z.AI Quick Start](https://docs.z.ai/guides/overview/quick-start)
- [Usage Query Plugin](https://docs.z.ai/devpack/extension/usage-query-plugin)