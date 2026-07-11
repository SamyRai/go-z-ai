# Quota System Developer Guide

## Architecture Overview

The quota system has been refactored to use proper enums and structured mappings instead of hardcoded conditionals. This makes the code more maintainable and easier to extend.

## Core Components

### **Constants and Enums** (`pkg/client/quota.go`)

```go
// Quota limit type constants
const (
    QuotaTypeTokensLimit = "TOKENS_LIMIT" // API request token limits
    QuotaTypeTimeLimit   = "TIME_LIMIT"   // MCP tools usage limits
)

// Time unit code constants from Z.ai API
const (
    UnitCodeHourly  = 3 // 5-hour rolling window (for TOKENS_LIMIT)
    UnitCodeWeekly  = 6 // Weekly rolling window (for TOKENS_LIMIT)
    UnitCodeMonthly = 5 // Monthly quota (for TIME_LIMIT - MCP tools)
)
```

### **Structured Mappings**

The `quotaWindowConfigs` array provides a centralized mapping of all known quota window types:

```go
var quotaWindowConfigs = []QuotaWindowConfig{
    // Token limit windows
    {Type: QuotaTypeTokensLimit, UnitCode: UnitCodeHourly, Number: 5, Description: "5-hour rolling token window"},
    {Type: QuotaTypeTokensLimit, UnitCode: UnitCodeWeekly, Number: 1, Description: "weekly token window"},
    
    // MCP tools limit windows
    {Type: QuotaTypeTimeLimit, UnitCode: UnitCodeMonthly, Number: 1, Description: "monthly MCP tools quota"},
}
```

### **Helper Methods**

The `QuotaLimit` struct now has helper methods for common operations:

```go
// Get human-readable description
limit.WindowDescription() // "5-hour rolling token window"

// Check limit type
limit.IsTokenLimit()  // true for TOKENS_LIMIT
limit.IsToolsLimit()  // true for TIME_LIMIT

// Check usage status
limit.IsExhausted()   // true if >= 99% used
limit.IsLow()        // true if >= 80% used

// Get reset time
limit.ResetTime()     // time.Time of next reset
```

## Adding New Quota Window Types

When Z.ai adds new quota window types, follow these steps:

### **1. Add Constants**

```go
// Time unit code constants from Z.ai API
const (
    UnitCodeHourly  = 3 // 5-hour rolling window (for TOKENS_LIMIT)
    UnitCodeWeekly  = 6 // Weekly rolling window (for TOKENS_LIMIT)
    UnitCodeMonthly = 5 // Monthly quota (for TIME_LIMIT - MCP tools)
    UnitCodeDaily   = 4 // ADD NEW: Daily quota (example)
)
```

### **2. Add Configuration**

```go
var quotaWindowConfigs = []QuotaWindowConfig{
    // ... existing configs ...
    
    // ADD NEW: Daily window configuration
    {Type: QuotaTypeTokensLimit, UnitCode: UnitCodeDaily, Number: 1, Description: "daily token window"},
}
```

### **3. Update Documentation**

- Update `QUOTA_LIMITS_GUIDE.md` with the new window type
- Add unit code to the reference table
- Update examples if needed

## How the Mapping Works

The `findWindowConfig()` function searches the `quotaWindowConfigs` array for a matching combination of:

1. **Type** (TOKENS_LIMIT or TIME_LIMIT)
2. **UnitCode** (3, 5, 6, etc.)
3. **Number** (5, 1, etc.)

If a match is found, it returns the configured description. If no match is found, it generates a generic fallback description like "unknown TOKENS_LIMIT window (unit 4 × 2)".

## Benefits of This Approach

### **Maintainability**
- **Single source of truth**: All quota window definitions in one place
- **Easy to extend**: Add new types by adding to the config array
- **Type safety**: Use constants instead of magic strings/numbers

### **Clarity**
- **Self-documenting**: Clear constant names like `UnitCodeHourly`
- **Centralized logic**: No scattered switch statements
- **Fallback handling**: Graceful handling of unknown types

### **Testing**
- **Easy to test**: Can test the mapping function independently
- **Predictable output**: Consistent descriptions for known types
- **Graceful degradation**: Works even with unknown API changes

## Migration from Old Approach

### **Before (Hardcoded Conditionals)**
```go
func (q *QuotaLimit) WindowDescription() string {
    switch q.Type {
    case "TOKENS_LIMIT":
        if q.Unit == 3 && q.Number == 5 {
            return "5-hour rolling token window"
        } else if q.Unit == 6 && q.Number == 1 {
            return "weekly token window"
        } else {
            return fmt.Sprintf("token window (unit %d × %d)", q.Unit, q.Number)
        }
    // ... more nested conditionals
    }
}
```

### **After (Structured Mapping)**
```go
func (q *QuotaLimit) WindowDescription() string {
    config := findWindowConfig(q.Type, q.Unit, q.Number)
    return config.Description
}
```

## Testing the System

### **Unit Tests**
Test the mapping function with different combinations:

```go
func TestFindWindowConfig(t *testing.T) {
    // Test known configuration
    config := findWindowConfig(QuotaTypeTokensLimit, UnitCodeHourly, 5)
    assert.Equal(t, "5-hour rolling token window", config.Description)
    
    // Test unknown configuration
    config = findWindowConfig("UNKNOWN_TYPE", 999, 999)
    assert.Contains(t, config.Description, "unknown")
}
```

### **Integration Tests**
Test the full quota display with real API responses:

```bash
./zai-client accounts quota
./zai-client accounts quota --format json
```

## Troubleshooting

### **New quota types not displaying correctly**
1. Check if the unit code and number match exactly
2. Add the new configuration to `quotaWindowConfigs`
3. Rebuild and test

### **Generic fallback descriptions**
- This means the quota type is not in the known configurations
- Add it to `quotaWindowConfigs` or it might be a new API change

### **Code compiles but descriptions are wrong**
- Verify the constants match the API response
- Check the order of parameters in `findWindowConfig()`
- Test with JSON output to see raw API values

## File Structure

```
pkg/client/quota.go          # Core quota types and mappings
usage.go                     # Display logic for CLI output
QUOTA_LIMITS_GUIDE.md        # User-facing documentation
QUOTA_SYSTEM_DEVELOPER_GUIDE.md  # This file
```

## Future Improvements

Potential enhancements to consider:

1. **Dynamic configuration**: Load quota window configs from a JSON file
2. **User-defined aliases**: Let users customize window descriptions  
3. **Alert thresholds**: Customizable warning levels for different windows
4. **Historical tracking**: Track usage patterns over time
5. **Multi-language support**: Localize descriptions for international users

## Related Code

- **Account management**: `pkg/accounts/accounts.go`
- **CLI interface**: `accounts_cli.go`
- **API client**: `pkg/client/`
- **Documentation**: `QUOTA_LIMITS_GUIDE.md`
