# Humantime

A next-generation CLI time tracking tool inspired by [Zeit](https://github.com/mrusme/zeit). Track your work with natural language commands.

## Features

- **Natural Language Interface** - Use intuitive commands like `start on myproject` and `stop`
- **Hierarchical Projects** - Organize work with projects and tasks (`project/task`)
- **Flexible Time Tracking** - Start/stop tracking, or log completed blocks with custom timestamps
- **Time Range Queries** - Query blocks and stats with natural phrases like "this week", "last month"
- **Multiple Output Formats** - CLI, JSON, and plain text output for scripting
- **Local SQLite Storage** - Your data stays on your machine with BadgerDB
- **Shell Completion** - Full completion support for Bash, Zsh, Fish, and PowerShell
- **Export Capabilities** - Export data as JSON or CSV, or create full backups

## Installation

### Quick Install (Recommended)

Install humantime with a single command:

```bash
curl -fsSL https://raw.githubusercontent.com/manav03panchal/humantime/main/install.sh | sh
```

This works on **macOS**, **Linux**, and **Windows** (via WSL/Git Bash). The script automatically:
- Detects your OS and architecture
- Downloads the correct binary
- Installs to `~/.local/bin` (or `/usr/local/bin` if writable)
- Adds the install directory to your PATH

#### Install Options

```bash
# Install a specific version
curl -fsSL https://raw.githubusercontent.com/manav03panchal/humantime/main/install.sh | HUMANTIME_VERSION=v0.3.0 sh

# Install to a custom directory
curl -fsSL https://raw.githubusercontent.com/manav03panchal/humantime/main/install.sh | HUMANTIME_INSTALL_DIR=/opt/bin sh

# Skip PATH modification
curl -fsSL https://raw.githubusercontent.com/manav03panchal/humantime/main/install.sh | HUMANTIME_NO_MODIFY_PATH=1 sh
```

### Using Go Install

If you have Go installed:

```bash
go install github.com/manav03panchal/humantime@latest
```

### Download Binary

Pre-built binaries for all platforms are available on the [Releases](https://github.com/manav03panchal/humantime/releases) page.

| Platform | Architecture | Binary |
|----------|--------------|--------|
| macOS | Intel (x64) | `humantime-darwin-amd64` |
| macOS | Apple Silicon (M1/M2/M3) | `humantime-darwin-arm64` |
| Linux | x64 | `humantime-linux-amd64` |
| Linux | ARM64 | `humantime-linux-arm64` |
| Windows | x64 | `humantime-windows-amd64.exe` |

### Build from Source

```bash
git clone https://github.com/manav03panchal/humantime.git
cd humantime
make build
```

Or without make:

```bash
go build -o humantime .
```

## Quick Start

```bash
# Start tracking time on a project
humantime start on myproject

# Start with a task and note
humantime start on clientwork/bugfix with note "fixing login issue"

# Stop tracking
humantime stop

# View time blocks for this week
humantime blocks this week

# View statistics for a project from last month
humantime stats on clientwork from last month
```

## Command Reference

### Status

```bash
humantime              # Show current tracking status
```

### Start Tracking

```bash
humantime start on <project>                    # Start tracking on a project
humantime start on <project>/<task>             # Start with a task
humantime start on <project> with note 'note'  # Start with a note
humantime start on <project> 2 hours ago        # Start with custom timestamp
humantime start on <project> from 9am to 11am  # Log a completed block
```

Aliases: `sta`, `str`, `s`, `started`, `switch`, `sw`

### Stop Tracking

```bash
humantime stop                                  # Stop current tracking
humantime stop with note 'completed feature'   # Stop with a note
humantime stop 10 minutes ago                   # Stop with custom end time
```

Aliases: `stp`, `e`, `end`, `pause`

### Resume Tracking

```bash
humantime start resume                          # Resume last project/task
```

### View Blocks

```bash
humantime blocks                                # List recent blocks
humantime blocks on <project>                   # Filter by project
humantime blocks on <project>/<task>            # Filter by task
humantime blocks today                          # Blocks from today
humantime blocks this week                      # Blocks from this week
humantime blocks from monday to friday          # Custom time range
humantime blocks <block-id>                     # Show specific block
humantime blocks edit <block-id> --note "new"  # Edit a block
```

Aliases: `block`, `blk`, `b`

### View Statistics

```bash
humantime stats                                 # Stats for today
humantime stats today                           # Stats for today
humantime stats this week                       # Stats for this week
humantime stats on <project>                    # Stats for a project
humantime stats on <project> from last month   # Project stats with timeframe
```

Aliases: `stat`, `stt`

### Manage Projects

```bash
humantime project                               # List all projects
humantime project <project-sid>                 # Show project details
humantime project create "Project Name"         # Create a project
humantime project create "Name" --sid custom   # Create with custom SID
humantime project edit <sid> --name "New Name" # Rename a project
humantime project edit <sid> --color "#FF5733" # Set project color
```

Aliases: `projects`, `proj`, `prj`, `pj`

### Export Data

```bash
humantime export                                # Export all blocks as JSON
humantime export on <project>                   # Export project blocks
humantime export --format csv -o report.csv    # Export as CSV
humantime export --backup -o backup.json       # Full database backup
```

Aliases: `ex`, `x`, `dump`

### Version

```bash
humantime version                               # Show version information
```

## Configuration

### Database Location

By default, Humantime stores data in `~/.local/share/humantime/humantime.db` (following XDG conventions).

Override the database location with the `HUMANTIME_DATABASE` environment variable:

```bash
export HUMANTIME_DATABASE=/path/to/custom/humantime.db
```

### Output Options

```bash
--format, -f    # Output format: cli, json, plain (default: cli)
--color         # Color output: auto, always, never (default: auto)
--debug         # Enable debug output
```

## Shell Completion

Generate shell completion scripts for your shell:

### Bash

```bash
humantime completion bash > /etc/bash_completion.d/humantime
# Or for user installation:
humantime completion bash > ~/.local/share/bash-completion/completions/humantime
```

### Zsh

```bash
humantime completion zsh > "${fpath[1]}/_humantime"
# Then add to your .zshrc if not already present:
# autoload -U compinit && compinit
```

### Fish

```bash
humantime completion fish > ~/.config/fish/completions/humantime.fish
```

### PowerShell

```powershell
humantime completion powershell | Out-String | Invoke-Expression
# To load completions for every session, add the output to your profile
```

## Time Range Syntax

Humantime understands natural language time expressions:

- **Relative**: `2 hours ago`, `10 minutes ago`, `yesterday`
- **Periods**: `today`, `this week`, `last week`, `this month`, `last month`, `this year`
- **Times**: `9am`, `5pm`, `14:30`
- **Ranges**: `from monday to friday`, `from 9am to 5pm`

## License

Licensed under the [SEGV License](LICENSE), Version 1.0.

This software is a derivative work based on [Zeit](https://github.com/mrusme/zeit).
- Original work copyright (c) mrusme
- Modifications copyright (c) Manav Panchal

## Contributing

Contributions are welcome! Please see the [Contributing Guidelines](CONTRIBUTING.md) for more information.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
