package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/checklogsdev/go-sdk"
)

func main() {
	// Get API key from environment variable
	apiKey := os.Getenv("CHECKLOGS_API_KEY")
	if apiKey == "" {
		log.Fatal("CHECKLOGS_API_KEY environment variable is required")
	}

	fmt.Println("🚀 CheckLogs Go SDK - Complete Example")
	fmt.Println("=====================================")

	// Test all functionality
	runBasicExample(apiKey)
	runClientExample(apiKey)
	runLoggerExample(apiKey)
	runChildLoggerExample(apiKey)
	runTimerExample(apiKey)
	runErrorHandlingExample(apiKey)
	runRetryQueueExample(apiKey)
	runStatisticsExample(apiKey)
	runWebApplicationExample(apiKey)

	fmt.Println("\n✅ All examples completed successfully!")
}

// Basic logger usage
func runBasicExample(apiKey string) {
	fmt.Println("\n📝 1. Basic Logger Example")
	fmt.Println("-------------------------")

	// Create a basic logger
	logger := checklogs.CreateLogger(apiKey, nil)
	ctx := context.Background()

	// Log different levels
	logger.Debug(ctx, "Debug message for troubleshooting")
	logger.Info(ctx, "Application started successfully")
	logger.Warning(ctx, "This is a warning message")
	logger.Error(ctx, "An error occurred", map[string]interface{}{
		"error_code": 500,
		"component":  "database",
	})
	logger.Critical(ctx, "Critical system failure", map[string]interface{}{
		"severity": "high",
		"action":   "immediate_attention_required",
	})

	fmt.Println("✅ Basic logging completed")
}

// Client direct usage
func runClientExample(apiKey string) {
	fmt.Println("\n🔧 2. Client Direct Usage Example")
	fmt.Println("---------------------------------")

	// Create client with custom options
	options := &checklogs.ClientOptions{
		Timeout:         15 * time.Second,
		ValidatePayload: true,
		BaseURL:         checklogs.DefaultBaseURL,
	}

	client := checklogs.NewCheckLogsClient(apiKey, options)
	ctx := context.Background()

	// Send log using client directly
	logData := checklogs.LogData{
		Message: "Direct client usage example",
		Level:   checklogs.LogLevelInfo,
		Source:  "example-app",
		Context: map[string]interface{}{
			"method":    "direct",
			"timestamp": time.Now().Unix(),
			"example":   true,
		},
	}

	if err := client.Log(ctx, logData); err != nil {
		fmt.Printf("❌ Failed to send log: %v\n", err)
	} else {
		fmt.Println("✅ Direct client log sent successfully")
	}

	// Try to retrieve logs (this might fail if API doesn't support it yet)
	params := checklogs.GetLogsParams{
		Limit: 10,
		Level: checklogs.LogLevelInfo,
		Since: time.Now().Add(-24 * time.Hour),
	}

	logs, err := client.GetLogs(ctx, params)
	if err != nil {
		fmt.Printf("⚠️  Failed to retrieve logs (might not be implemented): %v\n", err)
	} else {
		fmt.Printf("✅ Retrieved %d logs\n", len(logs.Data))
	}
}

// Advanced logger configuration
func runLoggerExample(apiKey string) {
	fmt.Println("\n⚙️  3. Advanced Logger Configuration")
	fmt.Println("-----------------------------------")

	userID := int64(12345)

	// Create logger with advanced options
	options := &checklogs.LoggerOptions{
		ClientOptions: checklogs.ClientOptions{
			Timeout:         20 * time.Second,
			ValidatePayload: true,
		},
		Source:  "advanced-example",
		UserID:  &userID,
		DefaultContext: map[string]interface{}{
			"environment": "development",
			"version":     "1.0.0",
			"service":     "example-service",
		},
		Silent:           false,
		ConsoleOutput:    true,
		EnabledLevels:    []checklogs.LogLevel{checklogs.LogLevelInfo, checklogs.LogLevelError, checklogs.LogLevelCritical},
		IncludeTimestamp: true,
		IncludeHostname:  true,
	}

	logger := checklogs.NewCheckLogsLogger(apiKey, options)
	ctx := context.Background()

	// Log with inherited context
	logger.Info(ctx, "Advanced logger initialized")
	logger.Error(ctx, "Simulated error with rich context", map[string]interface{}{
		"request_id":   generateRequestID(),
		"user_action":  "file_upload",
		"file_size":    1024000,
		"error_detail": "insufficient_storage",
	})

	fmt.Println("✅ Advanced logger configuration completed")
}

// Child logger demonstration
func runChildLoggerExample(apiKey string) {
	fmt.Println("\n👶 4. Child Logger Example")
	fmt.Println("--------------------------")

	// Main logger with service context
	mainLogger := checklogs.CreateLogger(apiKey, &checklogs.LoggerOptions{
		Source: "microservice",
		DefaultContext: map[string]interface{}{
			"service": "user-management",
			"version": "2.1.0",
		},
	})

	ctx := context.Background()

	// Create child loggers for different modules
	authLogger := mainLogger.Child(map[string]interface{}{
		"module": "authentication",
	})

	userLogger := mainLogger.Child(map[string]interface{}{
		"module": "user-crud",
	})

	// Each child inherits parent context
	authLogger.Info(ctx, "User authentication attempt", map[string]interface{}{
		"username": "john.doe",
		"method":   "oauth2",
	})

	userLogger.Info(ctx, "User profile updated", map[string]interface{}{
		"user_id": 12345,
		"fields":  []string{"email", "phone"},
	})

	// Create nested child logger
	requestLogger := authLogger.Child(map[string]interface{}{
		"request_id": generateRequestID(),
		"session_id": generateSessionID(),
	})

	requestLogger.Warning(ctx, "Multiple failed login attempts detected")

	fmt.Println("✅ Child logger example completed")
}

// Timer functionality demonstration
func runTimerExample(apiKey string) {
	fmt.Println("\n⏱️  5. Timer Example")
	fmt.Println("--------------------")

	logger := checklogs.CreateLogger(apiKey, &checklogs.LoggerOptions{
		Source: "timer-example",
	})

	// Simulate database operation
	timer := logger.Time("database-query", "Executing complex database query")

	// Simulate some work
	simulateDatabaseWork()

	duration := timer.End()
	fmt.Printf("✅ Database operation completed in %v\n", duration)

	// Multiple timers
	timer1 := logger.Time("api-call", "Calling external API")
	timer2 := logger.Time("data-processing", "Processing user data")

	simulateAPICall()
	duration1 := timer1.End()

	simulateDataProcessing()
	duration2 := timer2.End()

	fmt.Printf("✅ API call took %v, data processing took %v\n", duration1, duration2)
}

// Error handling demonstration
func runErrorHandlingExample(apiKey string) {
	fmt.Println("\n🚨 6. Error Handling Example")
	fmt.Println("----------------------------")

	logger := checklogs.CreateLogger(apiKey, &checklogs.LoggerOptions{
		Source: "error-example",
	})

	ctx := context.Background()

	// Test with invalid data to trigger validation error
	invalidLogger := checklogs.CreateLogger("", nil) // Empty API key
	err := invalidLogger.Info(ctx, "This should fail")

	if err != nil {
		handleCheckLogsError(err)
	}

	// Test with oversized message
	oversizedMessage := make([]byte, 1025) // Over 1024 character limit
	for i := range oversizedMessage {
		oversizedMessage[i] = 'A'
	}

	err = logger.Info(ctx, string(oversizedMessage))
	if err != nil {
		handleCheckLogsError(err)
	}

	// Test with oversized context
	oversizedContext := make(map[string]interface{})
	largeData := make([]byte, 5001) // Over 5000 byte limit when serialized
	for i := range largeData {
		largeData[i] = 'X'
	}
	oversizedContext["large_field"] = string(largeData)

	err = logger.Info(ctx, "Normal message", oversizedContext)
	if err != nil {
		handleCheckLogsError(err)
	}

	fmt.Println("✅ Error handling example completed")
}

// Retry queue demonstration
func runRetryQueueExample(apiKey string) {
	fmt.Println("\n🔄 7. Retry Queue Example")
	fmt.Println("-------------------------")

	client := checklogs.NewCheckLogsClient(apiKey, &checklogs.ClientOptions{
		Timeout: 1 * time.Millisecond, // Very short timeout to trigger network errors
	})

	ctx := context.Background()

	// Send some logs that might fail due to short timeout
	for i := 0; i < 5; i++ {
		logData := checklogs.LogData{
			Message: fmt.Sprintf("Test log %d", i+1),
			Level:   checklogs.LogLevelInfo,
			Context: map[string]interface{}{
				"attempt": i + 1,
				"test":    true,
			},
		}

		err := client.Log(ctx, logData)
		if err != nil {
			fmt.Printf("⚠️  Log %d failed (expected): %v\n", i+1, err)
		}
	}

	// Check retry queue status
	status := client.GetRetryQueueStatus()
	fmt.Printf("📊 Retry queue status: %d logs pending\n", status.Count)

	// Create a new client with normal timeout for flushing
	normalClient := checklogs.NewCheckLogsClient(apiKey, nil)
	
	// Move failed logs to normal client (in real scenario, you'd use the same client)
	// This is just for demonstration
	flushCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	success := normalClient.Flush(flushCtx)
	fmt.Printf("✅ Flush result: %t\n", success)

	// Clear retry queue
	client.ClearRetryQueue()
	fmt.Println("✅ Retry queue cleared")
}

// Statistics demonstration
func runStatisticsExample(apiKey string) {
	fmt.Println("\n📊 8. Statistics Example")
	fmt.Println("------------------------")

	client := checklogs.NewCheckLogsClient(apiKey, nil)
	ctx := context.Background()

	// Try to get statistics (might not be implemented yet)
	stats, err := client.GetStats(ctx)
	if err != nil {
		fmt.Printf("⚠️  Failed to get stats (might not be implemented): %v\n", err)
	} else {
		fmt.Printf("📈 Total logs: %d\n", stats.TotalLogs)
		fmt.Printf("📈 Error rate: %.2f%%\n", stats.ErrorRate)
		fmt.Printf("📈 Last log: %v\n", stats.LastLog)
	}

	// Try to get summary
	summary, err := client.GetSummary(ctx)
	if err != nil {
		fmt.Printf("⚠️  Failed to get summary (might not be implemented): %v\n", err)
	} else {
		fmt.Printf("📊 Analytics error rate: %.2f%%\n", summary.Data.Analytics.ErrorRate)
		fmt.Printf("📊 Trend: %s\n", summary.Data.Analytics.Trend)
		fmt.Printf("📊 Peak day: %s\n", summary.Data.Analytics.PeakDay)
	}

	// Try individual metrics
	errorRate, err := client.GetErrorRate(ctx)
	if err != nil {
		fmt.Printf("⚠️  Failed to get error rate: %v\n", err)
	} else {
		fmt.Printf("📉 Current error rate: %.2f%%\n", errorRate)
	}

	fmt.Println("✅ Statistics example completed")
}

// Web application simulation
func runWebApplicationExample(apiKey string) {
	fmt.Println("\n🌐 9. Web Application Simulation")
	fmt.Println("--------------------------------")

	// Simulate a web application with request logging
	appLogger := checklogs.CreateLogger(apiKey, &checklogs.LoggerOptions{
		Source: "web-app",
		DefaultContext: map[string]interface{}{
			"application": "ecommerce-api",
			"environment": "production",
		},
	})

	// Simulate handling multiple requests
	for i := 1; i <= 3; i++ {
		simulateWebRequest(appLogger, i)
	}

	fmt.Println("✅ Web application simulation completed")
}

// Helper functions

func generateRequestID() string {
	return fmt.Sprintf("req_%d_%d", time.Now().Unix(), rand.Intn(10000))
}

func generateSessionID() string {
	return fmt.Sprintf("sess_%d_%d", time.Now().Unix(), rand.Intn(10000))
}

func simulateDatabaseWork() {
	time.Sleep(100 * time.Millisecond)
}

func simulateAPICall() {
	time.Sleep(200 * time.Millisecond)
}

func simulateDataProcessing() {
	time.Sleep(150 * time.Millisecond)
}

func handleCheckLogsError(err error) {
	switch e := err.(type) {
	case *checklogs.APIError:
		fmt.Printf("🔴 API Error: %d - %s\n", e.StatusCode, e.Message)
		if e.IsAuthError() {
			fmt.Println("   → Authentication problem detected")
		} else if e.IsRateLimitError() {
			fmt.Println("   → Rate limit exceeded")
		}

	case *checklogs.NetworkError:
		fmt.Printf("🌐 Network Error: %s\n", e.Message)
		if e.IsTimeoutError() {
			fmt.Println("   → Request timed out")
		}

	case *checklogs.ValidationError:
		fmt.Printf("✋ Validation Error on field '%s': %s\n", e.Field, e.Message)

	default:
		fmt.Printf("❓ Unknown error: %v\n", err)
	}
}

func simulateWebRequest(logger *checklogs.CheckLogsLogger, requestNum int) {
	ctx := context.Background()

	// Create request-specific logger
	requestID := generateRequestID()
	userID := int64(1000 + requestNum)

	requestLogger := logger.Child(map[string]interface{}{
		"request_id": requestID,
		"user_id":    userID,
		"endpoint":   "/api/users/profile",
		"method":     "GET",
	})

	// Start request timer
	timer := requestLogger.Time("request-processing", fmt.Sprintf("Processing request %d", requestNum))

	// Log request start
	requestLogger.Info(ctx, "Request received", map[string]interface{}{
		"ip":         fmt.Sprintf("192.168.1.%d", 100+requestNum),
		"user_agent": "CheckLogs-Go-SDK-Example/1.0",
	})

	// Simulate some processing
	time.Sleep(time.Duration(50+requestNum*10) * time.Millisecond)

	// Simulate authentication
	authLogger := requestLogger.Child(map[string]interface{}{
		"component": "auth",
	})

	authLogger.Debug(ctx, "Validating user token")

	// Simulate database query
	dbLogger := requestLogger.Child(map[string]interface{}{
		"component": "database",
	})

	dbTimer := dbLogger.Time("db-query", "Fetching user profile")
	time.Sleep(30 * time.Millisecond)
	dbDuration := dbTimer.End()

	dbLogger.Info(ctx, "User profile retrieved", map[string]interface{}{
		"query_time_ms": dbDuration.Milliseconds(),
		"records_found": 1,
	})

	// Simulate potential error for request 2
	if requestNum == 2 {
		requestLogger.Error(ctx, "Temporary service unavailable", map[string]interface{}{
			"error_code": "SERVICE_UNAVAILABLE",
			"retry_after": 5,
		})
	} else {
		// Successful response
		requestLogger.Info(ctx, "Request completed successfully", map[string]interface{}{
			"status_code": 200,
			"response_size": 1024 + requestNum*100,
		})
	}

	// End request timer
	totalDuration := timer.End()

	// Log request summary
	requestLogger.Info(ctx, "Request finished", map[string]interface{}{
		"total_duration_ms": totalDuration.Milliseconds(),
		"status": func() string {
			if requestNum == 2 {
				return "error"
			}
			return "success"
		}(),
	})

	fmt.Printf("   🌐 Request %d completed in %v\n", requestNum, totalDuration)
}

// Benchmark function (not called in main, but useful for testing)
func benchmarkLogging(apiKey string, numLogs int) {
	fmt.Printf("\n🏃 Running benchmark with %d logs\n", numLogs)

	logger := checklogs.CreateLogger(apiKey, &checklogs.LoggerOptions{
		ConsoleOutput: false, // Disable console output for benchmark
		Silent:        false,
	})

	ctx := context.Background()
	start := time.Now()

	for i := 0; i < numLogs; i++ {
		logger.Info(ctx, fmt.Sprintf("Benchmark log %d", i+1), map[string]interface{}{
			"index":     i + 1,
			"benchmark": true,
			"timestamp": time.Now().Unix(),
		})
	}

	duration := time.Since(start)
	logsPerSecond := float64(numLogs) / duration.Seconds()

	fmt.Printf("✅ Benchmark completed: %d logs in %v (%.2f logs/sec)\n", 
		numLogs, duration, logsPerSecond)
}

// Performance test function
func performanceTest(apiKey string) {
	fmt.Println("\n⚡ Performance Test")
	fmt.Println("-------------------")

	// Test with different batch sizes
	batchSizes := []int{10, 100, 500}

	for _, size := range batchSizes {
		benchmarkLogging(apiKey, size)
	}
}

// Stress test function
func stressTest(apiKey string) {
	fmt.Println("\n💪 Stress Test")
	fmt.Println("---------------")

	logger := checklogs.CreateLogger(apiKey, nil)
	ctx := context.Background()

	// Create multiple goroutines to test concurrency
	numGoroutines := 10
	logsPerGoroutine := 50

	start := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			workerLogger := logger.Child(map[string]interface{}{
				"worker_id": workerID,
			})

			for j := 0; j < logsPerGoroutine; j++ {
				workerLogger.Info(ctx, fmt.Sprintf("Concurrent log from worker %d", workerID), map[string]interface{}{
					"log_index": j + 1,
					"stress_test": true,
				})
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	totalLogs := numGoroutines * logsPerGoroutine
	logsPerSecond := float64(totalLogs) / duration.Seconds()

	fmt.Printf("✅ Stress test completed: %d logs from %d goroutines in %v (%.2f logs/sec)\n", 
		totalLogs, numGoroutines, duration, logsPerSecond)
}

// Main function with CLI arguments support
func init() {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())
}

// Additional CLI support (uncomment and modify main() to use)
/*
func main() {
	apiKey := os.Getenv("CHECKLOGS_API_KEY")
	if apiKey == "" {
		log.Fatal("CHECKLOGS_API_KEY environment variable is required")
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "basic":
			runBasicExample(apiKey)
		case "performance":
			performanceTest(apiKey)
		case "stress":
			stressTest(apiKey)
		case "benchmark":
			if len(os.Args) > 2 {
				if numLogs, err := strconv.Atoi(os.Args[2]); err == nil {
					benchmarkLogging(apiKey, numLogs)
				} else {
					fmt.Println("Invalid number of logs for benchmark")
				}
			} else {
				benchmarkLogging(apiKey, 1000)
			}
		default:
			fmt.Println("Available commands: basic, performance, stress, benchmark [num_logs]")
		}
	} else {
		// Run all examples
		runBasicExample(apiKey)
		runClientExample(apiKey)
		runLoggerExample(apiKey)
		runChildLoggerExample(apiKey)
		runTimerExample(apiKey)
		runErrorHandlingExample(apiKey)
		runRetryQueueExample(apiKey)
		runStatisticsExample(apiKey)
		runWebApplicationExample(apiKey)
		fmt.Println("\n✅ All examples completed successfully!")
	}
}
*/