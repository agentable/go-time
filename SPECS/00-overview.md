# go-time: Time Semantics

## Overview

`go-time` is the Agent OS foundation for time semantics. It turns human time input into precise, typed, computable value objects.

The library does not format values for display. Callers bridge out through stdlib types (`time.Time`, `time.Duration`) or simple data slots (`DurationComponents`, exported `Period` fields), then render with whatever formatter belongs to the product layer.

## Positioning

- **What it is**: parsing, value semantics, conversion, comparison, arithmetic, timezone semantics, and stable JSON.
- **What it is not**: a formatter, locale library, reminder engine, calendar product, scheduler, cron library, recurrence engine, business calendar, or UI policy layer.

> **Why**: Agent OS products need one shared meaning for dates, instants, intervals, durations, periods, and zones. Fragmented time handling creates inconsistent parsing, timezone, and arithmetic behavior.
>
> **Rejected**: putting time semantics separately into every product, or expanding this package into a full calendar/scheduling suite.

## Design Philosophy

The API follows constraint-led design: remove every decision the caller should not have to make, and make the remaining decisions obvious.

1. **One obvious way**: one concept, one type, one primary verb.
2. **Names read as English**: `gotime.Now()`, `dt.In(zone)`, `iv.Contains(t)`.
3. **No surprise**: fallible APIs return `error`; the only public panic API is `MustLoadZone`.
4. **Primitives, not products**: no display policy, ambiguity UX, scheduling, persistence, or CLI protocol.
5. **Immutable values**: all operations return new values.
6. **One concept per type**: `Duration` is exact nanoseconds; `Period` is calendar Y/M/D; `Interval` is zone-free.
7. **Stable wire format**: JSON is deterministic and never depends on marshal-time state.
8. **Errors compose with stdlib**: `Err*` sentinels for `errors.Is`, `*TimeError` for `errors.As`.
9. **Minimal dependency surface**: public API uses stdlib plus `golang.org/x/text/language.Tag`; no formatter, CLDR, or locale-data dependency.

## Architecture

```text
Product layer      applications, agents, CLIs, services
Display layer      stdlib time.Format, external formatters, logs, templates
                  ↑ time.Time / time.Duration / data slots
go-time            typed time semantics
                  ↓
Go stdlib          time, errors, slices, etc.
```

Internal dependency direction is strict:

```text
Layer 3: arithmetic and comparison
Layer 2: parsing and natural-language helpers
Layer 1: value objects and stable JSON
```

Layer 1 is stdlib-only except `zone.go`, which reads static IANA data from `internal/zone`. There is no formatting layer.

## Module Path

```text
github.com/agentable/go-time
```

## Version Contract

These specs describe the current v2 contract. If implementation and specs disagree, update the stale side in the same work cycle. Specs must describe behavior that code can satisfy or violate; tutorials belong in `README.md`.

## Permanent Non-Goals

- No formatter or locale types (`DateTimeFormat`, `RelativeTimeFormat`, `DurationFormat`, `Locale`, `HourCycle`, `Calendar`, `Style`).
- No i18n, CLDR, message-format, locale-data, JSON/YAML/template locale assets, or formatter dependency.
- No event, reminder, alarm, scheduler, cron, RRULE, recurrence, business-calendar, holiday, lunar-calendar, Julian-day, sunrise/sunset, or moon-phase model.
- No semantic mixing: `Instant + Period`, `Date + Duration`, and `DateTime + DateTime` remain undefined.
- No marshal-time calls to `time.Now()`.
- No public `Must*` except `MustLoadZone`.
- No exposed monotonic clock concept.
- No zone-free `DateTime`; unresolved wall-clock values use `LocalDateTime`, while persisted `DateTime` values carry a `Zone`.
