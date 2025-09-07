// Package checklogs provides a Go SDK for CheckLogs.dev
// Official Go SDK for CheckLogs.dev - A powerful log monitoring system.
package checklogs

import (
	"context"
	"time"
)

const (
	// Version of the CheckLogs Go SDK
	Version = "1.0.0"
	
	// DefaultBaseURL is the default API endpoint
	DefaultBaseURL = "https://checklogs.dev/api/logs"
	
	// DefaultTimeout is the default request timeout
	DefaultTimeout = 30 * time.Second
)

// LogLevel represents the severity level of a log entry
type LogLevel string

const (
	LogLevelDebug    LogLevel = "debug"
	LogLevelInfo     LogLevel = "info"
	LogLevelWarning  LogLevel = "warning"
	LogLevelError    LogLevel = "error"
	LogLevelCritical LogLevel = "critical"
)

// LogData represents a log entry to be sent to CheckLogs
type LogData struct {
	Message   string                 `json:"message"`
	Level     LogLevel               `json:"level"`
	Source    string                 `json:"source,omitempty"`
	UserID    *int64                 `json:"user_id,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Hostname  string                 `json:"hostname,omitempty"`
}

// ClientOptions represents configuration options for CheckLogsClient
type ClientOptions struct {
	Timeout         time.Duration `json:"timeout"`
	ValidatePayload bool          `json:"validate_payload"`
	BaseURL         string        `json:"base_url"`
}

// LoggerOptions represents configuration options for CheckLogsLogger
type LoggerOptions struct {
	ClientOptions
	Source           string                 `json:"source"`
	UserID           *int64                 `json:"user_id"`
	DefaultContext   map[string]interface{} `json:"default_context"`
	Silent           bool                   `json:"silent"`
	ConsoleOutput    bool                   `json:"console_output"`
	EnabledLevels    []LogLevel             `json:"enabled_levels"`
	IncludeTimestamp bool                   `json:"include_timestamp"`
	IncludeHostname  bool                   `json:"include_hostname"`
}

// CheckLogsClient is the main client for the CheckLogs API
type CheckLogsClient struct {
	apiKey     string
	options    ClientOptions
	httpClient HTTPClient
	retryQueue RetryQueue
	stats      StatsManager
}

// CheckLogsLogger provides advanced logging functionality with convenience methods
type CheckLogsLogger struct {
	client         *CheckLogsClient
	options        LoggerOptions
	defaultContext map[string]interface{}
}

// Timer represents a timing operation for measuring execution time
type Timer struct {
	start    time.Time
	name     string
	message  string
	logger   *CheckLogsLogger
}

// Stats represents basic statistics data
type Stats struct {
	TotalLogs int64     `json:"total_logs"`
	LastLog   time.Time `json:"last_log"`
	ErrorRate float64   `json:"error_rate"`
}

// Summary represents analytics summary data
type Summary struct {
	Data struct {
		Analytics struct {
			ErrorRate float64 `json:"error_rate"`
			Trend     string  `json:"trend"`
			PeakDay   string  `json:"peak_day"`
		} `json:"analytics"`
	} `json:"data"`
}

// RetryQueueStatus represents the status of the retry queue
type RetryQueueStatus struct {
	Count      int       `json:"count"`
	LastRetry  time.Time `json:"last_retry,omitempty"`
	NextRetry  time.Time `json:"next_retry,omitempty"`
	Failed     int       `json:"failed"`
	Successful int       `json:"successful"`
}

// GetLogsParams represents parameters for retrieving logs
type GetLogsParams struct {
	Limit  int       `json:"limit,omitempty"`
	Level  LogLevel  `json:"level,omitempty"`
	Since  time.Time `json:"since,omitempty"`
	Until  time.Time `json:"until,omitempty"`
	Source string    `json:"source,omitempty"`
	UserID *int64    `json:"user_id,omitempty"`
}

// LogsResponse represents the response when retrieving logs
type LogsResponse struct {
	Data []LogData `json:"data"`
	Meta struct {
		Total      int  `json:"total"`
		Page       int  `json:"page"`
		PerPage    int  `json:"per_page"`
		HasMore    bool `json:"has_more"`
		NextCursor *string `json:"next_cursor"`
	} `json:"meta"`
}

// NewCheckLogsClient creates a new CheckLogs client with the specified API key and options
func NewCheckLogsClient(apiKey string, options *ClientOptions) *CheckLogsClient {
	return newCheckLogsClient(apiKey, options)
}

// NewCheckLogsLogger creates a new CheckLogs logger with the specified API key and options
func NewCheckLogsLogger(apiKey string, options *LoggerOptions) *CheckLogsLogger {
	return newCheckLogsLogger(apiKey, options)
}

// CreateLogger is a convenience function to create a logger with default options
func CreateLogger(apiKey string, options *LoggerOptions) *CheckLogsLogger {
	return NewCheckLogsLogger(apiKey, options)
}

// Client methods

// Log sends a log entry to CheckLogs
func (c *CheckLogsClient) Log(ctx context.Context, data LogData) error {
	return c.sendLog(ctx, data)
}

// GetLogs retrieves logs from CheckLogs based on the provided parameters
func (c *CheckLogsClient) GetLogs(ctx context.Context, params GetLogsParams) (*LogsResponse, error) {
	return c.getLogs(ctx, params)
}

// GetStats retrieves basic statistics from CheckLogs
func (c *CheckLogsClient) GetStats(ctx context.Context) (*Stats, error) {
	return c.getStats(ctx)
}

// GetSummary retrieves analytics summary from CheckLogs
func (c *CheckLogsClient) GetSummary(ctx context.Context) (*Summary, error) {
	return c.getSummary(ctx)
}

// GetErrorRate retrieves the current error rate
func (c *CheckLogsClient) GetErrorRate(ctx context.Context) (float64, error) {
	return c.getErrorRate(ctx)
}

// GetTrend retrieves the current trend information
func (c *CheckLogsClient) GetTrend(ctx context.Context) (string, error) {
	return c.getTrend(ctx)
}

// GetPeakDay retrieves the peak day information
func (c *CheckLogsClient) GetPeakDay(ctx context.Context) (string, error) {
	return c.getPeakDay(ctx)
}

// GetRetryQueueStatus returns the current status of the retry queue
func (c *CheckLogsClient) GetRetryQueueStatus() RetryQueueStatus {
	return c.getRetryQueueStatus()
}

// Flush waits for all pending logs in the retry queue to be sent
func (c *CheckLogsClient) Flush(ctx context.Context) bool {
	return c.flush(ctx)
}

// ClearRetryQueue clears all pending logs from the retry queue
func (c *CheckLogsClient) ClearRetryQueue() {
	c.clearRetryQueue()
}

// Logger methods

// Debug logs a debug message
func (l *CheckLogsLogger) Debug(ctx context.Context, message string, context ...map[string]interface{}) error {
	return l.log(ctx, LogLevelDebug, message, mergeContext(context...))
}

// Info logs an info message
func (l *CheckLogsLogger) Info(ctx context.Context, message string, context ...map[string]interface{}) error {
	return l.log(ctx, LogLevelInfo, message, mergeContext(context...))
}

// Warning logs a warning message
func (l *CheckLogsLogger) Warning(ctx context.Context, message string, context ...map[string]interface{}) error {
	return l.log(ctx, LogLevelWarning, message, mergeContext(context...))
}

// Error logs an error message
func (l *CheckLogsLogger) Error(ctx context.Context, message string, context ...map[string]interface{}) error {
	return l.log(ctx, LogLevelError, message, mergeContext(context...))
}

// Critical logs a critical message
func (l *CheckLogsLogger) Critical(ctx context.Context, message string, context ...map[string]interface{}) error {
	return l.log(ctx, LogLevelCritical, message, mergeContext(context...))
}

// Child creates a child logger with inherited context
func (l *CheckLogsLogger) Child(context map[string]interface{}) *CheckLogsLogger {
	return l.createChild(context)
}

// Time creates a timer for measuring execution time
func (l *CheckLogsLogger) Time(name, message string) *Timer {
	return l.createTimer(name, message)
}

// GetClient returns the underlying CheckLogsClient
func (l *CheckLogsLogger) GetClient() *CheckLogsClient {
	return l.client
}

// Timer methods

// End ends the timer and logs the duration
func (t *Timer) End() time.Duration {
	return t.endTimer()
}

// GetDuration returns the current duration without ending the timer
func (t *Timer) GetDuration() time.Duration {
	return time.Since(t.start)
}

// Helper function to merge multiple context maps
func mergeContext(contexts ...map[string]interface{}) map[string]interface{} {
	if len(contexts) == 0 {
		return nil
	}
	
	if len(contexts) == 1 {
		return contexts[0]
	}
	
	merged := make(map[string]interface{})
	for _, ctx := range contexts {
		if ctx != nil {
			for k, v := range ctx {
				merged[k] = v
			}
		}
	}
	
	return merged
}