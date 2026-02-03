# Changelog

All notable changes to Humantime will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-02-03

### Breaking Changes

This release simplifies Humantime to its core purpose: tracking time on projects.

**Removed features:**
- Tasks (hierarchical `project/task` syntax)
- Goals and goal tracking
- Reminders and notifications
- Webhooks
- Background daemon
- Import command
- Configuration file
- Structured logging/observability

**Command changes:**
- Binary renamed from `humantime` to `ht`
- `ht start` no longer accepts task syntax
- `ht project delete` now archives (soft delete)

### Why v1.0.0?

Previous versions accumulated features that added complexity without proportional value. This release strips Humantime back to what matters:

- **Projects** - Containers for your work
- **Blocks** - Time entries with start/end times

That's it. No tasks, no goals, no reminders. Just time tracking.

### Migration from v0.x

1. Export your data first: `humantime export -o backup.json`
2. Install v1.0.0
3. Your projects and blocks are preserved
4. Tasks will need to be tracked as separate projects or via notes

### Added
- `ht` as the primary command (shorter, faster)
- Project archiving (soft delete)
- User-facing documentation in `docs/`

### Removed
- ~28,000 lines of code
- Task model and commands
- Goal model and commands
- Reminder system
- Webhook system
- Daemon process
- Config file support
- Import command
- Structured logging infrastructure
- Health check endpoints

---

## Previous Releases (v0.x)

The v0.x series is available on the `legacy` branch. These versions included additional features that have been removed in v1.0.0.

### [0.4.3] - 2026-01-xx
- Last release before simplification

### [0.3.0] - 2026-01-08
- Reminder system with daemon-based notifications
- Discord and generic webhook integrations
- Natural language deadline parsing

### [0.2.0] - 2025-12-15
- Project and task management
- Time block editing and deletion
- Export to JSON and CSV formats

### [0.1.0] - 2025-11-01
- Initial release
- Basic time tracking (start/stop)
- Project creation
