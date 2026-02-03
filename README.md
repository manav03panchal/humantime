# Humantime

A minimal CLI time tracking tool inspired by [Zeit](https://github.com/mrusme/zeit). Track your work with natural language commands.

## Features

- **Natural Language Interface** - Use intuitive commands like `ht start myproject` and `ht stop`
- **Simple Model** - Just projects and time blocks, nothing more
- **Flexible Time Tracking** - Start/stop tracking, or log completed blocks with custom timestamps
- **Time Range Queries** - Query blocks and stats with natural phrases like "this week", "last month"
- **Multiple Output Formats** - CLI, JSON, and plain text output for scripting
- **Local Storage** - Your data stays on your machine with BadgerDB
- **Shell Completion** - Full completion support for Bash, Zsh, Fish, and PowerShell
- **Export Capabilities** - Export data as JSON or CSV

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
curl -fsSL https://raw.githubusercontent.com/manav03panchal/humantime/main/install.sh | HUMANTIME_VERSION=v1.0.0 sh

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
go build -o ht .
```

## Quick Start

```bash
# Start tracking time on a project
ht start myproject

# Start with a note
ht start myproject --note "working on feature"

# Stop tracking
ht stop

# View time blocks for this week
ht blocks this week

# View statistics
ht stats today
```

## Command Reference

### Status

```bash
ht                     # Show current tracking status
ht status              # Same as above
```

### Start Tracking

```bash
ht start <project>                      # Start tracking on a project
ht start <project> --note "note"        # Start with a note
ht start <project> --at "2 hours ago"   # Start with custom timestamp
ht start                                # Resume last project
```

Aliases: `sta`, `s`

### Stop Tracking

```bash
ht stop                                 # Stop current tracking
ht stop --note "completed feature"      # Stop with a note
ht stop --at "10 minutes ago"           # Stop with custom end time
```

Aliases: `stp`, `e`, `end`

### View Blocks

```bash
ht blocks                               # List recent blocks
ht blocks <project>                     # Filter by project
ht blocks today                         # Blocks from today
ht blocks this week                     # Blocks from this week
ht blocks --from monday --to friday     # Custom time range
```

Aliases: `block`, `blk`, `b`

### Edit Blocks

```bash
ht edit <block-id> --note "new note"    # Edit a block's note
ht edit <block-id> --project other      # Move block to another project
```

### Delete Blocks

```bash
ht delete <block-id>                    # Delete a block
ht delete <block-id> --force            # Delete without confirmation
```

Aliases: `del`, `rm`

### Undo

```bash
ht undo                                 # Undo last action
```

### View Statistics

```bash
ht stats                                # Stats for today
ht stats today                          # Stats for today
ht stats this week                      # Stats for this week
ht stats <project>                      # Stats for a project
```

Aliases: `stat`, `stt`

### Manage Projects

```bash
ht projects                             # List all projects
ht project <project>                    # Show project details
ht project create "Project Name"        # Create a project
ht project delete <project>             # Archive a project
```

Aliases: `proj`, `prj`, `p`

### Export Data

```bash
ht export                               # Export all blocks as JSON
ht export <project>                     # Export project blocks
ht export --format csv -o report.csv    # Export as CSV
```

Aliases: `ex`, `x`

### Version

```bash
ht version                              # Show version information
```

## Configuration

### Database Location

By default, Humantime stores data in `~/.local/share/humantime/` (following XDG conventions).

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
ht completion bash > /etc/bash_completion.d/ht
# Or for user installation:
ht completion bash > ~/.local/share/bash-completion/completions/ht
```

### Zsh

```bash
ht completion zsh > "${fpath[1]}/_ht"
# Then add to your .zshrc if not already present:
# autoload -U compinit && compinit
```

### Fish

```bash
ht completion fish > ~/.config/fish/completions/ht.fish
```

### PowerShell

```powershell
ht completion powershell | Out-String | Invoke-Expression
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

Contributions are welcome!

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
