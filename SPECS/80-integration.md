# Integration Boundaries

## Overview

go-time is the primitive time-semantics layer for Agent OS. It answers "what time value is this?" and "how do time values compute?" It does not answer "what product action happens at that time?"

Products may depend on go-time. go-time must not depend on products.

## agenttime CLI

agenttime is a direct consumer of go-time. CLI commands map to primitives:

| CLI task | go-time API |
|---|---|
| now | `Now`, `NowIn` |
| parse | `Parse` or typed `Parse*` |
| format | `.Std()`, `Duration.Decompose()`, `Period` fields, then an external formatter |
| convert zone | `DateTime.In` |
| diff | typed `Sub` methods |
| compare | `Compare` |
| add exact duration | `DateTime.Add` or `Instant.Add` |
| add calendar period | `DateTime.AddPeriod` or `Date.Add` |
| create interval | `NewInterval` or `IntervalOf` |
| interval operations | `Length`, `Contains`, `Overlaps`, `Adjacent`, `Intersect`, `Union`, `Shift`, `Expand` |
| zone show | `LoadZone` plus `Snapshot` |
| zone list | `Zones` |
| zone resolve | `ResolveZone` |
| local zone | `Local` |

The CLI owns command parsing, stdout/stderr, exit codes, JSON vs text output, terminal layout, colors, prompts, defaults, and ambiguity-resolution UX.

go-time owns parsing, typed values, arithmetic, timezone rules, stable JSON, and stdlib bridges.

## Ambiguity Protocol

```go
result := gotime.Parse(input, gotime.WithInputLocale(locale), gotime.WithZone(zone))
switch result.Status {
case gotime.StatusResolved:
    commit(result.Value())
case gotime.StatusAmbiguous:
    // Product policy: fail, choose first, prompt, or apply a documented product rule.
case gotime.StatusInvalid:
    // Show result.Error.Message and result.Error.Hint.
}
```

The product chooses the resolution strategy. go-time only reports ambiguity.

## Product Boundaries

### agentcalendar

Uses go-time for date parsing, timezone projection, interval arithmetic, and stdlib bridges. Owns events, participants, invitations, recurrence, RRULE, availability policy, and display.

### agentreminder

Uses go-time for reminder-time parsing and exact/calendar deltas. Owns notification policy, snooze, deduplication, channels, and rendered text.

### agenttask

Uses go-time for due-date parsing, overdue calculations, and stdlib bridges. Owns task state, lifecycle, dependencies, and display.

### agentmail

Uses go-time for timestamp parsing and projection. Owns mail routing, attachments, threads, and presentation.

### automation / workflow

Uses go-time for time windows and offsets. Owns scheduling, retries, cron expressions, execution state, and human-readable rendering.

## Boundary Table

| go-time owns | Products own |
|---|---|
| Instants, dates, clock times | Events, meetings, schedules |
| Exact durations | Reminders, alarms, notifications |
| Calendar periods | Recurrence, RRULE, cron |
| Half-open intervals | Availability, bookings, execution plans |
| Zones and stdlib bridges | Formatting, locale display, participants, status machines |
| Parsing and arithmetic | Persistence, scheduling, retries |

## Forbidden

- Do not add event, reminder, alarm, cron, RRULE, or scheduler concepts to go-time.
- Do not add business calendars, holidays, or workday rules to go-time.
- Product code should not reimplement time parsing when go-time can parse the input.
- Product code should not add mutable global defaults for zone or language on top of go-time.
