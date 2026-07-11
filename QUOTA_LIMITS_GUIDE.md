# Z.ai GLM Coding Plan Quota Limits Guide

> **For developers**: See [QUOTA_SYSTEM_DEVELOPER_GUIDE.md](QUOTA_SYSTEM_DEVELOPER_GUIDE.md) for architecture, maintenance, and extension details.

## Overview

Z.ai GLM Coding Plan has multiple quota windows that track different types of usage:

### **Token Limits** (API Requests)
- **5-hour rolling window**: Tracks API request tokens over a 5-hour rolling period
- **Weekly window**: Tracks API request tokens over a 7-day rolling period

### **MCP Tools Limits** (External Tool Usage)
- **Monthly quota**: Tracks usage of MCP tools (web search, web-reader, zread) per month

## Understanding the Display

When you run `zai-client accounts quota`, you'll see output like:

```
📊 GLM Coding Plan Usage (PRO tier)

• 5-hour rolling token window
  Usage: 100%
  Resets: 2026-07-08 03:51:20 CEST (in 51m)

• weekly token window
  Usage: 99%
  Resets: 2026-07-10 09:17:18 CEST (in 2d 6h)

• monthly MCP tools quota
  Usage: 574/1000 (57%) — 426 remaining
  Resets: 2026-07-26 09:17:18 CEST (in 18d 6h)
  By tool:
    - search-prime: 470
    - web-reader: 97
    - zread: 7
```

## Account Examples

### **Damir's Account** (All three quotas)
- ✅ **5-hour rolling token window**: 100% used (resets soon)
- ⚠️ **Weekly token window**: 99% used (near limit)
- ✅ **Monthly MCP tools quota**: 57% used (426 remaining)

### **Kirill's Account** (Two quotas)
- ✅ **5-hour rolling token window**: 29% used (plenty of capacity)
- ✅ **Monthly MCP tools quota**: 5% used (950 remaining)
- ❌ **Weekly token window**: Not shown (may not apply to this plan)

## How the Quota Windows Work

### **Rolling Windows**
The 5-hour and weekly windows are **rolling windows**, not fixed periods:
- They track the total usage within the last 5 hours or 7 days
- As time passes, older usage drops out of the window
- This is why you see continuous reset times rather than fixed weekly resets

### **Monthly Quotas**
MCP tools have **monthly quotas** that reset on a fixed schedule:
- The quota is measured in tool calls, not tokens
- Each tool type (search-prime, web-reader, zread) counts toward the total
- Monthly reset happens at a fixed calendar date

## API Response Structure

The Z.ai API returns quota data with these fields:

```json
{
  "type": "TOKENS_LIMIT",     // or "TIME_LIMIT" for MCP tools
  "unit": 3,                  // Time unit code (3=hours, 5=monthly, 6=weekly)
  "number": 5,                // Number of units
  "currentValue": 0,          // Current usage
  "usage": 0,                 // Total limit (0 = not provided)
  "remaining": 0,             // Remaining quota
  "percentage": 100.0,        // Usage percentage
  "nextResetTime": 1783475480459,  // Reset timestamp (milliseconds)
  "usageDetails": [...]       // Tool breakdown for TIME_LIMIT
}
```

## Unit Code Reference

| Unit Code | Number | Meaning | Example |
|-----------|--------|---------|---------|
| 3 | 5 | 5-hour rolling window | TOKENS_LIMIT |
| 6 | 1 | Weekly rolling window | TOKENS_LIMIT |
| 5 | 1 | Monthly quota | TIME_LIMIT |

## Tips for Managing Quotas

1. **Watch the 5-hour window closely** - It resets frequently but fills quickly
2. **Use the weekly window** for planning larger tasks
3. **Monitor MCP tools usage** - Monthly quota can run out if you make many web searches
4. **Check multiple accounts** - Use `zai-client accounts quota` to see all accounts
5. **Switch accounts** - Use `zai-client accounts use <name>` to switch between accounts

## Troubleshooting

### **Why don't I see all quota windows?**
- Different plan tiers may have different quota structures
- Some accounts may not have certain quota types enabled
- Check with `zai-client accounts show <name>` to see account details

### **Why are my limits different from the examples?**
- Plan tiers (Lite/Pro/Max) have different quota amounts
- Custom enterprise plans may have custom limits
- Current usage affects remaining percentages

### **What happens when I exceed a limit?**
- API requests will fail with quota exceeded errors
- You'll need to wait for the window to reset
- Consider using a different account if available
