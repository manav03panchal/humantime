# Changelog

All notable changes to Humantime will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### Error Handling Infrastructure
- New `internal/errors` package with UserError, SystemError, and RecoverableError types
- Error categorization and classification system
- Context chain support for error wrapping with stack traces
- Recovery suggestions for common error scenarios
- Debug mode formatting for detailed error output

#### Testing Infrastructure
- Property-based tests for time, duration, and deadline parsers
- Fuzz tests for parsers, validation, and logging
- Integration tests for all major workflows (tracking, reminders, webhooks, export)
- Edge case tests for DST, timezones, long inputs, invalid configs
- Security tests for injection, path traversal, and credential masking
- Performance benchmarks for startup, memory, and database operations

#### Security Hardening
- Input sanitization functions in `internal/validate/sanitize.go`
- Webhook URL validation (SSRF protection, HTTPS enforcement)
- Path traversal detection and prevention
- Sensitive data masking in logs (tokens, secrets, URLs)
- File permission enforcement (0600 for data, 0700 for dirs)

#### Observability
- Structured logging with slog in `internal/logging`
- Request ID correlation for tracing operations
- Health check endpoint with JSON output
- In-memory metrics tracking (notifications, reminders, errors)
- Debug trace output with timing information

#### Reliability
- Database integrity checking on startup
- Atomic write operations to prevent data corruption
- File-based locking for concurrent access prevention
- Retry queue with exponential backoff for webhook failures
- Disk space checking before write operations

### Changed
- Relaxed performance test thresholds for CI environments
- Improved error messages with consistent format (what + why + how to fix)

### Fixed
- Various edge cases in parser error handling
- Timeout handling for external HTTP requests

## [0.3.0] - 2026-01-08

### Added
- Reminder system with daemon-based notifications
- Discord and generic webhook integrations
- Natural language deadline parsing
- Recurring reminder support

## [0.2.0] - 2025-12-15

### Added
- Project and task management
- Time block editing and deletion
- Export to JSON and CSV formats
- Import from JSON and CSV

## [0.1.0] - 2025-11-01

### Added
- Initial release
- Basic time tracking (start/stop)
- Project creation
- Simple stats and blocks listing
