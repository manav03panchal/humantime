# Exporting Data

Export your time tracking data for reports, backups, or integration with other tools.

## Quick Export

Export all blocks as JSON:

```bash
ht export
```

## Export Formats

### JSON (default)

```bash
ht export
ht export --format json
```

Output:
```json
[
  {
    "id": "abc123",
    "project": "myproject",
    "start": "2024-01-15T09:00:00Z",
    "end": "2024-01-15T12:30:00Z",
    "note": "morning work session"
  }
]
```

### CSV

```bash
ht export --format csv
```

Output:
```csv
id,project,start,end,duration,note
abc123,myproject,2024-01-15T09:00:00Z,2024-01-15T12:30:00Z,3h30m,morning work session
```

## Filter by Project

Export only specific project data:

```bash
ht export myproject
ht export myproject --format csv
```

## Filter by Time Range

```bash
ht export --from "this week"
ht export --from monday --to friday
ht export myproject --from "last month"
```

## Save to File

Use `-o` or `--output` to save directly to a file:

```bash
ht export -o timesheet.json
ht export --format csv -o report.csv
ht export myproject -o project-hours.json
```

## Use Cases

### Weekly Timesheet

```bash
ht export --from "this week" --format csv -o weekly-timesheet.csv
```

### Monthly Project Report

```bash
ht export myproject --from "this month" -o monthly-report.json
```

### Full Backup

```bash
ht export -o backup-$(date +%Y%m%d).json
```

## Piping to Other Tools

JSON output works great with `jq`:

```bash
# Total hours this week
ht export --from "this week" | jq '[.[].duration] | add'

# List unique projects
ht export | jq '[.[].project] | unique'

# Filter by note content
ht export | jq '[.[] | select(.note | contains("meeting"))]'
```

CSV output works with standard tools:

```bash
# Open in your spreadsheet app
ht export --format csv -o report.csv && open report.csv

# Quick totals with awk
ht export --format csv | awk -F',' '{sum += $5} END {print sum}'
```
