# Parsing

## Overview

Parsing converts standard time formats and controlled natural-language expressions into typed value objects. Determinism wins: standard formats are tried first, then natural language is attempted only when an input locale is provided.

There are two entry paths:

- **Typed path**: `ParseInstant`, `ParseDateTime`, `ParseLocalDateTime`, `ParseDate`, `ParseTime`, `ParseDuration`, `ParsePeriod`, `ParseInterval`. These return `(T, error)` and are the default choice when the caller knows the expected type.
- **Diagnostic path**: `Parse` returns `ParseResult` with status, kind, candidates, warnings, `HasZone`, and error details. Use it when the input type is unknown or ambiguity metadata matters.

`Parse` never returns a Go `error`; semantic outcomes live in `ParseResult.Status`.

## Supported Inputs

| Input | Example | Result kind |
|---|---|---|
| RFC 3339 datetime with offset | `2026-03-27T13:00:00+09:00`, `2026-03-27T04:00:00Z` | `KindInstant` |
| ISO local datetime | `2026-03-27T13:00:00` | `KindLocalDateTime`, or `KindDateTime` with `WithZone` |
| Compact datetime | `20260327T130000`, `20260327T130000+0900`, `20260327T040000Z` | `KindLocalDateTime` without offset, `KindInstant` with offset |
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

The grammar is intentionally small: relative dates, week expressions, basic date+time expressions, basic exact-duration expressions, and calendar period expressions for day/week/month/year units. Natural-language intervals are outside this contract.

Natural-language day, week, month, and year units route to `KindPeriod`. They are never approximated as 24-hour, 7-day, 30-day, or 365-day `Duration` values. Natural-language second, minute, and hour units route to `KindDuration`.

## Contract Decisions

### Explicit Human Context

- **Decision**: Natural-language parsing requires `WithInputLocale`. Natural date/datetime expressions that need a calendar reference also require `WithReference`.
- **Why**: Locale and reference time are human interpretation context. A semantics kernel must not read ambient process time or silently infer language policy.
- **Rejected**: Defaulting relative phrases to `time.Now()`, global parser defaults, `WithNow`, `WithClock`, and strategy knobs that hide ambiguity.
- **Contract Impact**: Product code chooses "now" explicitly at the boundary with `WithReference(gotime.Now())`; deterministic code passes a fixed `Instant`.

### Formal Interval Grammar

- **Decision**: Interval parsing is contained to formal instant/datetime/duration subparsers. It does not re-enter public `Parse` for interval parts.
- **Why**: `Parse` is an inspection dispatcher whose accepted grammar can grow. Interval grammar must not widen accidentally when natural language or other dispatch paths change.
- **Rejected**: Natural-language interval endpoints, date-only interval endpoints, and recursive public-dispatch parsing of interval sides.
- **Contract Impact**: Intervals accept only explicit absolute endpoints or one explicit endpoint plus exact duration.

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
    KindInstant       Kind = "instant"
    KindDateTime      Kind = "datetime"
    KindLocalDateTime Kind = "local_datetime"
    KindDate          Kind = "date"
    KindTime          Kind = "time"
    KindDuration      Kind = "duration"
    KindPeriod        Kind = "period"
    KindInterval      Kind = "interval"
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

`Candidates` is recursive: each candidate is a resolved `ParseResult`. Access parsed values through comma-ok accessors.

```go
switch result.Status {
case gotime.StatusResolved:
    switch result.Kind {
    case gotime.KindDateTime:
        if dt, ok := result.DateTime(); ok {
            handle(dt)
        }
    case gotime.KindLocalDateTime:
        if ldt, ok := result.LocalDateTime(); ok {
            handle(ldt)
        }
    case gotime.KindDate:
        if d, ok := result.Date(); ok {
            handle(d)
        }
    }
case gotime.StatusAmbiguous, gotime.StatusInvalid:
    handleNonResolved(result)
}
```

Accessors such as `DateTime() (DateTime, bool)` and `LocalDateTime() (LocalDateTime, bool)` return `ok=false` unless `Status == StatusResolved` and the kind matches.

## Options

```go
func WithInputLocale(tag language.Tag) Option
func WithZone(zone Zone) Option
func WithReference(t Instant) Option
```

- `WithInputLocale` enables natural-language parsing and disambiguates slash dates.
- `WithZone` supplies the zone for floating datetimes. Without it, local datetimes remain `KindLocalDateTime` instead of defaulting to UTC.
- `WithReference` supplies the base instant for natural date/datetime expressions.

There is no `WithStrategy`. Ambiguity is surfaced through `Candidates`; callers decide.

Natural date/datetime expressions that need a calendar reference, such as `tomorrow` or `next Friday`, return `StatusInvalid` with `ErrInvalidFormat` unless `WithReference` is provided. Exact natural durations and periods that resolve directly to `Duration` or `Period` do not require a reference.

When `WithZone` resolves a floating datetime, `ParseResult.Warnings` includes `WarnAssumedZone`. Without `WithZone`, the same input resolves to `KindLocalDateTime` and carries no zone assumption. When fractional seconds exceed nanosecond precision, `ParseResult.Warnings` includes `WarnTruncatedPrecision` and the value is truncated to nanoseconds. Slash-date candidates use `WarnInferredCalendar` to explain month-first vs day-first interpretation.

## Ambiguity

Slash dates follow locale when provided. With no locale, they resolve only when one interpretation is valid; otherwise `Parse` returns `StatusAmbiguous`.

Floating datetime parsing uses the same projection rule as `LocalDateTime.Resolve` only when `WithZone` is supplied. DST fall-back local times return `StatusAmbiguous` with `DateTime` candidates. Each candidate carries `WarnDuplicateTime` with its abbreviation and offset. DST spring-forward gaps return `StatusInvalid` with `CodeNonexistentTime`.

Typed parsers preserve the ambiguity cause when translating `StatusAmbiguous` into an error. Slash-date ambiguity returns a `*TimeError` wrapping `ErrAmbiguousDate`. DST fall-back ambiguity returns a `*TimeError` wrapping `ErrDuplicateTime`, including when the duplicate local time appears inside an interval endpoint.

`HasZone` reports whether the original input explicitly included a timezone or offset. It is the caller's hook for detecting floating time.

Interval boundaries must resolve to `KindInstant` or `KindDateTime`. Date-only interval boundaries are invalid because an interval is an absolute UTC range and a bare date has no time or zone.
Natural-language interval boundaries are invalid even when `WithInputLocale` and `WithReference` are supplied.

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
- Do not read `time.Now()` while interpreting input. Callers pass reference time explicitly.
- Do not let interval parsing re-enter public `Parse` for interval subparts.
- Do not return zero values from accessors when the kind does not match.
- Do not encode warnings as raw strings; use `Warning{Code, Message, Hint}`.
- Do not add strategy options for ambiguity resolution.

## Acceptance Criteria

- Relative natural date/datetime input without `WithReference` returns `StatusInvalid` with an actionable error.
- Exact natural durations and periods still resolve without a reference when locale is supplied.
- Slash-date and DST ambiguity surface through `StatusAmbiguous` and candidates, not a strategy option.
- Interval tests prove date-only and natural-language boundaries are rejected unless a future spec deliberately changes the interval grammar.
