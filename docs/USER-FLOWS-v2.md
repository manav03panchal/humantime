# Humantime v2 - User Flows

## Mental Model

```
┌─────────────────────────────────────────────────────────────┐
│                        HUMANTIME                            │
│                                                             │
│   "What am I working on?"  →  ht                           │
│   "Start working"          →  ht start <project>           │
│   "Stop working"           →  ht stop                      │
│   "What did I do?"         →  ht list / ht summary         │
│   "Bill the client"        →  ht export                    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## State Machine

```
                    ┌──────────────┐
                    │    IDLE      │
                    │  (no active  │
                    │   tracking)  │
                    └──────┬───────┘
                           │
                           │ ht start <project>
                           ▼
                    ┌──────────────┐
                    │   TRACKING   │◄─────────────────┐
                    │   (timer     │                  │
                    │   running)   │  ht start <new>  │
                    └──────┬───────┘  (auto-stops     │
                           │          previous)       │
                           │                          │
                           │ ht stop                  │
                           ▼                          │
                    ┌──────────────┐                  │
                    │  BLOCK       │                  │
                    │  CREATED     │──────────────────┘
                    │  (saved)     │
                    └──────────────┘
```

---

## Flow 1: First Time User

```
┌─────────────────────────────────────────────────────────────┐
│ INSTALL                                                     │
└─────────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────┐
│ $ ht                                                        │
│                                                             │
│ Welcome to Humantime!                                       │
│                                                             │
│ Quick start:                                                │
│   ht start <project>    Start tracking time                 │
│   ht stop               Stop tracking                       │
│   ht list               See today's time                    │
│   ht help               Full command list                   │
│                                                             │
│ Try: ht start my-first-project                              │
└─────────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────┐
│ $ ht start my-first-project                                 │
│                                                             │
│ ▶ Started tracking my-first-project at 2:30 PM              │
│                                                             │
│ Project 'my-first-project' created automatically.           │
└─────────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────┐
│ $ ht                                                        │
│                                                             │
│ ▶ my-first-project                                          │
│   Running for 45m (started 2:30 PM)                         │
└─────────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────┐
│ $ ht stop                                                   │
│                                                             │
│ ■ Stopped my-first-project                                  │
│   Duration: 1h 15m                                          │
└─────────────────────────────────────────────────────────────┘
```

---

## Flow 2: Daily Workflow

```
┌────────────────────────────────────────────────────────────────────┐
│                         MORNING                                    │
└────────────────────────────────────────────────────────────────────┘

$ ht                          # Check status
Not tracking anything.
Today: 0h

$ ht start acme-corp          # Start work
▶ Started tracking acme-corp at 9:00 AM

┌────────────────────────────────────────────────────────────────────┐
│                         MID-MORNING                                │
└────────────────────────────────────────────────────────────────────┘

$ ht                          # Quick check
▶ acme-corp
  Running for 2h 30m (started 9:00 AM)

$ ht start internal           # Switch projects (auto-stops previous)
■ Stopped acme-corp: 2h 45m
▶ Started tracking internal at 11:45 AM

┌────────────────────────────────────────────────────────────────────┐
│                           LUNCH                                    │
└────────────────────────────────────────────────────────────────────┘

$ ht stop                     # Stop for lunch
■ Stopped internal: 30m

$ ht                          # Verify stopped
Not tracking anything.
Today: 3h 15m
  acme-corp   2h 45m
  internal      30m

┌────────────────────────────────────────────────────────────────────┐
│                         AFTERNOON                                  │
└────────────────────────────────────────────────────────────────────┘

$ ht start acme-corp "bug fixes"    # Resume with note
▶ Started tracking acme-corp at 1:00 PM

... work ...

$ ht stop "fixed auth bug"          # Stop with note
■ Stopped acme-corp: 3h 30m

┌────────────────────────────────────────────────────────────────────┐
│                        END OF DAY                                  │
└────────────────────────────────────────────────────────────────────┘

$ ht                          # Daily summary
Not tracking anything.

Today: 6h 45m
  acme-corp   6h 15m  ██████████████████░░  93%
  internal      30m   █░░░░░░░░░░░░░░░░░░░   7%

$ ht list                     # Detailed view
Today: 6h 45m

09:00 - 11:45  acme-corp   2h 45m
11:45 - 12:15  internal      30m
13:00 - 16:30  acme-corp   3h 30m  bug fixes → fixed auth bug
```

---

## Flow 3: Retroactive Logging

```
┌────────────────────────────────────────────────────────────────────┐
│ Scenario: Forgot to track a 2-hour meeting yesterday               │
└────────────────────────────────────────────────────────────────────┘

$ ht log acme-corp yesterday 10am-12pm "client meeting"
✓ Logged 2h to acme-corp
  Yesterday, 10:00 AM - 12:00 PM
  Note: client meeting

┌────────────────────────────────────────────────────────────────────┐
│ Scenario: Just finished a call, log it now                         │
└────────────────────────────────────────────────────────────────────┘

$ ht log acme-corp 30m "quick sync call"
✓ Logged 30m to acme-corp
  Today, 2:30 PM - 3:00 PM
  Note: quick sync call

┌────────────────────────────────────────────────────────────────────┐
│ Scenario: Log last week's workshop                                 │
└────────────────────────────────────────────────────────────────────┘

$ ht log training "last tuesday" 9am-5pm "security workshop"
✓ Logged 8h to training
  Tue Jan 9, 9:00 AM - 5:00 PM
  Note: security workshop
```

---

## Flow 4: Weekly Review

```
$ ht summary this week
═══════════════════════════════════════════════════════════════
This Week: 38h 15m
═══════════════════════════════════════════════════════════════

  acme-corp       24h 30m  ████████████████░░░░░░░░  64%
  internal         8h 45m  ██████░░░░░░░░░░░░░░░░░░  23%
  training         5h 00m  ███░░░░░░░░░░░░░░░░░░░░░  13%

───────────────────────────────────────────────────────────────
By Day:
───────────────────────────────────────────────────────────────
  Mon   8h 15m  ████████████████░░░░
  Tue   7h 30m  ███████████████░░░░░
  Wed   8h 00m  ████████████████░░░░
  Thu   7h 45m  ███████████████░░░░░
  Fri   6h 45m  █████████████░░░░░░░


$ ht list this week --project acme-corp
═══════════════════════════════════════════════════════════════
acme-corp - This Week: 24h 30m
═══════════════════════════════════════════════════════════════

Mon Jan 8
  09:00 - 12:30   3h 30m  sprint planning
  13:30 - 18:00   4h 30m  feature development

Tue Jan 9
  09:00 - 12:00   3h 00m  feature development
  13:00 - 17:30   4h 30m  code review + bug fixes

Wed Jan 10
  ...
```

---

## Flow 5: Export for Billing

```
┌────────────────────────────────────────────────────────────────────┐
│ Scenario: Monthly invoice for client                               │
└────────────────────────────────────────────────────────────────────┘

$ ht summary acme-corp last month
═══════════════════════════════════════════════════════════════
acme-corp - December 2024: 86h 30m
═══════════════════════════════════════════════════════════════

Week 1:   22h 15m
Week 2:   24h 00m
Week 3:   18h 45m
Week 4:   21h 30m

$ ht export --project acme-corp --range "last month" --csv > invoice.csv
✓ Exported 89 blocks to stdout

$ cat invoice.csv
date,start,end,duration_hours,duration_decimal,note
2024-12-02,09:00,12:30,3.5,3.50,sprint planning
2024-12-02,13:30,18:00,4.5,4.50,feature development
2024-12-03,09:00,12:00,3.0,3.00,feature development
...

┌────────────────────────────────────────────────────────────────────┐
│ Scenario: JSON export for custom processing                        │
└────────────────────────────────────────────────────────────────────┘

$ ht export --range "this week" | jq '.[] | select(.project == "acme-corp")'
{
  "id": "01HQ3V...",
  "project": "acme-corp",
  "start": "2024-01-08T09:00:00Z",
  "end": "2024-01-08T12:30:00Z",
  "duration_seconds": 12600,
  "note": "sprint planning"
}
...
```

---

## Flow 6: Project Management

```
┌────────────────────────────────────────────────────────────────────┐
│ List projects                                                      │
└────────────────────────────────────────────────────────────────────┘

$ ht projects
Projects (3 active):

  acme-corp     86h 30m total   Last: today
  internal      24h 15m total   Last: yesterday
  training       5h 00m total   Last: 2 weeks ago

┌────────────────────────────────────────────────────────────────────┐
│ Create project explicitly (rare - usually auto-created)            │
└────────────────────────────────────────────────────────────────────┘

$ ht project new "new-client"
✓ Created project: new-client

┌────────────────────────────────────────────────────────────────────┐
│ Archive old project                                                │
└────────────────────────────────────────────────────────────────────┘

$ ht project archive training
✓ Archived project: training
  (86h 30m tracked, data preserved)

$ ht projects
Projects (2 active):

  acme-corp     86h 30m total   Last: today
  internal      24h 15m total   Last: yesterday

$ ht projects --archived
Archived Projects (1):

  training       5h 00m total   Archived: Jan 15
```

---

## Flow 7: Edit & Undo

```
┌────────────────────────────────────────────────────────────────────┐
│ Made a mistake? Undo it.                                           │
└────────────────────────────────────────────────────────────────────┘

$ ht stop
■ Stopped acme-corp: 2h 15m

$ ht undo
✓ Undone: stop
  acme-corp is now tracking again (restored)

▶ acme-corp
  Running for 2h 16m

┌────────────────────────────────────────────────────────────────────┐
│ Edit a specific block                                              │
└────────────────────────────────────────────────────────────────────┘

$ ht list today
Today: 6h 45m

[a1b2c3] 09:00 - 11:45  acme-corp   2h 45m
[d4e5f6] 11:45 - 12:15  internal      30m   # wrong project!
[g7h8i9] 13:00 - 16:30  acme-corp   3h 30m

$ ht edit d4e5f6
Editing block d4e5f6:
  Project: internal
  Start: 11:45 AM
  End: 12:15 PM
  Note: (none)

What to change?
  [p] Project
  [s] Start time
  [e] End time
  [n] Note
  [d] Delete
  [q] Cancel

> p
New project: acme-corp
✓ Updated block d4e5f6

┌────────────────────────────────────────────────────────────────────┐
│ Delete a block                                                     │
└────────────────────────────────────────────────────────────────────┘

$ ht delete a1b2c3
Delete this block?
  acme-corp: 09:00 - 11:45 (2h 45m)

Type 'yes' to confirm: yes
✓ Deleted block a1b2c3
```

---

## Flow 8: Tags (Optional Power Feature)

```
┌────────────────────────────────────────────────────────────────────┐
│ Tag blocks for filtering                                           │
└────────────────────────────────────────────────────────────────────┘

$ ht start acme-corp --tag billable
▶ Started tracking acme-corp at 9:00 AM [billable]

$ ht stop --tag meeting
■ Stopped acme-corp: 1h 30m [billable, meeting]

$ ht list --tag billable
Today (billable): 1h 30m

09:00 - 10:30  acme-corp   1h 30m  [billable, meeting]

$ ht export --tag billable --range "last month" --csv > billable-hours.csv
✓ Exported 45 blocks
```

---

## Error States

```
┌────────────────────────────────────────────────────────────────────┐
│ Already tracking                                                   │
└────────────────────────────────────────────────────────────────────┘

$ ht start acme-corp
▶ Started tracking acme-corp at 9:00 AM

$ ht start internal
■ Stopped acme-corp: 15m
▶ Started tracking internal at 9:15 AM

(Auto-switches, no error)

┌────────────────────────────────────────────────────────────────────┐
│ Nothing to stop                                                    │
└────────────────────────────────────────────────────────────────────┘

$ ht stop
Nothing is being tracked.

Tip: Start tracking with: ht start <project>

┌────────────────────────────────────────────────────────────────────┐
│ Invalid time                                                       │
└────────────────────────────────────────────────────────────────────┘

$ ht log acme-corp 25:00-26:00
Error: Invalid time format '25:00'

Valid formats:
  9am, 9:30am, 14:00, 2pm
  9am-5pm, 9:00-17:00

┌────────────────────────────────────────────────────────────────────┐
│ Block in the future                                                │
└────────────────────────────────────────────────────────────────────┘

$ ht log acme-corp tomorrow 9am-5pm
Error: Cannot log time in the future.

Did you mean: ht start acme-corp
```

---

## Command Cheatsheet

```
╔═══════════════════════════════════════════════════════════════════╗
║                      HUMANTIME CHEATSHEET                         ║
╠═══════════════════════════════════════════════════════════════════╣
║                                                                   ║
║  TRACK                                                            ║
║  ─────                                                            ║
║  ht start <project>          Start tracking                       ║
║  ht stop                     Stop tracking                        ║
║  ht log <project> <time>     Log retroactively                    ║
║                                                                   ║
║  VIEW                                                             ║
║  ────                                                             ║
║  ht                          Current status                       ║
║  ht list [range]             List blocks                          ║
║  ht summary [range]          Summary by project                   ║
║                                                                   ║
║  MANAGE                                                           ║
║  ──────                                                           ║
║  ht projects                 List projects                        ║
║  ht export [--csv]           Export data                          ║
║  ht edit <id>                Edit block                           ║
║  ht undo                     Undo last action                     ║
║                                                                   ║
║  TIME RANGES                                                      ║
║  ───────────                                                      ║
║  today, yesterday, this week, last week                           ║
║  this month, last month, 2024-01-15, jan 15                       ║
║                                                                   ║
╚═══════════════════════════════════════════════════════════════════╝
```
