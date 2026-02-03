# Time Ranges in Humantime

Humantime understands natural language for specifying times and date ranges.

## Relative Times

Use these when starting or stopping with `--at`:

| Expression | Meaning |
|------------|---------|
| `now` | Current time |
| `5 minutes ago` | 5 minutes before now |
| `2 hours ago` | 2 hours before now |
| `yesterday` | Yesterday at current time |
| `yesterday at 5pm` | Yesterday at 5pm |

### Examples

```bash
ht start myproject --at "30 minutes ago"
ht stop --at "5 minutes ago"
```

## Time of Day

Specify times in various formats:

| Format | Example |
|--------|---------|
| 12-hour | `9am`, `5pm`, `10:30am` |
| 24-hour | `14:30`, `09:00` |

### Examples

```bash
ht start myproject --at "9am"
ht stop --at "5:30pm"
```

## Named Periods

Use these with `ht blocks` and `ht stats`:

| Period | Meaning |
|--------|---------|
| `today` | From midnight today |
| `yesterday` | All of yesterday |
| `this week` | Monday to now (or Sunday, based on locale) |
| `last week` | The previous full week |
| `this month` | From the 1st of current month |
| `last month` | The previous full month |
| `this year` | From January 1st |

### Examples

```bash
ht blocks today
ht blocks this week
ht stats last month
```

## Day Names

Reference specific days:

```bash
ht blocks monday
ht blocks --from monday --to friday
```

## Custom Ranges

Use `--from` and `--to` for custom date ranges:

```bash
# From Monday to Friday
ht blocks --from monday --to friday

# Last 3 days
ht blocks --from "3 days ago" --to today

# Specific dates
ht blocks --from "jan 1" --to "jan 31"
```

## Tips

1. **Quotes**: Use quotes for multi-word expressions: `--at "2 hours ago"`
2. **Case**: Time expressions are case-insensitive: `9AM` = `9am` = `9Am`
3. **Flexibility**: Most reasonable expressions work - try it!
