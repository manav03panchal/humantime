# Contributing to Humantime

Welcome to Humantime! We are excited to have you contribute to this CLI time tracking tool. This document outlines our development practices, code standards, and contribution process.

## Development Setup

### Prerequisites

- **Go 1.21+**: Humantime is built with Go. Ensure you have Go 1.21 or later installed.
- **Git**: For version control and submitting contributions.

### Getting Started

1. **Clone the repository**:
   ```bash
   git clone https://github.com/your-org/humantime.git
   cd humantime
   ```

2. **Download dependencies**:
   ```bash
   go mod download
   ```

3. **Verify installation**:
   ```bash
   go build ./...
   ```

## Code Style

We follow standard Go conventions and enforce code quality through automated tooling.

### Formatting

- **gofmt**: All code must be formatted with `gofmt`. Run before committing:
  ```bash
  gofmt -w .
  ```

### Linting

- **golangci-lint**: We use golangci-lint for comprehensive static analysis:
  ```bash
  golangci-lint run
  ```

### Naming Conventions

- **Commands**: lowercase, verb-first (`start`, `end`, `list`)
- **Packages**: lowercase, singular (`model` not `models`)
- **Files**: snake_case (`active_block.go`)
- **Types**: PascalCase (`TimeBlock`)
- **Functions**: PascalCase for exported, camelCase for internal

## Test Requirements

All contributions must include appropriate tests. Run the full test suite with:

```bash
go test ./...
```

### Test Priorities

1. **Contract tests**: API boundaries
2. **Integration tests**: Real database, real I/O
3. **End-to-end tests**: Full user workflows
4. **Unit tests**: Isolated logic

### Test-First Development

We follow strict Test-Driven Development (TDD):

1. Write tests defining expected behavior
2. Validate tests FAIL (Red phase)
3. Implement code to make tests pass (Green phase)
4. Refactor while maintaining passing tests (Refactor phase)

## Pull Request Process

1. **Create a feature branch** from the main development branch
2. **Write tests first** following TDD principles
3. **Implement your changes** with minimal code to pass tests
4. **Ensure all quality gates pass**:
   - All tests pass
   - No linting errors (`golangci-lint`)
   - Code formatted (`gofmt`)
   - Constitution principles satisfied
5. **Submit a pull request** with a clear description of changes
6. **Address review feedback** promptly

### PR Requirements

- All changes require specification traceability
- Tests must cover acceptance criteria
- No decrease in test coverage
- Constitution compliance verified

## Issue Reporting Guidelines

When reporting issues, please include:

1. **Clear title**: Concise description of the issue
2. **Environment**: Go version, OS, Humantime version
3. **Steps to reproduce**: Detailed steps to recreate the issue
4. **Expected behavior**: What you expected to happen
5. **Actual behavior**: What actually happened
6. **Logs/output**: Any relevant error messages or logs

Use appropriate labels when creating issues:
- `bug`: Something is not working as expected
- `enhancement`: New feature or improvement
- `documentation`: Documentation updates needed
- `question`: Clarification needed

## Constitution Principles

Humantime is built on core principles that guide all development. Contributors should understand and follow these:

### I. Library-First Architecture

Every feature MUST begin as a standalone, independently testable library before CLI integration. Libraries must be self-contained, well-documented, and testable in isolation.

### II. CLI-First Interface

All functionality MUST be accessible through the command-line interface. Support both human-readable and JSON output formats (`-f cli`, `-f json`). Provide intuitive command aliases.

### III. Natural Language First

Prioritize natural language over complex flag syntax. Parse expressions like "4 hours ago", "last week", "yesterday at 3pm". Never require users to memorize arcane syntax.

### IV. Test-First Development

This is NON-NEGOTIABLE. All implementation MUST follow strict TDD. No implementation code shall be written before tests are written and validated to fail.

### V. Embedded-First Storage

Data persistence MUST use embedded storage with zero external dependencies. The tool must work offline with zero configuration.

### VI. Simplicity Over Cleverness

Favor simplicity in all implementations:
- No premature abstractions
- No "might need later" features
- Direct framework usage over wrapper patterns
- Duplicate before abstracting

### VII. Observability & Debuggability

All operations MUST be observable and debuggable with structured logging, traceable state changes, and clear error messages with actionable guidance.

---

For the complete constitution and detailed guidelines, see `.specify/memory/constitution.md`.

Thank you for contributing to Humantime!
