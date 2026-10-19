# Integration Boundaries

## Overview

go-time is the primitive time-semantics layer. It answers "what time value is this?" and "how do time values compute?" It does not answer "what product action happens at that time?"

Applications, agents, CLIs, and services may depend on go-time. go-time must not depend on them.

## Consumer Mapping

Consumers map workflow needs to stable primitives:

| Consumer task | go-time API |
|---|---|
| current instant or date | `Now`, `NowIn`, `TodayIn` |
| parse known input type | typed `Parse*` functions |
| parse unknown input type | `Parse` and `ParseResult` |
| represent unresolved wall-clock input | `LocalDateTime` |
| resolve wall-clock input in a zone | `LocalDateTime.Resolve`, `ParseDateTime` with `WithZone` |
| render output | `.Std()`, `Duration.Decompose()`, `Period` fields, then an external renderer |
| convert zone | `DateTime.In` |
| exact difference | checked `Instant.Sub`, `DateTime.Sub`, `Interval.Length` |
| calendar difference | `Date.DaysUntil` |
| compare values | `Compare` methods |
| add exact duration | `Instant.Add`, `DateTime.Add` |
| add calendar period | `DateTime.AddPeriod`, `Date.Add` |
| create interval from endpoints | `NewInterval` |
| create interval from one endpoint and length | `NewIntervalStartingAt`, `NewIntervalEndingAt` |
| interval operations | `Length`, `Contains`, `Overlaps`, `Adjacent`, `Intersect`, `Union`, `Shift`, `Expand` |
| strict zone identity | `LoadZone` |
| fuzzy zone input | `ResolveZone` |
| zone catalog | `Zones`, `ZoneCatalogVersion` |

Consumers own input channels, command routing, persistence, logging, stdout/stderr, exit codes, JSON vs text presentation, terminal layout, colors, prompts, defaults, authorization, retries, scheduling policy, and ambiguity-resolution UX.
Host-local timezone discovery is also a consumer responsibility; consumers
must pass an explicit IANA `Zone` into go-time.

go-time owns parsing, typed value objects, arithmetic, timezone rules, stable JSON, structured errors, warnings, and stdlib bridges.

## Field Values

Field owns its raw JSON grammar, formats, presence, defaults, and list semantics.
Validate there before constructing native values for computation:

| Field meaning | Native value when computation is needed |
|---|---|
| Calendar date | `Date`, without an invented midnight or zone |
| Clock time | `Time`, without a duration interpretation |
| Absolute date-time | `Instant`; consumers may subsequently select a Zone |
| Nonnegative number of seconds | `Duration` only if exact integer nanoseconds fit int64 |

The human parser, this library's closed `kind` JSON objects, and Field strings
or numeric seconds have different accepted domains. For example, `ParseTime`
accepts `09:30`, while Field requires seconds; Field accepts `12:30:00.100`,
while this library's Time JSON requires canonical `12:30:00.1`. Use
`encoding/json/v2` and `encoding/json/jsontext` at JSON boundaries.

go-field may use the standard library for its protocol validator. It need not
import go-time merely to store strings. Consumers do not duplicate Field
validation or reinterpret its format by calling diagnostic `Parse`.

A valid Field duration can have sub-nanosecond precision or exceed the native
Duration range. Check exact seconds-to-nanoseconds conversion only when handing
it to a timer; failed conversion does not invalidate the original Field value.
Do not first convert seconds to float64 and multiply by 1e9. Exporting a native
duration also requires checking the target number domain; `InSeconds()` may
approximate it. A negative Duration cannot be written to a nonnegative Field.

An Instant in year 2500 does not require a UnixNano projection. UTC wire-year
bounds and epoch scalar bounds are independent. Midnight, zero duration, and
the Go zero Instant can be provided values; `IsZero` is not a presence check.
`multiple` means independent values, not an Interval. Period has no implicit
Field mapping and is never approximated as a fixed number of seconds.

## Ambiguity Protocol

```go
result := gotime.Parse(input, gotime.WithInputLocale(locale), gotime.WithZone(zone))
switch result.Status {
case gotime.StatusResolved:
    switch result.Kind {
    case gotime.KindDateTime:
        if dt, ok := result.DateTime(); ok {
            commitDateTime(dt)
        }
    case gotime.KindDate:
        if d, ok := result.Date(); ok {
            commitDate(d)
        }
    }
case gotime.StatusAmbiguous:
    // Consumer policy: fail, prompt, or apply a documented rule.
case gotime.StatusInvalid:
    // Show result.Error.Message and result.Error.Hint.
}
```

The consumer chooses the resolution strategy. go-time only reports ambiguity through `ParseResult.Candidates`, warnings, and typed errors.

There is no `WithStrategy` option. Ambiguity policy belongs above this module because it depends on user intent, interaction model, and product risk.

## Domain Boundaries

Scheduling systems may use go-time for date parsing, timezone projection, interval arithmetic, and stdlib bridges. They own events, participants, invitations, recurrence, availability policy, and display.

Notification systems may use go-time for target-time parsing and exact/calendar deltas. They own delivery policy, snooze, deduplication, channels, and rendered text.

Task and workflow systems may use go-time for due-date parsing, overdue calculations, time windows, and offsets. They own lifecycle, dependencies, retries, execution state, and status machines.

Messaging, logging, and audit systems may use go-time for timestamp parsing and projection. They own routing, storage, redaction, and presentation.

## Boundary Table

| go-time owns | Consumers own |
|---|---|
| Instants, dates, local datetimes, zoned datetimes, clock times | Events, meetings, schedules |
| Exact durations | Reminders, alarms, notifications |
| Calendar periods | Recurrence, RRULE, cron |
| Half-open intervals | Availability, bookings, execution plans |
| Zones and stdlib bridges | Formatting, locale display, participants, status machines |
| Parsing and arithmetic | Persistence, scheduling, retries |
| Structured errors and warnings | Error presentation, recovery policy, telemetry |

## Forbidden

- Do not add event, reminder, alarm, cron, RRULE, scheduler, workflow, or messaging concepts to go-time.
- Do not add business calendars, holidays, workday rules, or availability policy to go-time.
- Do not add formatter or locale-display policy to go-time.
- Consumers reuse go-time for its accepted input domain; separate protocol
  validators retain their stricter lexical and semantic contracts.
- Consumer code should not add mutable global defaults for zone or language on top of go-time.
