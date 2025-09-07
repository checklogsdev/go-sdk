package checklogs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"
)

// HTTPClient interface for HTTP operations (allows for mocking in tests)
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// RetryQueue interface for managing retry operations
type RetryQueue interface {
	Add(data LogData)
	GetAll() []LogData
	Clear()
	Size() int
}

// StatsManager interface for managing statistics
type StatsManager interface {
	IncrementLogs()
	IncrementErrors()
	GetStats() Stats
}

// Custom error types

// APIError represents an API-related error
type APIError struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	Response   string `json:"response"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API Error %d: %s", e.StatusCode, e.Message)
}

// IsAuthError returns true if this is an authentication error
func (e *APIError) IsAuthError() bool {
	return e.StatusCode == 401 || e.StatusCode == 403
}

// IsRateLimitError returns true if this is a rate limit error
func (e *APIError) IsRateLimitError() bool {
	return e.StatusCode == 429
}

// NetworkError represents a network-related error
type NetworkError struct {
	Message string `json:"message"`
	Cause   error  `json:"cause"`
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("Network Error: %s", e.Message)
}

// IsTimeoutError returns true if this is a timeout error
func (e *NetworkError) IsTimeoutError() bool {
	return e.Cause != nil &&
		(e.Cause == context.DeadlineExceeded ||
			e.Cause.Error() == "timeout")
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("Validation Error on field '%s': %s", e.Field, e.Message)
}

// Internal implementations

// retryQueue implements RetryQueue
type retryQueue struct {
	queue []LogData
	mutex sync.RWMutex
}

func newRetryQueue() RetryQueue {
	return &retryQueue{
		queue: make([]LogData, 0),
	}
}

func (rq *retryQueue) Add(data LogData) {
	rq.mutex.Lock()
	defer rq.mutex.Unlock()
	rq.queue = append(rq.queue, data)
}

func (rq *retryQueue) GetAll() []LogData {
	rq.mutex.RLock()
	defer rq.mutex.RUnlock()
	result := make([]LogData, len(rq.queue))
	copy(result, rq.queue)
	return result
}

func (rq *retryQueue) Clear() {
	rq.mutex.Lock()
	defer rq.mutex.Unlock()
	rq.queue = rq.queue[:0]
}

func (rq *retryQueue) Size() int {
	rq.mutex.RLock()
	defer rq.mutex.RUnlock()
	return len(rq.queue)
}

// statsManager implements StatsManager
type statsManager struct {
	totalLogs  int64
	totalErrors int64
	lastLog    time.Time
	mutex      sync.RWMutex
}

func newStatsManager() StatsManager {
	return &statsManager{}
}

func (sm *statsManager) IncrementLogs() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.totalLogs++
	sm.lastLog = time.Now()
}

func (sm *statsManager) IncrementErrors() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.totalErrors++
}

func (sm *statsManager) GetStats() Stats {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	
	errorRate := 0.0
	if sm.totalLogs > 0 {
		errorRate = float64(sm.totalErrors) / float64(sm.totalLogs) * 100
	}
	
	return Stats{
		TotalLogs: sm.totalLogs,
		LastLog:   sm.lastLog,
		ErrorRate: errorRate,
	}
}

// Client implementation functions

// newCheckLogsClient creates a new CheckLogs client
func newCheckLogsClient(apiKey string, options *ClientOptions) *CheckLogsClient {
	opts := ClientOptions{
		Timeout:         DefaultTimeout,
		ValidatePayload: true,
		BaseURL:         DefaultBaseURL,
	}

	if options != nil {
		if options.Timeout > 0 {
			opts.Timeout = options.Timeout
		}
		opts.ValidatePayload = options.ValidatePayload
		if options.BaseURL != "" {
			opts.BaseURL = options.BaseURL
		}
	}

	return &CheckLogsClient{
		apiKey: apiKey,
		options: opts,
		httpClient: &http.Client{
			Timeout: opts.Timeout,
		},
		retryQueue: newRetryQueue(),
		stats:      newStatsManager(),
	}
}

// validateLogData validates a log entry according to CheckLogs requirements
func (c *CheckLogsClient) validateLogData(data *LogData) error {
	if !c.options.ValidatePayload {
		return nil
	}

	if data.Message == "" {
		return &ValidationError{Field: "message", Message: "message is required"}
	}

	if len(data.Message) > 1024 {
		return &ValidationError{Field: "message", Message: "message must not exceed 1024 characters"}
	}

	if data.Source != "" && len(data.Source) > 100 {
		return &ValidationError{Field: "source", Message: "source must not exceed 100 characters"}
	}

	if data.Context != nil {
		contextBytes, _ := json.Marshal(data.Context)
		if len(contextBytes) > 5000 {
			return &ValidationError{Field: "context", Message: "context must not exceed 5000 characters when serialized"}
		}
	}

	// Validate log level
	validLevels := map[LogLevel]bool{
		LogLevelDebug:    true,
		LogLevelInfo:     true,
		LogLevelWarning:  true,
		LogLevelError:    true,
		LogLevelCritical: true,
	}

	if !validLevels[data.Level] {
		return &ValidationError{Field: "level", Message: "invalid log level"}
	}

	return nil
}

// sendLog sends a log entry to CheckLogs with automatic retry logic
func (c *CheckLogsClient) sendLog(ctx context.Context, data LogData) error {
	// Set default timestamp if not provided
	if data.Timestamp.IsZero() {
		data.Timestamp = time.Now()
	}

	if err := c.validateLogData(&data); err != nil {
		return err
	}

	c.stats.IncrementLogs()

	jsonData, err := json.Marshal(data)
	if err != nil {
		c.stats.IncrementErrors()
		return fmt.Errorf("failed to marshal log data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.options.BaseURL+"/api/logs", bytes.NewBuffer(jsonData))
	if err != nil {
		c.stats.IncrementErrors()
		return &NetworkError{Message: "failed to create request", Cause: err}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", "CheckLogs-Go-SDK/"+Version)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.stats.IncrementErrors()
		c.retryQueue.Add(data)
		return &NetworkError{Message: "request failed", Cause: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		c.stats.IncrementErrors()
		body, _ := io.ReadAll(resp.Body)
		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Message:    resp.Status,
			Response:   string(body),
		}

		// Add to retry queue for retriable errors
		if resp.StatusCode >= 500 || resp.StatusCode == 429 {
			c.retryQueue.Add(data)
		}

		return apiErr
	}

	return nil
}

// getLogs retrieves logs from CheckLogs
func (c *CheckLogsClient) getLogs(ctx context.Context, params GetLogsParams) (*LogsResponse, error) {
	// Build query parameters
	queryParams := url.Values{}
	
	if params.Limit > 0 {
		queryParams.Set("limit", strconv.Itoa(params.Limit))
	}
	
	if params.Level != "" {
		queryParams.Set("level", string(params.Level))
	}
	
	if !params.Since.IsZero() {
		queryParams.Set("since", params.Since.Format(time.RFC3339))
	}
	
	if !params.Until.IsZero() {
		queryParams.Set("until", params.Until.Format(time.RFC3339))
	}
	
	if params.Source != "" {
		queryParams.Set("source", params.Source)
	}
	
	if params.UserID != nil {
		queryParams.Set("user_id", strconv.FormatInt(*params.UserID, 10))
	}

	reqURL := c.options.BaseURL + "/api/logs"
	if len(queryParams) > 0 {
		reqURL += "?" + queryParams.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, &NetworkError{Message: "failed to create request", Cause: err}
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", "CheckLogs-Go-SDK/"+Version)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &NetworkError{Message: "request failed", Cause: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    resp.Status,
			Response:   string(body),
		}
	}

	var response LogsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// getStats retrieves basic statistics
func (c *CheckLogsClient) getStats(ctx context.Context) (*Stats, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.options.BaseURL+"/api/stats", nil)
	if err != nil {
		return nil, &NetworkError{Message: "failed to create request", Cause: err}
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", "CheckLogs-Go-SDK/"+Version)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &NetworkError{Message: "request failed", Cause: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    resp.Status,
			Response:   string(body),
		}
	}

	var response struct {
		Data Stats `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response.Data, nil
}

// getSummary retrieves analytics summary
func (c *CheckLogsClient) getSummary(ctx context.Context) (*Summary, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.options.BaseURL+"/api/summary", nil)
	if err != nil {
		return nil, &NetworkError{Message: "failed to create request", Cause: err}
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", "CheckLogs-Go-SDK/"+Version)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &NetworkError{Message: "request failed", Cause: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    resp.Status,
			Response:   string(body),
		}
	}

	var summary Summary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &summary, nil
}

// getErrorRate retrieves the current error rate
func (c *CheckLogsClient) getErrorRate(ctx context.Context) (float64, error) {
	summary, err := c.getSummary(ctx)
	if err != nil {
		return 0, err
	}
	return summary.Data.Analytics.ErrorRate, nil
}

// getTrend retrieves the current trend
func (c *CheckLogsClient) getTrend(ctx context.Context) (string, error) {
	summary, err := c.getSummary(ctx)
	if err != nil {
		return "", err
	}
	return summary.Data.Analytics.Trend, nil
}

// getPeakDay retrieves the peak day
func (c *CheckLogsClient) getPeakDay(ctx context.Context) (string, error) {
	summary, err := c.getSummary(ctx)
	if err != nil {
		return "", err
	}
	return summary.Data.Analytics.PeakDay, nil
}

// getRetryQueueStatus returns the retry queue status
func (c *CheckLogsClient) getRetryQueueStatus() RetryQueueStatus {
	return RetryQueueStatus{
		Count: c.retryQueue.Size(),
	}
}

// flush processes all items in the retry queue
func (c *CheckLogsClient) flush(ctx context.Context) bool {
	queue := c.retryQueue.GetAll()
	c.retryQueue.Clear()

	success := true
	for _, data := range queue {
		if err := c.sendLog(ctx, data); err != nil {
			success = false
		}
	}

	return success
}

// clearRetryQueue clears the retry queue
func (c *CheckLogsClient) clearRetryQueue() {
	c.retryQueue.Clear()
}

// Logger implementation functions

// newCheckLogsLogger creates a new CheckLogs logger
func newCheckLogsLogger(apiKey string, options *LoggerOptions) *CheckLogsLogger {
	opts := LoggerOptions{
		ClientOptions: ClientOptions{
			Timeout:         DefaultTimeout,
			ValidatePayload: true,
			BaseURL:         DefaultBaseURL,
		},
		Silent:           false,
		ConsoleOutput:    true,
		EnabledLevels:    []LogLevel{LogLevelDebug, LogLevelInfo, LogLevelWarning, LogLevelError, LogLevelCritical},
		IncludeTimestamp: true,
		IncludeHostname:  true,
	}

	if options != nil {
		// Copy client options
		if options.Timeout > 0 {
			opts.Timeout = options.Timeout
		}
		opts.ValidatePayload = options.ValidatePayload
		if options.BaseURL != "" {
			opts.BaseURL = options.BaseURL
		}

		// Copy logger-specific options
		if options.Source != "" {
			opts.Source = options.Source
		}
		if options.UserID != nil {
			opts.UserID = options.UserID
		}
		if options.DefaultContext != nil {
			opts.DefaultContext = make(map[string]interface{})
			for k, v := range options.DefaultContext {
				opts.DefaultContext[k] = v
			}
		}
		opts.Silent = options.Silent
		opts.ConsoleOutput = options.ConsoleOutput
		if len(options.EnabledLevels) > 0 {
			opts.EnabledLevels = options.EnabledLevels
		}
		opts.IncludeTimestamp = options.IncludeTimestamp
		opts.IncludeHostname = options.IncludeHostname
	}

	client := newCheckLogsClient(apiKey, &opts.ClientOptions)

	return &CheckLogsLogger{
		client:         client,
		options:        opts,
		defaultContext: opts.DefaultContext,
	}
}

// isLevelEnabled checks if a log level is enabled
func (l *CheckLogsLogger) isLevelEnabled(level LogLevel) bool {
	for _, enabledLevel := range l.options.EnabledLevels {
		if enabledLevel == level {
			return true
		}
	}
	return false
}

// buildLogData constructs a LogData struct with defaults and context
func (l *CheckLogsLogger) buildLogData(level LogLevel, message string, context map[string]interface{}) LogData {
	data := LogData{
		Message:   message,
		Level:     level,
		Source:    l.options.Source,
		UserID:    l.options.UserID,
		Context:   make(map[string]interface{}),
		Timestamp: time.Now(),
	}

	// Add hostname if enabled
	if l.options.IncludeHostname {
		if hostname, err := os.Hostname(); err == nil {
			data.Hostname = hostname
		}
	}

	// Merge default context
	if l.defaultContext != nil {
		for k, v := range l.defaultContext {
			data.Context[k] = v
		}
	}

	// Merge provided context
	if context != nil {
		for k, v := range context {
			data.Context[k] = v
		}
	}

	return data
}

// log is the internal logging method
func (l *CheckLogsLogger) log(ctx context.Context, level LogLevel, message string, context map[string]interface{}) error {
	if l.options.Silent || !l.isLevelEnabled(level) {
		return nil
	}

	data := l.buildLogData(level, message, context)

	// Console output
	if l.options.ConsoleOutput {
		timestamp := ""
		if l.options.IncludeTimestamp {
			timestamp = data.Timestamp.Format("2006-01-02 15:04:05") + " "
		}
		fmt.Printf("%s[%s] %s\n", timestamp, level, message)
	}

	return l.client.sendLog(ctx, data)
}

// createChild creates a child logger with inherited context
func (l *CheckLogsLogger) createChild(context map[string]interface{}) *CheckLogsLogger {
	newContext := make(map[string]interface{})

	// Copy parent context
	if l.defaultContext != nil {
		for k, v := range l.defaultContext {
			newContext[k] = v
		}
	}

	// Add child context
	if context != nil {
		for k, v := range context {
			newContext[k] = v
		}
	}

	// Create new logger with merged context
	childOptions := l.options
	childOptions.DefaultContext = newContext

	return &CheckLogsLogger{
		client:         l.client,
		options:        childOptions,
		defaultContext: newContext,
	}
}

// createTimer creates a new timer
func (l *CheckLogsLogger) createTimer(name, message string) *Timer {
	return &Timer{
		start:   time.Now(),
		name:    name,
		message: message,
		logger:  l,
	}
}

// Timer implementation functions

// endTimer ends the timer and logs the duration
func (t *Timer) endTimer() time.Duration {
	duration := time.Since(t.start)

	ctx := context.Background()
	context := map[string]interface{}{
		"operation":   t.name,
		"duration_ms": duration.Milliseconds(),
	}

	t.logger.Info(ctx, fmt.Sprintf("%s completed in %v", t.message, duration), context)

	return duration
}

// Helper functions

// sanitizeContext removes nil values and ensures the context is JSON serializable
func sanitizeContext(context map[string]interface{}) map[string]interface{} {
	if context == nil {
		return nil
	}

	sanitized := make(map[string]interface{})
	for k, v := range context {
		if v != nil {
			sanitized[k] = v
		}
	}

	return sanitized
}

// buildUserAgent creates a user agent string for requests
func buildUserAgent() string {
	return fmt.Sprintf("CheckLogs-Go-SDK/%s (Go)", Version)
}

// exponentialBackoff calculates the backoff duration for retry attempts
func exponentialBackoff(attempt int, baseDelay time.Duration) time.Duration {
	if attempt <= 0 {
		return baseDelay
	}
	
	// Cap at 30 seconds maximum
	maxDelay := 30 * time.Second
	delay := baseDelay * time.Duration(1<<uint(attempt))
	
	if delay > maxDelay {
		return maxDelay
	}
	
	return delay
}

// isRetriableError determines if an error should trigger a retry
func isRetriableError(err error) bool {
	switch e := err.(type) {
	case *APIError:
		// Retry on server errors and rate limits
		return e.StatusCode >= 500 || e.StatusCode == 429
	case *NetworkError:
		// Retry on network errors
		return true
	default:
		return false
	}
}

// formatLogLevel formats a log level for display
func formatLogLevel(level LogLevel) string {
	switch level {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarning:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelCritical:
		return "CRIT"
	default:
		return string(level)
	}
}

// validateAPIKey validates the format of an API key
func validateAPIKey(apiKey string) error {
	if apiKey == "" {
		return &ValidationError{Field: "api_key", Message: "API key is required"}
	}
	
	if len(apiKey) < 10 {
		return &ValidationError{Field: "api_key", Message: "API key appears to be invalid (too short)"}
	}
	
	return nil
}

// mergeContexts merges multiple context maps, with later maps taking precedence
func mergeContexts(contexts ...map[string]interface{}) map[string]interface{} {
	if len(contexts) == 0 {
		return nil
	}
	
	merged := make(map[string]interface{})
	for _, ctx := range contexts {
		if ctx != nil {
			for k, v := range ctx {
				merged[k] = v
			}
		}
	}
	
	if len(merged) == 0 {
		return nil
	}
	
	return merged
}

// getDefaultEnabledLevels returns the default enabled log levels
func getDefaultEnabledLevels() []LogLevel {
	return []LogLevel{
		LogLevelDebug,
		LogLevelInfo,
		LogLevelWarning,
		LogLevelError,
		LogLevelCritical,
	}
}

// isValidLogLevel checks if a log level is valid
func isValidLogLevel(level LogLevel) bool {
	validLevels := map[LogLevel]bool{
		LogLevelDebug:    true,
		LogLevelInfo:     true,
		LogLevelWarning:  true,
		LogLevelError:    true,
		LogLevelCritical: true,
	}
	
	return validLevels[level]
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	
	if maxLength <= 3 {
		return s[:maxLength]
	}
	
	return s[:maxLength-3] + "..."
}

// getContextSize calculates the size of a context map when JSON serialized
func getContextSize(context map[string]interface{}) int {
	if context == nil {
		return 0
	}
	
	jsonBytes, err := json.Marshal(context)
	if err != nil {
		return 0
	}
	
	return len(jsonBytes)
}

// addProcessInfo adds process information to the context
func addProcessInfo(context map[string]interface{}) {
	if context == nil {
		return
	}
	
	context["process_id"] = os.Getpid()
	context["go_version"] = fmt.Sprintf("go%s", os.Getenv("GO_VERSION"))
}

// addTimestampToContext adds a timestamp to the context if not already present
func addTimestampToContext(context map[string]interface{}) {
	if context == nil {
		return
	}
	
	if _, exists := context["timestamp"]; !exists {
		context["timestamp"] = time.Now().Unix()
	}
}

// cleanupContext removes empty or nil values from context
func cleanupContext(context map[string]interface{}) map[string]interface{} {
	if context == nil {
		return nil
	}
	
	cleaned := make(map[string]interface{})
	for k, v := range context {
		if v != nil {
			// Check for empty strings
			if str, ok := v.(string); ok && str == "" {
				continue
			}
			
			// Check for empty slices
			if slice, ok := v.([]interface{}); ok && len(slice) == 0 {
				continue
			}
			
			// Check for empty maps
			if m, ok := v.(map[string]interface{}); ok && len(m) == 0 {
				continue
			}
			
			cleaned[k] = v
		}
	}
	
	if len(cleaned) == 0 {
		return nil
	}
	
	return cleaned
}

// validateURL validates a URL format
func validateURL(urlStr string) error {
	if urlStr == "" {
		return &ValidationError{Field: "url", Message: "URL cannot be empty"}
	}
	
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return &ValidationError{Field: "url", Message: "invalid URL format"}
	}
	
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return &ValidationError{Field: "url", Message: "URL must use http or https scheme"}
	}
	
	return nil
}

// setDefaultTimeout sets a default timeout if none is provided
func setDefaultTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return DefaultTimeout
	}
	return timeout
}

// buildHeaders builds common HTTP headers for requests
func buildHeaders(apiKey string) map[string]string {
	return map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + apiKey,
		"User-Agent":    buildUserAgent(),
		"Accept":        "application/json",
	}
}

// parseTimeParam parses a time parameter from various formats
func parseTimeParam(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, nil
	}
	
	// Try RFC3339 format first
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t, nil
	}
	
	// Try other common formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}
	
	return time.Time{}, &ValidationError{
		Field:   "time",
		Message: fmt.Sprintf("unable to parse time: %s", timeStr),
	}
}