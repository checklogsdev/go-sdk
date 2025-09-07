# CheckLogs Go SDK

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/checklogsdev/go-sdk)](https://goreportcard.com/report/github.com/checklogsdev/go-sdk)

Official Go SDK for [CheckLogs.dev](https://checklogs.dev) - A powerful log monitoring system.

```bash
go get github.com/checklogsdev/checklogs-go-sdk
```

For a guided setup, you can run our quick-start example:

```bash
# Set your API key
export CHECKLOGS_API_KEY=your-api-key-here

# Run the example
go run github.com/checklogsdev/checklogs-go-sdk/examples/basic.go
```

This will:
- Test your API key connection
- Show basic logging functionality  
- Demonstrate advanced features
- Provide next steps to get you started

## Basic Usage

```go
package main

import (
    "context"
    "github.com/checklogsdev/checklogs-go-sdk"
)

func main() {
    // Create a logger instance
    logger := checklogs.CreateLogger("your-api-key-here")
    
    ctx := context.Background()
    
    // Log messages
    logger.Info(ctx, "Application started")
    logger.Error(ctx, "Something went wrong", map[string]interface{}{
        "error_code": 500,
    })
}
```

## Package Support

This package supports all Go versions 1.21 and above.

The package automatically provides clean, idiomatic Go code with:
- **Thread-safe operations** - Safe for concurrent goroutines
- **Context support** - Native Go context integration for cancellation and timeouts
- **Structured logging** - Rich metadata and context support

## Features

- ✅ Full API coverage (logging, retry management, analytics)
- ✅ Thread-safe for concurrent use
- ✅ Native Go context support
- ✅ Automatic retry mechanism with exponential backoff
- ✅ Enhanced logging with metadata (hostname, process info, timestamps)
- ✅ Console output integration
- ✅ Child loggers with inherited context
- ✅ Timer functionality for performance measurement
- ✅ Error handling with custom error types
- ✅ Validation and sanitization
- ✅ Configurable timeouts and endpoints

## Core Usage

### Basic Logger

```go
package main

import (
    "context"
    "github.com/checklogsdev/checklogs-go-sdk"
)

func main() {
    logger := checklogs.CreateLogger("your-api-key")
    ctx := context.Background()
    
    // Log at different levels
    logger.Debug(ctx, "Debug information")
    logger.Info(ctx, "Application started")
    logger.Warning(ctx, "This is a warning")
    logger.Error(ctx, "An error occurred")
    logger.Critical(ctx, "Critical system failure")
}
```

### Advanced Logger

```go
package main

import (
    "context"
    "time"
    "github.com/checklogsdev/checklogs-go-sdk"
)

func main() {
    userID := int64(123)
    
    // Create logger with options
    options := &checklogs.Options{
        Source:  "my-go-app",
        UserID:  &userID,
        Context: map[string]interface{}{
            "version": "1.0.0",
            "env":     "production",
        },
        ConsoleOutput: true,
        Timeout:       30 * time.Second,
    }
    
    logger := checklogs.NewLogger("your-api-key", options)
    ctx := context.Background()
    
    // Send log with additional context
    logger.Info(ctx, "User action performed", map[string]interface{}{
        "action": "file_upload",
        "file_size": 1024000,
        "duration_ms": 250,
    })
}
```

## Configuration Options

```go
type Options struct {
    Source        string                 // Default source identifier
    UserID        *int64                 // Default user ID
    Context       map[string]interface{} // Default context merged with all logs
    Silent        bool                   // Suppress HTTP requests (console only)
    ConsoleOutput bool                   // Enable console output (default: true)
    BaseURL       string                 // Custom API endpoint
    Timeout       time.Duration          // HTTP request timeout (default: 30s)
}
```

## Child Loggers

Create child loggers with inherited context:

```go
package main

import (
    "context"
    "github.com/checklogsdev/checklogs-go-sdk"
)

func main() {
    // Main logger with service context
    mainLogger := checklogs.CreateLogger("your-api-key")
    
    // Child logger for user module
    userLogger := mainLogger.Child(map[string]interface{}{
        "module": "user",
        "service": "authentication",
    })
    
    // Child logger for order module  
    orderLogger := mainLogger.Child(map[string]interface{}{
        "module": "orders",
        "service": "payment",
    })
    
    ctx := context.Background()
    
    // Each child inherits parent context
    userLogger.Info(ctx, "User login attempt")  // Context: {module: "user", service: "authentication"}
    orderLogger.Error(ctx, "Payment failed")    // Context: {module: "orders", service: "payment"}
}
```

## Performance Timing

Measure execution time:

```go
package main

import (
    "context"
    "time"
    "github.com/checklogsdev/checklogs-go-sdk"
)

func main() {
    logger := checklogs.CreateLogger("your-api-key")
    
    // Start timer
    timer := logger.Time("db-query", "Executing database query")
    
    // Simulate some work
    time.Sleep(100 * time.Millisecond)
    
    // End timer (automatically logs end time with duration)
    duration := timer.End()
    fmt.Printf("Operation took %v\n", duration)
}
```

## Error Handling

The SDK provides specific error types:

```go
package main

import (
    "context"
    "fmt"
    "github.com/checklogsdev/checklogs-go-sdk"
)

func main() {
    logger := checklogs.CreateLogger("your-api-key")
    ctx := context.Background()
    
    err := logger.Info(ctx, "Test message")
    if err != nil {
        if checkLogsErr, ok := err.(*checklogs.CheckLogsError); ok {
            switch checkLogsErr.Type {
            case "ValidationError":
                fmt.Println("Validation failed:", checkLogsErr.Message)
            case "APIError":
                fmt.Printf("API error: %s (code: %d)\n", checkLogsErr.Message, checkLogsErr.Code)
            case "NetworkError":
                fmt.Println("Network problem:", checkLogsErr.Message)
            case "ConfigurationError":
                fmt.Println("Configuration issue:", checkLogsErr.Message)
            }
        }
    }
}
```

## Retry Queue Management

The logger automatically retries failed requests:

```go
package main

import (
    "context"
    "fmt"
    "github.com/checklogsdev/checklogs-go-sdk"
)

func main() {
    logger := checklogs.CreateLogger("your-api-key")
    
    // Check retry queue status
    queueSize := logger.GetRetryQueueSize()
    fmt.Printf("%d logs pending retry\n", queueSize)
    
    // Wait for all logs to be sent
    ctx := context.Background()
    success := logger.FlushRetryQueue(ctx)
    fmt.Printf("Successfully sent %d logs\n", success)
    
    // Clear retry queue if needed
    logger.ClearRetryQueue()
}
```

## Log Levels

Supported log levels (in order of severity):

- `checklogs.Debug` - Development and troubleshooting information
- `checklogs.Info` - General application flow
- `checklogs.Warning` - Potentially harmful situations  
- `checklogs.Error` - Error events that might still allow the application to continue
- `checklogs.Critical` - Very severe error events that might cause the application to abort

## Data Validation

The SDK automatically validates and sanitizes data:

- **Message**: Required, max 1024 characters
- **Level**: Must be valid level
- **Source**: Max 100 characters  
- **Context**: Objects only, max 5000 characters when serialized
- **User ID**: Must be a valid int64

## Best Practices

### Goroutine Safety
The logger is safe for concurrent use across goroutines:

```go
func handleRequest(logger *checklogs.Logger, requestID string) {
    requestLogger := logger.Child(map[string]interface{}{
        "request_id": requestID,
    })
    
    // Use requestLogger safely in this goroutine
    requestLogger.Info(context.Background(), "Processing request")
}
```

### Context Management
Always use context for cancellation and timeouts:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := logger.Info(ctx, "Important message")
if err != nil {
    log.Printf("Failed to send log: %v", err)
}
```

### Error Handling
Always handle potential logging errors:

```go
if err := logger.Error(ctx, "Database connection failed", map[string]interface{}{
    "database": "users",
    "error": dbErr.Error(),
}); err != nil {
    // Log locally as fallback
    log.Printf("Failed to send to CheckLogs: %v", err)
}
```

### Graceful Shutdown
Flush pending logs before shutdown:

```go
func gracefulShutdown(logger *checklogs.Logger) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    success := logger.FlushRetryQueue(ctx)
    if success == 0 {
        log.Println("Warning: Some logs may not have been sent")
    }
}
```

## Framework Integration

### Gin Web Framework

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/checklogsdev/checklogs-go-sdk"
)

func main() {
    logger := checklogs.CreateLogger("your-api-key")
    
    r := gin.Default()
    
    // Request logging middleware
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
        logger := c.MustGet("logger").(*checklogs.Logger)
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

### Echo Web Framework

```go
package main

import (
    "github.com/labstack/echo/v4"
    "github.com/checklogsdev/checklogs-go-sdk"
)

func main() {
    logger := checklogs.CreateLogger("your-api-key")
    
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

### Background Job Processing

```go
package main

import (
    "context"
    "github.com/checklogsdev/checklogs-go-sdk"
)

func processJob(jobID string) {
    logger := checklogs.CreateLogger("your-api-key")
    
    jobLogger := logger.Child(map[string]interface{}{
        "job_id": jobID,
        "worker": "background-processor",
    })
    
    timer := jobLogger.Time("job-processing", "Processing background job")
    
    ctx := context.Background()
    jobLogger.Info(ctx, "Job started")
    
    // ... job processing logic ...
    
    duration := timer.End()
    jobLogger.Info(ctx, "Job completed", map[string]interface{}{
        "status": "success",
        "duration_seconds": duration.Seconds(),
    })
}
```

Note: The SDK supports Go 1.21 and above. Use standard `import` statements as shown in the examples.

---

**License**: MIT

**Documentation**: [https://docs.checklogs.dev](https://docs.checklogs.dev)  
**Issues**: [GitHub Issues](https://github.com/checklogsdev/checklogs-go-sdk/issues)  
**Email**: [contact@loggersimple.com](mailto:contact@loggersimple.com)