# Getting Started with Humantime

Humantime is a minimal time tracking CLI. This guide will get you up and running.

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/manav03panchal/humantime/main/install.sh | sh
```

After installation, the `ht` command will be available.

## Basic Workflow

### 1. Start tracking

```bash
ht start myproject
```

This starts a timer on "myproject". If the project doesn't exist, it's created automatically.

### 2. Work on your task

Do your work. The timer runs in the background.

### 3. Stop tracking

```bash
ht stop
```

This saves a time block with your start and end times.

### 4. View your time

```bash
ht blocks today
```

See all time blocks from today.

## Common Commands

| What you want | Command |
|---------------|---------|
| Start tracking | `ht start <project>` |
| Stop tracking | `ht stop` |
| See what's running | `ht` or `ht status` |
| View today's blocks | `ht blocks today` |
| View this week | `ht blocks this week` |
| See time stats | `ht stats today` |
| List projects | `ht projects` |

## Adding Notes

Add context to your time blocks:

```bash
# When starting
ht start myproject --note "implementing auth"

# When stopping
ht stop --note "finished login page"
```

## Backdating Time

Forgot to start the timer? No problem:

```bash
# Start from 2 hours ago
ht start myproject --at "2 hours ago"

# Stop at a specific time
ht stop --at "5pm"
```

## Resume Last Project

Quickly resume what you were working on:

```bash
ht start
```

Running `ht start` without a project resumes tracking on your last project.

## View Statistics

See how you've spent your time:

```bash
# Today's stats
ht stats today

# This week
ht stats this week

# Specific project
ht stats myproject
```

## Next Steps

- [Time Ranges](time-ranges.md) - Learn all the ways to specify time
- [Export & Reports](export.md) - Export your data for reporting
