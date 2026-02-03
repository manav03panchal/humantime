# Humantime v2 - Product Requirements Document

## Vision

**Humantime is a CLI time tracker that gets out of your way.**

Track time with natural language. See where your time goes. Export for billing. That's it.

---

## Core Principles

1. **One way to do things** - No redundant commands
2. **Zero configuration required** - Works out of the box
3. **No background processes** - CLI tool, not a service
4. **Data portability** - Your data, exportable anytime
5. **Natural language first** - Speak like a human, not a machine

---

## What Humantime Is NOT

- A todo app (use Things, Todoist, etc.)
- A notification system (use your calendar)
- A dashboard (use the terminal)
- A goal tracker (use a spreadsheet)
- A pomodoro timer (use a dedicated app)

---

## User Personas

### Primary: Freelancer/Contractor
- Needs to track billable hours
- Works on multiple client projects
- Exports time for invoicing
- Lives in the terminal

### Secondary: Developer
- Wants to know where time goes
- Tracks personal projects
- Uses for self-awareness, not billing
- Keyboard-first workflow

---

## Core Entities (Simplified)

```
Project
├── name: string (human readable)
├── sid: string (short identifier, auto-generated)
├── color: string (for future TUI, optional)
├── archived: bool
└── created_at: timestamp

Block (a tracked time period)
├── id: uuid
├── project: reference to Project
├── start: timestamp
├── end: timestamp (null = active)
├── note: string (optional)
└── tags: []string (optional, freeform)
```

**That's it. Two entities.**

No tasks, no goals, no reminders, no webhooks, no notifications, no daemon.

---

## Commands

### Track Time

```bash
# Start tracking
ht start <project>                    # Start on project
ht start <project> "working on X"     # Start with note
ht start <project> --tag billable     # Start with tag

# Stop tracking
ht stop                               # Stop current
ht stop "finished feature"            # Stop with note

# Quick track (retroactive)
ht log <project> 2h                   # Log 2 hours ending now
ht log <project> 2h "meeting"         # Log with note
ht log <project> 9am-11am             # Log specific range
ht log <project> yesterday 2pm-4pm    # Log past time
```

### View Time

```bash
# Current status (default command)
ht                                    # What am I tracking now?

# View blocks
ht list                               # Today's blocks
ht list yesterday                     # Yesterday
ht list this week                     # This week
ht list last month                    # Last month
ht list <project>                     # Filter by project
ht list --tag billable                # Filter by tag

# Summarize
ht summary                            # Today's summary by project
ht summary this week                  # Week summary
ht summary <project> this month       # Project summary
```

### Manage Projects

```bash
# List projects
ht projects                           # All active projects

# Create (usually implicit via start)
ht project new <name>                 # Explicit create

# Archive (soft delete)
ht project archive <project>          # Hide from lists
ht projects --archived                # Show archived
```

### Data Operations

```bash
# Export
ht export                             # JSON to stdout
ht export --csv                       # CSV format
ht export --range "last month"        # Date range
ht export > timesheet.json            # Save to file

# Import
ht import timesheet.json              # Import from file

# Edit/Delete (rare operations)
ht edit <block-id>                    # Interactive edit
ht delete <block-id>                  # Delete block (with confirmation)
ht undo                               # Undo last destructive action
```

### Meta

```bash
ht version                            # Version info
ht help                               # Help
ht completion bash                    # Shell completions
```

---

## Command Count

| Category | Commands | Notes |
|----------|----------|-------|
| Track | 3 | start, stop, log |
| View | 3 | (default), list, summary |
| Projects | 2 | projects, project |
| Data | 4 | export, import, edit, delete, undo |
| Meta | 3 | version, help, completion |
| **Total** | **15** | Down from 46+ |

---

## User Flows

### Flow 1: Daily Tracking (90% of usage)

```
Morning:
$ ht start client-a
▶ Started tracking client-a at 9:00 AM

Work happens...

$ ht
▶ client-a: 2h 34m (started 9:00 AM)

Lunch:
$ ht stop
■ Stopped client-a: 2h 45m

$ ht start personal/learning
▶ Started tracking personal/learning at 11:46 AM

End of day:
$ ht stop
■ Stopped personal/learning: 45m

$ ht
Today: 3h 30m
  client-a         2h 45m  ████████████░░░░  78%
  personal/learning  45m   ████░░░░░░░░░░░░  22%
```

### Flow 2: Retroactive Logging

```
Forgot to track a meeting:
$ ht log client-a 10am-11:30am "sprint planning"
✓ Logged 1h 30m to client-a

Log yesterday's work:
$ ht log client-b yesterday 2h "code review"
✓ Logged 2h to client-b (yesterday)
```

### Flow 3: Weekly Review

```
$ ht summary this week
This Week: 32h 15m

  client-a          18h 30m  ████████████████░░░░  57%
  client-b           8h 45m  ████████░░░░░░░░░░░░  27%
  personal/learning  5h 00m  █████░░░░░░░░░░░░░░░  16%

$ ht list this week --tag billable
Mon  client-a   9:00-12:00   3h 00m  sprint planning
Mon  client-a  13:00-17:30   4h 30m  feature dev
Tue  client-b   9:00-11:00   2h 00m  code review
...
```

### Flow 4: Export for Invoice

```
$ ht export --csv --range "last month" --tag billable > invoice-oct.csv
✓ Exported 45 blocks to invoice-oct.csv

$ cat invoice-oct.csv
date,project,duration_hours,note,tags
2024-10-01,client-a,3.5,"sprint planning",billable
2024-10-01,client-a,4.5,"feature dev",billable
...
```

---

## Natural Language Time Parsing

### Supported Formats

**Relative:**
- `today`, `yesterday`, `tomorrow`
- `this week`, `last week`, `next week`
- `this month`, `last month`
- `2 days ago`, `3 weeks ago`

**Absolute:**
- `monday`, `tuesday`, ... (most recent)
- `jan 15`, `january 15`, `2024-01-15`
- `9am`, `9:30am`, `14:00`, `2pm`

**Durations:**
- `30m`, `30min`, `30 minutes`
- `2h`, `2hr`, `2 hours`
- `1h30m`, `1.5h`, `90m`

**Ranges:**
- `9am-5pm`
- `9:00-17:00`
- `monday to friday`
- `jan 1 - jan 15`

---

## Output Formats

### Default (Human-readable)

```
$ ht list
Today: 5h 30m

09:00 - 12:15  client-a      3h 15m  sprint + feature work
13:00 - 15:15  client-a      2h 15m  bug fixes
```

### JSON (for scripting)

```bash
$ ht list --json
[
  {"id": "abc123", "project": "client-a", "start": "2024-01-15T09:00:00Z", ...}
]
```

### CSV (for spreadsheets)

```bash
$ ht list --csv
date,project,start,end,duration,note,tags
2024-01-15,client-a,09:00,12:15,3.25,sprint + feature work,billable
```

---

## Data Storage

- **Location:** `~/.local/share/humantime/` (XDG compliant)
- **Format:** BadgerDB (embedded, no server)
- **Backup:** `ht export > backup.json`
- **Restore:** `ht import backup.json`

---

## What We're Removing

| Feature | Reason |
|---------|--------|
| Tasks | Unnecessary hierarchy. Projects are enough. |
| Goals | Just numbers without actionable guidance. Use a spreadsheet. |
| Reminders | Not time tracking. Use a todo app. |
| Daemon | Background process overkill. No notifications needed. |
| Webhooks | Feature creep. Pipe to curl if needed. |
| Dashboard TUI | Redundant with `ht` and `ht summary`. |
| Pomodoro | Niche. Use a dedicated timer. |
| NotifyConfig | Gone with daemon. |
| Multiple view commands | Consolidated to `list` and `summary`. |

---

## Migration Path

For existing users:

```bash
$ ht migrate
Migrating from v1...
  ✓ Blocks: 1,234 migrated
  ✓ Projects: 12 migrated
  ⚠ Tasks: 5 converted to project notes
  ⚠ Goals: Exported to ~/humantime-goals-backup.json
  ⚠ Reminders: Exported to ~/humantime-reminders-backup.json
  ✓ Migration complete

Your goals and reminders have been exported.
Consider using a dedicated app for these features.
```

---

## Success Metrics

1. **Time to first track:** < 5 seconds (just `ht start project`)
2. **Commands memorized:** 5 or fewer for daily use
3. **Zero configuration:** Works immediately after install
4. **Binary size:** < 15MB
5. **Startup time:** < 100ms

---

## CLI Name

Consider renaming from `humantime` to `ht` as the primary command:
- Faster to type
- Muscle memory friendly
- `humantime` can be an alias

---

## Implementation Phases

### Phase 1: Core (MVP)
- [ ] `ht start/stop`
- [ ] `ht` (status)
- [ ] `ht list`
- [ ] `ht summary`
- [ ] `ht projects`
- [ ] `ht export/import`
- [ ] Migration from v1

### Phase 2: Polish
- [ ] `ht log` (retroactive)
- [ ] `ht edit/delete`
- [ ] `ht undo`
- [ ] Shell completions
- [ ] Tags support

### Phase 3: Nice-to-have
- [ ] `ht project archive`
- [ ] Color output customization
- [ ] Git integration (auto-detect project from repo)

---

## Non-Goals (Explicitly Out of Scope)

- Mobile app
- Web interface
- Team features / sharing
- Integrations (Jira, GitHub, etc.)
- Notifications of any kind
- Background processes
- Calendars or scheduling
- Invoicing (export and use a real tool)

---

## Appendix: Command Reference

```
ht                              Show current tracking status
ht start <project> [note]       Start tracking time
ht stop [note]                  Stop tracking time
ht log <project> <duration>     Log time retroactively
ht list [range] [filters]       List time blocks
ht summary [range] [project]    Summarize time by project
ht projects                     List all projects
ht project new <name>           Create a project
ht project archive <name>       Archive a project
ht export [--csv] [--range]     Export data
ht import <file>                Import data
ht edit <block-id>              Edit a block
ht delete <block-id>            Delete a block
ht undo                         Undo last action
ht version                      Show version
ht help                         Show help
ht completion <shell>           Generate completions
```

**15 commands. That's the whole product.**
