# CheckLogs Go SDK

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/checklogsdev/go-sdk)](https://goreportcard.com/report/github.com/checklogsdev/go-sdk)

Official Go SDK for [CheckLogs.dev](https://checklogs.dev) - A powerful log monitoring system.

## Features

- ✅ **Complete API Coverage** - Full support for logging, retrieval, and analytics
- ✅ **High Performance** - Optimized for high-throughput applications
- ✅ **Automatic Retry** - Built-in retry mechanism with exponential backoff
- ✅ **Context Support** - Full Go context support for cancellation and timeouts
- ✅ **Enhanced Logging** - Rich metadata including hostname, process info, and timestamps
- ✅ **Child Loggers** - Create loggers with inherited context
- ✅ **Timer Support** - Built-in execution time measurement
- ✅ **Statistics** - Comprehensive analytics and metrics
- ✅ **Error Handling** - Detailed error types with helper methods
- ✅ **Thread Safe** - Safe for concurrent use
- ✅ **Validation** - Automatic data validation and sanitization

## Installation

```bash
go get github.com/checklogsdev/go-sdk
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/checklogsdev/go-sdk"
)

func main() {
    // Create a logger
    logger := checklogs.CreateLogger("your-api-key-here", nil)
    
    ctx := context.Background()
    
    // Log messages
    logger.Info(ctx, "Application started")
    logger.Error(ctx, "Something went wrong", map[string]interface{}{
        "error_code": 500,
        "user_id":    123,
    })
}
```

### Using the Client Directly

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/checklogsdev/go-sdk"
)

func main() {
    // Create a client
    client := checklogs.NewCheckLogsClient("your-api-key", nil)
    
    ctx := context.Background()
    
    // Send a log
    logData := checklogs.LogData{
        Message: "User logged in",
        Level:   checklogs.LogLevelInfo,
        Context: map[string]interface{}{
            "user_id": 123,
            "ip":      "192.168.1.1",
        },
    }
    
    if err := client.Log(ctx, logData); err != nil {
        log.Printf("Failed to send log: %v", err)
    }
    
    // Retrieve logs
    params := checklogs.GetLogsParams{
        Limit: 100,
        Level: checklogs.LogLevelError,
    }
    
    logs, err := client.GetLogs(ctx, params)
    if err != nil {
        log.Printf("Failed to retrieve logs: %v", err)
        return
    }
    
    fmt.Printf("Retrieved %d logs\n", len(logs.Data))
}
```

## Configuration

### Client Options

```go
options := &checklogs.ClientOptions{
    Timeout:         30 * time.Second,  // Request timeout
    ValidatePayload: true,              // Validate data before sending
    BaseURL:         "https://api.checklogs.dev", // Custom API endpoint
}

client := checklogs.NewCheckLogsClient("api-key", options)
```

### Logger Options

```go
options := &checklogs.LoggerOptions{
    // Client options
    ClientOptions: checklogs.ClientOptions{
        Timeout:         30 * time.Second,
        ValidatePayload: true,
    },
    
    // Logger-specific options
    Source:           "my-app",                    // Default source
    UserID:           &userID,                     // Default user ID
    DefaultContext:   map[string]interface{}{      // Default context
        "environment": "production",
        "version":     "1.0.0",
    },
    Silent:           false,                       // Suppress all output
    ConsoleOutput:    true,                        // Also log to console
    EnabledLevels:    []checklogs.LogLevel{        // Enabled log levels
        checklogs.LogLevelInfo,
        checklogs.LogLevelError,
    },
    IncludeTimestamp: true,                        // Add timestamp to context
    IncludeHostname:  true,                        // Add hostname to context
}

logger := checklogs.NewCheckLogsLogger("api-key", options)
```

## Usage Examples

### Child Loggers

Create child loggers with inherited context:

```go
// Main logger with service context
mainLogger := checklogs.CreateLogger("api-key", &checklogs.LoggerOptions{
    DefaultContext: map[string]interface{}{
        "service": "api",
        "version": "1.0.0",
    },
})

// Child logger for user module
userLogger := mainLogger.Child(map[string]interface{}{
    "module": "user",
})

// Child logger for order module  
orderLogger := mainLogger.Child(map[string]interface{}{
    "module": "orders",
})

ctx := context.Background()

// Each child inherits parent context
userLogger.Info(ctx, "User created")   // Context: {service: "api", version: "1.0.0", module: "user"}
orderLogger.Error(ctx, "Order failed") // Context: {service: "api", version: "1.0.0", module: "orders"}
```

### Timing Operations

Measure execution time:

```go
logger := checklogs.CreateLogger("api-key", nil)

// Start timer
timer := logger.Time("db-query", "Executing database query")

// ... your code here ...

// End timer (automatically logs duration)
duration := timer.End()
fmt.Printf("Operation took %v\n", duration)
```

### Error Handling

The SDK provides specific error types:

```go
import (
    "context"
    "errors"
    "fmt"
    
    "github.com/checklogsdev/go-sdk"
)

func handleLogging() {
    logger := checklogs.CreateLogger("api-key", nil)
    ctx := context.Background()
    
    err := logger.Info(ctx, "Test message")
    if err != nil {
        var apiErr *checklogs.APIError
        var netErr *checklogs.NetworkError
        var valErr *checklogs.ValidationError
        
        switch {
        case errors.As(err, &apiErr):
            fmt.Printf("API error: %d - %s\n", apiErr.StatusCode, apiErr.Message)
            if apiErr.IsAuthError() {
                fmt.Println("Authentication problem")
            } else if apiErr.IsRateLimitError() {
                fmt.Println("Rate limit exceeded")
            }
            
        case errors.As(err, &netErr):
            fmt.Printf("Network error: %s\n", netErr.Message)
            if netErr.IsTimeoutError() {
                fmt.Println("Request timed out")
            }
            
        case errors.As(err, &valErr):
            fmt.Printf("Validation error on %s: %s\n", valErr.Field, valErr.Message)
        }
    }
}
```

### Retry Queue Management

The logger automatically retries failed requests:

```go
client := checklogs.NewCheckLogsClient("api-key", nil)

// Check retry queue status
status := client.GetRetryQueueStatus()
fmt.Printf("%d logs pending retry\n", status.Count)

// Wait for all logs to be sent
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

success := client.Flush(ctx)
if success {
    fmt.Println("All logs sent successfully")
} else {
    fmt.Println("Some logs failed to send")
}

// Clear retry queue if needed
client.ClearRetryQueue()
```

### Statistics and Analytics

```go
client := checklogs.NewCheckLogsClient("api-key", nil)
ctx := context.Background()

// Get basic statistics
stats, err := client.GetStats(ctx)
if err == nil {
    fmt.Printf("Total logs: %d\n", stats.TotalLogs)
    fmt.Printf("Error rate: %.2f%%\n", stats.ErrorRate)
}

// Get analytics summary
summary, err := client.GetSummary(ctx)
if err == nil {
    fmt.Printf("Error rate: %.2f%%\n", summary.Data.Analytics.ErrorRate)
    fmt.Printf("Trend: %s\n", summary.Data.Analytics.Trend)
    fmt.Printf("Peak day: %s\n", summary.Data.Analytics.PeakDay)
}

// Get specific metrics
errorRate, err := client.GetErrorRate(ctx)
trend, err := client.GetTrend(ctx)
peakDay, err := client.GetPeakDay(ctx)
```

## Log Levels

Supported log levels (in order of severity):

- `LogLevelDebug` - Development and troubleshooting information
- `LogLevelInfo` - General application flow
- `LogLevelWarning` - Potentially harmful situations
- `LogLevelError` - Error events that might still allow the application to continue
- `LogLevelCritical` - Very severe error events that might cause the application to abort

## Data Validation

The SDK automatically validates and sanitizes data:

- **Message**: Required, max 1024 characters
- **Level**: Must be valid level, defaults to 'info'
- **Source**: Max 100 characters
- **Context**: Objects only, max 5000 characters when serialized
- **User ID**: Must be a valid int64

## Best Practices

### Batch Operations
Use child loggers for related operations:

```go
mainLogger := checklogs.CreateLogger("api-key", nil)
requestLogger := mainLogger.Child(map[string]interface{}{
    "request_id": generateRequestID(),
    "user_id":    userID,
})

// Use requestLogger for all operations in this request
```

### Level Filtering
Only enable necessary log levels in production:

```go
logger := checklogs.NewCheckLogsLogger("api-key", &checklogs.LoggerOptions{
    EnabledLevels: []checklogs.LogLevel{
        checklogs.LogLevelInfo,
        checklogs.LogLevelError,
        checklogs.LogLevelCritical,
    },
})
```

### Context Size
Keep context objects reasonably small:

```go
// Good
context := map[string]interface{}{
    "user_id": 123,
    "action":  "login",
}

// Avoid large objects
// context := map[string]interface{}{
//     "large_data": hugeStruct,
// }
```

### Error Handling
Always handle potential network issues:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := logger.Info(ctx, "Message"); err != nil {
    // Handle error appropriately
    log.Printf("Failed to send log: %v", err)
}
```

### Graceful Shutdown
Call `Flush()` before application termination:

```go
func gracefulShutdown(client *checklogs.CheckLogsClient) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if !client.Flush(ctx) {
        log.Println("Warning: Some logs may not have been sent")
    }
}
```

## Web Framework Integration

### Gin Integration

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/checklogsdev/go-sdk"
)

func main() {
    logger := checklogs.CreateLogger("your-api-key", &checklogs.LoggerOptions{
        Source: "web-api",
    })
    
    r := gin.Default()
    
    // Logging middleware
    r.Use(func(c *gin.Context) {
        requestLogger := logger.Child(map[string]interface{}{
            "request_id": c.GetHeader("X-Request-ID"),
            "method":     c.Request.Method,
            "path":       c.Request.URL.Path,
            "ip":         c.ClientIP(),
        })
        
        c.Set("logger", requestLogger)
        c.Next()
    })
    
    r.GET("/users/:id", func(c *gin.Context) {
        logger := c.MustGet("logger").(*checklogs.CheckLogsLogger)
        userID := c.Param("id")
        
        logger.Info(c.Request.Context(), "Fetching user", map[string]interface{}{
            "user_id": userID,
        })
        
        // ... your logic here ...
        
        logger.Info(c.Request.Context(), "User fetched successfully")
        c.JSON(200, gin.H{"user": "data"})
    })
    
    r.Run(":8080")
}
```

### Echo Integration

```go
package main

import (
    "github.com/labstack/echo/v4"
    "github.com/checklogsdev/go-sdk"
)

func main() {
    logger := checklogs.CreateLogger("your-api-key", nil)
    
    e := echo.New()
    
    // Logging middleware
    e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            requestLogger := logger.Child(map[string]interface{}{
                "request_id": c.Request().Header.Get("X-Request-ID"),
                "method":     c.Request().Method,
                "path":       c.Request().URL.Path,
            })
            
            c.Set("logger", requestLogger)
            return next(c)
        }
    })
    
    e.Start(":8080")
}
```

## Testing

### Running Tests

```bash
go test ./...
```

### Coverage

```bash
go test -cover ./...
```

### Benchmarks

```bash
go test -bench=. ./...
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: [https://docs.checklogs.dev](https://docs.checklogs.dev)
- **Issues**: [GitHub Issues](https://github.com/checklogsdev/go-sdk/issues)
- **Email**: [support@checklogs.dev](mailto:support@checklogs.dev)

## Changelog

### v1.0.0
- Initial release
- Complete API coverage
- Automatic retry mechanism
- Child loggers support
- Timer functionality
- Comprehensive error handling
- Statistics and analytics
- Thread-safe operations