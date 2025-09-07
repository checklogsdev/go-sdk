// Package checklogs provides a simple Go SDK for CheckLogs.dev
package checklogs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	Version    = "1.0.0"
	DefaultURL = "https://checklogs.dev"
)

// LogLevel represents the severity level of a log entry
type LogLevel string

const (
	Debug    LogLevel = "debug"
	Info     LogLevel = "info"
	Warning  LogLevel = "warning"
	Error    LogLevel = "error"
	Critical LogLevel = "critical"
)

// LogData represents a log entry
type LogData struct {
	Message   string                 `json:"message"`
	Level     LogLevel               `json:"level"`
	Source    string                 `json:"source,omitempty"`
	UserID    *int64                 `json:"user_id,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Hostname  string                 `json:"hostname,omitempty"`
}

// Options represents configuration for the logger
type Options struct {
	Source        string                 `json:"source"`
	UserID        *int64                 `json:"user_id"`
	Context       map[string]interface{} `json:"default_context"`
	Silent        bool                   `json:"silent"`
	ConsoleOutput bool                   `json:"console_output"`
	BaseURL       string                 `json:"base_url"`
	Timeout       time.Duration          `json:"timeout"`
}

// Logger represents the CheckLogs logger
type Logger struct {
	apiKey     string
	options    Options
	httpClient *http.Client
	retryQueue []LogData
	mutex      sync.RWMutex
}

// Timer represents a timing operation
type Timer struct {
	start   time.Time
	name    string
	message string
	logger  *Logger
}

// Custom error types
type CheckLogsError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

func (e *CheckLogsError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// NewLogger creates a new CheckLogs logger
func NewLogger(apiKey string, opts *Options) *Logger {
	// Set default options
	options := Options{
		ConsoleOutput: true,
		BaseURL:       DefaultURL,
		Timeout:       30 * time.Second,
	}

	// Override with provided options
	if opts != nil {
		if opts.Source != "" {
			options.Source = opts.Source
		}
		if opts.UserID != nil {
			options.UserID = opts.UserID
		}
		if opts.Context != nil {
			options.Context = opts.Context
		}
		options.Silent = opts.Silent
		options.ConsoleOutput = opts.ConsoleOutput
		if opts.BaseURL != "" {
			options.BaseURL = opts.BaseURL
		}
		if opts.Timeout > 0 {
			options.Timeout = opts.Timeout
		}
	}

	return &Logger{
		apiKey:     apiKey,
		options:    options,
		httpClient: &http.Client{Timeout: options.Timeout},
		retryQueue: make([]LogData, 0),
	}
}

// NewLoggerWithValidation creates a new CheckLogs logger and validates the API key
func NewLoggerWithValidation(apiKey string, opts *Options) (*Logger, error) {
	logger := NewLogger(apiKey, opts)
	
	// Valider la clé API si elle est fournie
	if apiKey != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		if err := logger.ValidateAPIKey(ctx); err != nil {
			return nil, fmt.Errorf("API key validation failed: %w", err)
		}
	}
	
	return logger, nil
}

// CreateLogger is a convenience function to create a logger with minimal config
func CreateLogger(apiKey string) *Logger {
	return NewLogger(apiKey, nil)
}

// CreateLoggerWithValidation is a convenience function to create and validate a logger
func CreateLoggerWithValidation(apiKey string) (*Logger, error) {
	return NewLoggerWithValidation(apiKey, nil)
}

// ValidateAPIKey validates the API key by making a test request
func (l *Logger) ValidateAPIKey(ctx context.Context) error {
	if l.apiKey == "" {
		return &CheckLogsError{Type: "ConfigurationError", Message: "API key is required"}
	}

	// Test avec une requête de validation
	req, err := http.NewRequestWithContext(ctx, "GET", l.options.BaseURL+"/api/validate", nil)
	if err != nil {
		return &CheckLogsError{Type: "NetworkError", Message: "Cannot create validation request: " + err.Error()}
	}

	req.Header.Set("Authorization", "Bearer "+l.apiKey)
	req.Header.Set("User-Agent", "CheckLogs-Go-SDK/"+Version)

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return &CheckLogsError{Type: "NetworkError", Message: "Cannot reach CheckLogs API: " + err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return &CheckLogsError{Type: "AuthenticationError", Message: "Invalid API key", Code: 401}
	}
	
	if resp.StatusCode == 403 {
		return &CheckLogsError{Type: "AuthorizationError", Message: "API key does not have required permissions", Code: 403}
	}
	
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return &CheckLogsError{Type: "APIError", Message: fmt.Sprintf("API validation failed (HTTP %d): %s", resp.StatusCode, string(body)), Code: resp.StatusCode}
	}

	return nil
}

// GetStatus returns the current status of the logger and API connection
func (l *Logger) GetStatus(ctx context.Context) (map[string]interface{}, error) {
	status := map[string]interface{}{
		"api_key_present":  l.apiKey != "",
		"retry_queue_size": l.GetRetryQueueSize(),
		"base_url":         l.options.BaseURL,
		"api_reachable":    false,
		"api_key_valid":    false,
		"sdk_version":      Version,
	}

	if l.apiKey == "" {
		status["error"] = "No API key provided"
		return status, nil
	}

	// Test de connectivité
	req, err := http.NewRequestWithContext(ctx, "GET", l.options.BaseURL+"/api/status", nil)
	if err != nil {
		status["error"] = "Cannot create request: " + err.Error()
		return status, nil
	}

	req.Header.Set("Authorization", "Bearer "+l.apiKey)
	req.Header.Set("User-Agent", "CheckLogs-Go-SDK/"+Version)

	resp, err := l.httpClient.Do(req)
	if err != nil {
		status["error"] = "Cannot reach API: " + err.Error()
		return status, nil
	}
	defer resp.Body.Close()

	status["api_reachable"] = true

	if resp.StatusCode == 200 {
		status["api_key_valid"] = true
	} else if resp.StatusCode == 401 {
		status["error"] = "Invalid API key"
	} else if resp.StatusCode == 403 {
		status["error"] = "API key does not have required permissions"
	} else {
		body, _ := io.ReadAll(resp.Body)
		status["error"] = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return status, nil
}

// validateLogData validates a log entry
func (l *Logger) validateLogData(data *LogData) error {
	if data.Message == "" {
		return &CheckLogsError{Type: "ValidationError", Message: "message is required"}
	}
	if len(data.Message) > 1024 {
		return &CheckLogsError{Type: "ValidationError", Message: "message too long (max 1024 characters)"}
	}
	if data.Source != "" && len(data.Source) > 100 {
		return &CheckLogsError{Type: "ValidationError", Message: "source too long (max 100 characters)"}
	}
	return nil
}

// sendLog sends a log entry to CheckLogs
func (l *Logger) sendLog(ctx context.Context, data LogData) error {
	// Set defaults
	if data.Timestamp.IsZero() {
		data.Timestamp = time.Now()
	}
	if data.Source == "" && l.options.Source != "" {
		data.Source = l.options.Source
	}
	if data.UserID == nil && l.options.UserID != nil {
		data.UserID = l.options.UserID
	}

	// Add hostname
	if hostname, err := os.Hostname(); err == nil {
		data.Hostname = hostname
	}

	// Merge default context
	if l.options.Context != nil {
		if data.Context == nil {
			data.Context = make(map[string]interface{})
		}
		for k, v := range l.options.Context {
			if _, exists := data.Context[k]; !exists {
				data.Context[k] = v
			}
		}
	}

	// Validate
	if err := l.validateLogData(&data); err != nil {
		return err
	}

	// Console output
	if l.options.ConsoleOutput && !l.options.Silent {
		fmt.Printf("[%s] %s: %s\n", data.Timestamp.Format("15:04:05"), data.Level, data.Message)
	}

	// Skip HTTP request if no API key
	if l.apiKey == "" {
		err := &CheckLogsError{Type: "ConfigurationError", Message: "API key is required"}
		// Afficher l'erreur même en mode console
		if !l.options.Silent {
			fmt.Printf("[CHECKLOGS ERROR] %s\n", err.Message)
		}
		return err
	}

	// Skip HTTP request in silent mode
	if l.options.Silent {
		return nil
	}

	// Prepare JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return &CheckLogsError{Type: "SerializationError", Message: err.Error()}
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", l.options.BaseURL+"/api/logs", bytes.NewBuffer(jsonData))
	if err != nil {
		l.addToRetryQueue(data)
		return &CheckLogsError{Type: "NetworkError", Message: err.Error()}
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.apiKey)
	req.Header.Set("User-Agent", "CheckLogs-Go-SDK/"+Version)

	// Send request
	resp, err := l.httpClient.Do(req)
	if err != nil {
		l.addToRetryQueue(data)
		return &CheckLogsError{Type: "NetworkError", Message: err.Error()}
	}
	defer resp.Body.Close()

	// Handle response with improved error handling
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		
		var errType string
		var shouldRetry bool
		
		switch resp.StatusCode {
		case 401:
			errType = "AuthenticationError"
			shouldRetry = false
		case 403:
			errType = "AuthorizationError"
			shouldRetry = false
		case 429:
			errType = "RateLimitError"
			shouldRetry = true
		case 400:
			errType = "ValidationError"
			shouldRetry = false
		default:
			if resp.StatusCode >= 500 {
				errType = "ServerError"
				shouldRetry = true
			} else {
				errType = "ClientError"
				shouldRetry = false
			}
		}

		err := &CheckLogsError{
			Type:    errType,
			Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
			Code:    resp.StatusCode,
		}

		// Retry only on certain errors
		if shouldRetry {
			l.addToRetryQueue(data)
		}

		// Show critical errors even in console mode
		if (errType == "AuthenticationError" || errType == "AuthorizationError") && !l.options.Silent {
			fmt.Printf("[CHECKLOGS ERROR] %s\n", err.Message)
		}

		return err
	}

	return nil
}

// addToRetryQueue adds a log to the retry queue
func (l *Logger) addToRetryQueue(data LogData) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.retryQueue = append(l.retryQueue, data)
}

// GetRetryQueueSize returns the number of logs in the retry queue
func (l *Logger) GetRetryQueueSize() int {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return len(l.retryQueue)
}

// FlushRetryQueue attempts to send all logs in the retry queue
func (l *Logger) FlushRetryQueue(ctx context.Context) int {
	l.mutex.Lock()
	queue := make([]LogData, len(l.retryQueue))
	copy(queue, l.retryQueue)
	l.retryQueue = l.retryQueue[:0] // Clear queue
	l.mutex.Unlock()

	success := 0
	for _, data := range queue {
		if err := l.sendLog(ctx, data); err == nil {
			success++
		}
	}
	return success
}

// ClearRetryQueue clears the retry queue
func (l *Logger) ClearRetryQueue() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.retryQueue = l.retryQueue[:0]
}

// Log methods for different levels

// Debug logs a debug message
func (l *Logger) Debug(ctx context.Context, message string, context ...map[string]interface{}) error {
	return l.log(ctx, Debug, message, context...)
}

// Info logs an info message
func (l *Logger) Info(ctx context.Context, message string, context ...map[string]interface{}) error {
	return l.log(ctx, Info, message, context...)
}

// Warning logs a warning message
func (l *Logger) Warning(ctx context.Context, message string, context ...map[string]interface{}) error {
	return l.log(ctx, Warning, message, context...)
}

// Error logs an error message
func (l *Logger) Error(ctx context.Context, message string, context ...map[string]interface{}) error {
	return l.log(ctx, Error, message, context...)
}

// Critical logs a critical message
func (l *Logger) Critical(ctx context.Context, message string, context ...map[string]interface{}) error {
	return l.log(ctx, Critical, message, context...)
}

// log is the internal logging method
func (l *Logger) log(ctx context.Context, level LogLevel, message string, contexts ...map[string]interface{}) error {
	data := LogData{
		Message: message,
		Level:   level,
	}

	// Merge contexts
	if len(contexts) > 0 {
		data.Context = make(map[string]interface{})
		for _, ctx := range contexts {
			if ctx != nil {
				for k, v := range ctx {
					data.Context[k] = v
				}
			}
		}
	}

	return l.sendLog(ctx, data)
}

// Child creates a child logger with additional context
func (l *Logger) Child(context map[string]interface{}) *Logger {
	newContext := make(map[string]interface{})

	// Copy parent context
	if l.options.Context != nil {
		for k, v := range l.options.Context {
			newContext[k] = v
		}
	}

	// Add child context
	if context != nil {
		for k, v := range context {
			newContext[k] = v
		}
	}

	// Create child options
	childOptions := l.options
	childOptions.Context = newContext

	return &Logger{
		apiKey:     l.apiKey,
		options:    childOptions,
		httpClient: l.httpClient,
		retryQueue: make([]LogData, 0),
	}
}

// Time creates a timer for measuring execution time
func (l *Logger) Time(name, message string) *Timer {
	return &Timer{
		start:   time.Now(),
		name:    name,
		message: message,
		logger:  l,
	}
}

// End ends the timer and logs the duration
func (t *Timer) End() time.Duration {
	duration := time.Since(t.start)

	ctx := context.Background()
	context := map[string]interface{}{
		"operation":   t.name,
		"duration_ms": duration.Milliseconds(),
	}

	t.logger.Info(ctx, fmt.Sprintf("%s completed in %v", t.message, duration), context)

	return duration
}

// GetDuration returns the current duration without ending the timer
func (t *Timer) GetDuration() time.Duration {
	return time.Since(t.start)
}

// IsValidLevel checks if a log level is valid
func IsValidLevel(level LogLevel) bool {
	switch level {
	case Debug, Info, Warning, Error, Critical:
		return true
	default:
		return false
	}
}

// ParseLevel parses a string into a LogLevel
func ParseLevel(s string) (LogLevel, error) {
	level := LogLevel(s)
	if IsValidLevel(level) {
		return level, nil
	}
	return "", &CheckLogsError{Type: "ValidationError", Message: "invalid log level: " + s}
}