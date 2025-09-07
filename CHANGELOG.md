# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2024-12-XX

### Added
- Initial release of CheckLogs Go SDK
- Core logging functionality with multiple levels (Debug, Info, Warning, Error, Critical)
- Child loggers with inherited context
- Timer functionality for performance measurement
- Automatic retry queue for failed requests
- Comprehensive error handling with typed errors
- Data validation and sanitization
- Context support for metadata enrichment
- Hostname and timestamp automatic inclusion
- Console output option
- Silent mode for testing
- Configurable timeouts and base URLs

### Features
- Simple API with `CreateLogger()` and `NewLogger()` functions
- Support for custom options (source, user ID, default context)
- Thread-safe operations
- Comprehensive examples and documentation
- Windows, macOS, and Linux support

### Security
- API key authentication
- Request validation
- Secure HTTPS communication
- No sensitive data logging