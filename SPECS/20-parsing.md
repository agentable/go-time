# Parsing

## Overview

Parsing converts standard time formats and controlled natural-language expressions into typed value objects. Determinism wins: standard formats are tried first, then natural language is attempted only when an input locale is provided.

There are two entry paths:

- **Typed path**: `ParseInstant`, `ParseDateTime`, `ParseDate`, `ParseTime`, `ParseDuration`, `ParsePeriod`, `ParseInterval`. These return `(T, error)` and are the default choice when the caller knows the expected type.
- **Inspection path**: `Parse` returns `ParseResult` with status, kind, candidates, warnings, `HasZone`, and error details. Use it when the input type is unknown or ambiguity metadata matters.

`Parse` never returns a Go `error`; semantic outcomes live in `ParseResult.Status`.

## Supported Inputs

| Input | Example | Result kind |
|---|---|---|
| RFC 3339 UTC datetime | `2026-03-27T04:00:00Z` | `KindInstant` |
| ISO datetime with non-UTC offset | `2026-03-27T13:00:00+09:00` | `KindDateTime` |
| ISO local datetime | `2026-03-27T13:00:00` | `KindDateTime` |
| Compact datetime | `20260327T130000`, `20260327T130000+0900` | `KindDateTime` |
| Compact UTC datetime | `20260327T040000Z` | `KindInstant` |
| ISO date | `2026-03-27` | `KindDate` |
| Compact, ordinal, week date | `20260327`, `2026-086`, `2026-W13-5` | `KindDate` |
| Year-month | `2026-03` | `KindDate` |
| Slash date | `04/05/2026` | `KindDate` or ambiguous |
| 24h / 12h time | `15:00`, `08:30:45`, `3:30 PM` | `KindTime` |
| ISO time duration | `PT1H30M`, `PT45S` | `KindDuration` |
| ISO date period | `P1Y3M`, `P7D`, `P2W` | `KindPeriod` |
| ISO interval | `2026-03-27T00:00:00Z/2026-03-28T00:00:00Z`, `2026-03-27T09:00:00Z/PT9H` | `KindInterval` |

`P{date-only}` routes to `Period`. `PT{time-only}` routes to `Duration`. Mixed date/time duration forms such as `P1DT2H` are invalid because no single go-time type can carry both calendar and sub-day exact semantics.

## Natural Language

Natural language is a controlled fallback, not a competing parser.

Current language families:

- Arabic (`ar`)
- English (`en`)
- Hindi (`hi`)
- Japanese (`ja`)
- Korean (`ko`)
- Latin-script European languages in `internal/natural/latin` (`fr`, `de`, `es`, `pt`, `ru`)
- Chinese (`zh-Hans`, `zh-Hant`)

Current coverage is intentionally small: relative dates, week expressions, basic date+time expressions, basic exact-duration expressions, and calendar period expressions for month/year units. Natural-language intervals are outside the current implementation.

Natural-language month and year units route to `KindPeriod`. They are never approximated as 30-day or 365-day `Duration` values.

## ParseResult

```go
type Status string

const (
    StatusResolved  Status = "resolved"
    StatusAmbiguous Status = "ambiguous"
    StatusInvalid   Status = "invalid"
)

type Kind string

const (
    KindInstant  Kind = "instant"
    KindDateTime Kind = "datetime"
    KindDate     Kind = "date"
    KindTime     Kind = "time"
    KindDuration Kind = "duration"
    KindPeriod   Kind = "period"
    KindInterval Kind = "interval"
)

type ParseResult struct {
    Status     Status
    Kind       Kind
    Input      string
    Zone       Zone
    Reference  Instant
    HasZone    bool
    Warnings   []Warning
    Candidates []ParseResult
    Error      *TimeError
}
```

`Candidates` is recursive: each candidate is a resolved `ParseResult`. Access parsed values through either `Value()` or the comma-ok accessors.

```go
switch v := result.Value().(type) {
case gotime.DateTime:
    handle(v)
case gotime.Date:
    handle(v)
case nil:
    handleNonResolved(result)
}
```

`Value()` returns `nil` unless `Status == StatusResolved`. Accessors such as `DateTime() (DateTime, bool)` return `ok=false` when the kind does not match.

## Options

```go
func WithInputLocale(tag language.Tag) Option
func WithZone(zone Zone) Option
func WithReference(t Instant) Option
```

- `WithInputLocale` enables natural-language parsing and disambiguates slash dates.
- `WithZone` supplies the zone for floating datetimes; zero means UTC.
- `WithReference` supplies the base instant for relative expressions.

There is no `WithStrategy`. Ambiguity is surfaced through `Candidates`; callers decide.

When `WithZone` resolves a floating datetime, `ParseResult.Warnings` includes `WarnAssumedZone`. When fractional seconds exceed nanosecond precision, `ParseResult.Warnings` includes `WarnTruncatedPrecision` and the value is truncated to nanoseconds. Slash-date candidates use `WarnInferredCalendar` to explain month-first vs day-first interpretation.

## Ambiguity

Slash dates follow locale when provided. With no locale, they resolve only when one interpretation is valid; otherwise `Parse` returns `StatusAmbiguous`.

DST fall-back local times return `StatusAmbiguous` with two `DateTime` candidates. Each candidate carries `WarnDuplicateTime` with its abbreviation and offset. DST spring-forward gaps return `StatusInvalid` with `CodeNonexistentTime`.

`HasZone` reports whether the original input explicitly included a timezone or offset. It is the caller's hook for detecting floating time.

Interval boundaries must resolve to `KindInstant` or `KindDateTime`. Date-only interval boundaries are invalid because an interval is an absolute UTC range and a bare date has no time or zone.

## Processing Order

1. Trim input.
2. Empty input returns `StatusInvalid` with `ErrEmptyInput`.
3. Try datetime.
4. Try interval, duration, or period.
5. Try date, including slash-date routing.
6. Try time.
7. Try natural language when locale is set.
8. Return `StatusInvalid` when no parser accepts the input.

## Forbidden

- Do not silently choose between genuinely ambiguous interpretations.
- Do not make natural language the primary parser.
- Do not return zero values from accessors when the kind does not match.
- Do not encode warnings as raw strings; use `Warning{Code, Message, Hint}`.
- Do not add strategy options for ambiguity resolution.
