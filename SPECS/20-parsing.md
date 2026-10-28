# Parsing

## Overview

Parsing converts standard time formats and controlled natural-language expressions into typed value objects. Determinism wins: standard formats are tried first, then natural language is attempted only when an input locale is provided.

There are two entry paths:

- **Typed path**: `ParseInstant`, `ParseDateTime`, `ParseLocalDateTime`, `ParseDate`, `ParseTime`, `ParseDuration`, `ParsePeriod`, `ParseInterval`. These return `(T, error)` and are the default choice when the caller knows the expected type.
- **Diagnostic path**: `Parse` returns `ParseResult` with status, kind, candidates, warnings, `HasZone`, and error details. Use it when the input type is unknown or ambiguity metadata matters.

`Parse` never returns a Go `error`; semantic outcomes live in `ParseResult.Status()`.

## Supported Inputs

| Input | Example | Result kind |
|---|---|---|
| RFC 3339 datetime with offset | `2026-03-27T13:00:00+09:00`, `2026-03-27T04:00:00Z` | `KindInstant` |
| ISO local datetime | `2026-03-27T13:00:00` | `KindLocalDateTime`, or `KindDateTime` with `WithZone` |
| Compact datetime | `20260327T130000`, `20260327T130000+0900`, `20260327T040000Z` | `KindLocalDateTime` without offset, `KindInstant` with offset |
| ISO date | `2026-03-27` | `KindDate` |
| Compact, ordinal, week date | `20260327`, `2026-086`, `2026-W13-5` | `KindDate` |
| Incomplete year-month or ISO week | `2026-03`, `2026-W13` | `StatusInvalid`, with a hint requesting the missing day |
| Slash date | `04/05/2026` | `KindDate` or ambiguous |
| 24h / 12h time | `15:00`, `08:30:45`, `3:30 PM` | `KindTime` |
| ISO time duration | `PT1H30M`, `PT45S` | `KindDuration` |
| Calendar period text | `P1Y3M`, `P7D`, `P2W` | `KindPeriod` |
| ISO interval | `2026-03-27T00:00:00Z/2026-03-28T00:00:00Z`, `2026-03-27T09:00:00Z/PT9H` | `KindInterval` |

Formal local, compact, and offset datetimes accept civil years `0000..9999`;
year `0000` is a value, never a missing-component marker.
Minute-precision extended and compact datetimes default seconds to zero.
Fractions follow explicit seconds only; fractional hours or minutes are not
supported. Explicit offsets use `Z`, `±HH:MM`, or `±HHMM`; `±HH` is rejected.

`P{date-only}` routes to `Period`. `PT{time-only}` routes to `Duration`. Mixed date/time duration forms such as `P1DT2H` are invalid because no single go-time type can carry both calendar and sub-day exact semantics.
A `T` separator requires at least one clock component. Incomplete forms such
as `P1DT` return `ErrInvalidFormat`; `PT0S` remains a valid zero duration.
Date-only Period components may carry individual signs so every valid Period
has a lossless public text round trip, for example `P+1Y-2M+3D`.
A leading period sign applies to unsigned component magnitudes only. Inputs
such as `-P-1Y`, `-P+1Y`, and `-P-1W+2D` are invalid because two sign layers
would make the represented value unclear.

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

Chinese relative-unit counts use canonical decimal compositions through
thousands. `零` or `〇` is required when a composition skips a place. `两` and
`兩` may represent two alone or precede `百` or `千`; a trailing two in a
composite uses `二`, so `一百零二` is accepted and `一百零两` is rejected.
Noncanonical shorthand and repeated or ascending units do not match.

The internal natural parser receives a reference whose civil fields were
already projected into the caller's `WithZone`. It performs calendar math only
and returns unresolved civil components; it does not load or resolve zones.
The gotime parse boundary maps natural datetime components through the same
`LocalDateTime.Resolve` path as formal local datetime input.

## Contract Decisions

### Explicit Human Context

- **Decision**: Natural-language parsing requires `WithInputLocale`. Natural date/datetime expressions that need a calendar reference require both `WithReference` and `WithZone`.
- **Why**: Locale, reference time, and its calendar zone are human interpretation context. An `Instant` has no calendar date until projected into a zone. A semantics kernel must not read ambient process time or silently infer language/zone policy.
- **Rejected**: Defaulting relative phrases to `time.Now()`, global parser defaults, `WithNow`, `WithClock`, and strategy knobs that hide ambiguity.
- **Contract Impact**: Product code chooses "now" and its calendar frame explicitly with `WithReference(gotime.Now())` plus `WithZone(zone)`; deterministic code passes a fixed `Instant` and Zone.

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
    // Identity and typed value storage are private.
    Input      string
    Zone       Zone
    Reference  Instant
    HasZone    bool
    Warnings   []Warning
    Candidates []ParseResult
    Error      *TimeError
}
```

`Status() Status` and `Kind() Kind` read the package-owned identity. Callers
cannot retag a stored value or construct a successful result by setting fields.
A zero result has empty status/kind and all typed accessors return false.
Other diagnostic fields remain caller-owned: changing Input, warnings, errors,
or candidates does not change the stored value or its type. Such edits can make
the diagnostic differ from the original parse; this is not an immutable result
or an editable value builder. To obtain another interpretation, parse again or
select an already-resolved candidate.

`Candidates` is recursive: each candidate is a resolved `ParseResult`. Access parsed values through comma-ok accessors.

`Warnings` and `Candidates` are caller-owned slices. Copying a `ParseResult`
copies the slice headers and continues to share their backing arrays; it is not
a deep clone. Clone the specific slices, including nested candidate slices,
before independent mutation or owner handoff. Concurrent reads are safe only
while no alias mutates the same backing array. Each `Parse` call builds and
uses its own option config; the package exposes no runtime locale-registration
or mutable parser-registry API.

`ParseResult.MarshalJSON` rejects states that cannot form the tagged-sum wire
contract: unknown statuses or resolved/ambiguous kinds, invalid results without
an error, and ambiguous results with fewer than two resolved same-kind
candidates. Fields omitted by the selected status do not affect the wire shape.

### Diagnostic JSON Output

`ParseResult` JSON is a one-way diagnostic tagged output, not a persistence or
runtime-state recovery format. Depending on `Status`, it serializes the stable
diagnostic fields `status`, `input`, `warnings`, `kind`, `value`, `zone`,
`candidates`, and `error`. The in-process `Reference`, `HasZone`, typed accessor
storage, option presence, and package-owned ambiguity cause are intentionally
not serialized. The single `kind` identifies the concrete value for resolved
or ambiguous results and is omitted for invalid results. Concrete values do not
repeat that discriminator inside `value`.

`ParseResult.UnmarshalJSON` explicitly rejects JSON, including null, with
`ErrInvalidFormat` and leaves its receiver unchanged. Decoding the emitted object
into an arbitrary Go struct does not restore comma-ok accessor values,
ambiguity identity, parser options, or runtime metadata. A caller that needs a
new runtime result must retain the original result or parse the original input
again with explicit options.

```go
switch result.Status() {
case gotime.StatusResolved:
    switch result.Kind() {
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

Accessors such as `DateTime() (DateTime, bool)` and `LocalDateTime() (LocalDateTime, bool)` return `ok=false` unless `Status() == StatusResolved` and the kind matches.

## Options

```go
func WithInputLocale(tag language.Tag) Option
func WithZone(zone Zone) Option
func WithReference(t Instant) Option
```

- `WithInputLocale` enables natural-language parsing. For slash dates, the
  closed policy table supports month-first `en-US` and day-first `en-GB` /
  `en-AU`. Unsupported tags, including bare `en` and `en-CA`, use the same
  validity-based inference as no locale. Unicode `-u-` extensions do not change
  a supported locale's order.
- `WithZone` supplies the zone for formal floating datetimes and the required calendar frame for relative natural date/datetime expressions. Formal local datetimes remain `KindLocalDateTime` when it is omitted.
- `WithReference` supplies the base instant for relative natural date/datetime expressions and is used with `WithZone`.

Option presence is independent from value zero. `WithZone(Zone{})` explicitly
selects UTC, while omitting `WithZone` preserves a formal floating local
datetime but invalidates a reference-dependent natural date/datetime.
`WithReference(Instant{})` explicitly selects the Go zero instant; only an
omitted option means the reference is missing.

There is no `WithStrategy`. Ambiguity is surfaced through `Candidates`; callers decide.

Natural date/datetime expressions that need a calendar reference, such as
`tomorrow` or `next Friday`, require both options. Missing `WithReference`
returns `StatusInvalid` with `ErrInvalidFormat`; missing `WithZone` returns
`StatusInvalid` with `ErrInvalidZone`. Exact natural durations and periods that
resolve directly to `Duration` or `Period` require neither option.

When `WithZone` resolves a formal floating datetime, `ParseResult.Warnings`
includes `WarnAssumedZone`. Without `WithZone`, the same formal input resolves
to `KindLocalDateTime` and carries no zone assumption. Relative natural
date/datetime input instead returns `ErrInvalidZone` when the option is absent.
In diagnostic `Parse`, fractional seconds beyond nanosecond precision in
clock and datetime input produce `WarnTruncatedPrecision` and truncate to nanoseconds. Duration components
with more than nine fractional digits instead return `ErrInvalidDuration`;
values beyond the signed nanosecond range return `ErrOverflow`.
Typed parsers reject a resolved, matching-kind result carrying
`WarnTruncatedPrecision` with a zero value and a `*TimeError` wrapping
`ErrInvalidFormat`. More than nine digits trigger rejection even when the
truncated suffix is all zeroes. Invalid status, ambiguity, and kind mismatch
retain precedence over this precision check. Other warnings remain non-fatal;
callers needing diagnostic values use `Parse` explicitly. A resolved diagnostic
means one interpretation, not permission to submit a lossy value. Consumers must
review warnings; automatic submission uses the typed parser. Incomplete year-month
or ISO-week input is recognized but rejected with a hint instead of inventing
the first day of the month or Monday.
Slash-date candidates use `WarnInferredCalendar` to explain month-first vs
day-first interpretation.

## Ambiguity

Slash dates follow a locale only when it is in the closed policy table.
Otherwise validity decides the state: zero valid interpretations return
`StatusInvalid`, one resolves, and two distinct valid interpretations return
`StatusAmbiguous` with resolved candidates. The parser does not infer likely
regions from language-only tags.

Formal and natural local datetime parsing use `LocalDateTime.Resolve` when
`WithZone` is supplied. DST fall-back local times return `StatusAmbiguous` with
chronological `DateTime` candidates. Each candidate carries
`WarnDuplicateTime` with its abbreviation and offset. DST spring-forward gaps
return `StatusInvalid` with `CodeNonexistentTime`. The natural parser never
normalizes or selects these states before the shared resolver sees them.

Typed parsers preserve the ambiguity cause when translating `StatusAmbiguous` into an error. Slash-date ambiguity returns a `*TimeError` wrapping `ErrAmbiguousDate`. DST fall-back ambiguity returns a `*TimeError` wrapping `ErrDuplicateTime`, including when the duplicate local time appears inside an interval endpoint.

The ambiguity cause is package-owned state, not inferred from warnings.
Removing ambiguity warnings does not change the corresponding typed error
identity. Precision-loss rejection is the separate resolved-result rule above. Typed parsers narrow the same shared parse result; they are not a
second parser engine.

For `StatusResolved` and `StatusAmbiguous` results, `HasZone` reports whether
the original input explicitly included a timezone or offset. `WithZone` does
not set it: a false value does not mean that the result lacks a usable zone.
In `StatusInvalid` results, the field is unspecified; false does not prove that
the input omitted an offset (for example, `2026-02-30T13:00:00Z` is invalid).
To identify a floating datetime, check the result's status and `KindLocalDateTime`
rather than relying on this flag alone.

Interval boundaries must resolve to `KindInstant` or `KindDateTime`. This
constraint applies to every ambiguous candidate before interval construction. Date-only interval boundaries are invalid because an interval is an absolute UTC range and a bare date has no time or zone.
Natural-language interval boundaries are invalid even when `WithInputLocale` and `WithReference` are supplied.
Both interval sides are validated before ambiguity is returned. An invalid or
incompatible side takes precedence over ambiguity; if both sides fail, the
first error in input order is reported.
Valid endpoint candidates are combined into complete intervals. Reversed
combinations are discarded; equal endpoints remain valid empty intervals.
Distinct intervals are sorted by start, then end. Zero valid combinations
return `ErrIntervalReversed`, one resolves, and multiple return ambiguous
`KindInterval` with resolved Interval candidates. Duration forms require a
non-negative exact duration and follow the same candidate rules.
Interval results and their candidates retain complete input text. For resolved
or ambiguous intervals, `HasZone` is the logical OR of the endpoints' flags:
true means at least one endpoint explicitly includes an offset, not that every
endpoint does. For example, an interval with one offset endpoint and one local
endpoint resolved using `WithZone` has `HasZone == true`. Intervals rejected
because every candidate is reversed have no interpretable `HasZone` metadata.
Endpoint warnings are merged in input order, with
equal code/message pairs deduplicated. Candidate fold warnings describe
only the occurrences used by that complete interval.
When a formal interval endpoint is recognized but semantically invalid, the
interval preserves that endpoint's precise error category, such as
`ErrInvalidDate`, `ErrInvalidTime`, `ErrInvalidZone`, `ErrNonexistentTime`, or
`ErrOverflow`. The structured error's `Input` is the complete interval so a
caller can identify the rejected value as one operation. Malformed endpoint
syntax remains `ErrInvalidFormat`, and a recognized but incompatible endpoint
kind remains `ErrIncompatibleTypes`.

## Processing Order

1. Preserve original input, including surrounding whitespace; trim only the parsing work text. Results, errors and nested candidates retain the complete original input.
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

- Relative natural date/datetime input requires `WithReference` and `WithZone`;
  omission returns the corresponding actionable typed error.
- Exact natural durations and periods still resolve without either option when
  locale is supplied.
- Natural and formal datetime inputs produce the same gap/fold status,
  chronological candidates, and typed sentinel for equivalent civil intent.
- Slash-date and DST ambiguity surface through `StatusAmbiguous` and candidates, not a strategy option.
- ParseResult JSON rejects contradictory tagged-sum states before emitting a
  payload.
- Invalid slash dates never surface as ambiguity with invalid candidates.
- Interval tests prove date-only and natural-language boundaries are rejected unless a future spec deliberately changes the interval grammar.

## Controlled Natural Grammar

Chinese (explicit zh-Hans/zh-Hant), Japanese, and Korean week phrases use
Monday–Sunday calendar weeks: previous/current/next week selects the weekday
in that week. English next/last weekday continues to mean the next/previous
occurrence. Korean whitespace accepted by the grammar does not change the
week modifier's meaning. Japanese region and Unicode extension tags route to
the Japanese grammar.

English AM/PM hours must be 1..12 before conversion; 12am is midnight and 12pm
is noon. Invalid hours return `ErrInvalidTime`. Japanese 午前/午後 and Korean
오전/오후 reject marked hours above 12; unmarked hours remain 24-hour input.
Their existing zero-hour and twelve-hour conversions are unchanged. Russian relative units use a
finite table of complete words, including the 1/2/5 forms for seconds, minutes,
hours, days, weeks, and months; arbitrary suffixes return `ErrUnparseable`.

Chinese marked hours are validated before conversion. All four supported
time-of-day words reject hour 0; 早上/上午 also reject hours 12..23, and 晚上
rejects hour 12. These combinations return `ErrInvalidTime` with a hint to use
an ISO date-time with an explicit date and 24-hour clock. The parser does not
guess the midnight date boundary. 下午/晚上 retain explicit 13..23 hours;
marked 1..11 hours keep their existing conversion, and 下午12 is noon.
Unmarked 0..23 hours are 24-hour input; all hours above 23 are invalid.
These are controlled grammar rules, not geographic definitions of day periods.

Chinese 上个月/上個月/下个月/下個月 identify a month, not a calendar day.
They are recognized but return `ErrInvalidDate` with a hint requesting a
complete date; the parser does not fill in day 1. They retain the relative
expression context requirements and missing-context error precedence.
本月底 identifies the final calendar day of the reference month. Relative
month counts such as 1个月后 still produce Period, not Date.

Hindi bare कल and परसों return chronological Date candidates for respectively
±1 and ±2 calendar days; they do not imply a future preference. Typed ParseDate
returns ErrAmbiguousDate. If any alternative leaves the civil year domain, the
whole expression returns ErrInvalidDate rather than choosing the other direction.

## Formal Component Boundaries

Equivalent offset, local, and compact datetime forms report `ErrInvalidDate`
for invalid calendar components and `ErrInvalidTime` for invalid clocks.
Invalid offsets report `ErrInvalidZone`. Actual stdlib parser failures remain
in the error cause chain. Unrecognized text can still be `ErrUnparseable`;
this classification does not expand the grammar.

Time-only parsing accepts `HH:MM:SS.fraction` (decimal point or comma), with
fractions only after explicit seconds. It reads canonical `Time.String()`
values exactly. Diagnostic `Parse` truncates excess precision with
`WarnTruncatedPrecision`; typed `ParseTime` rejects that result. The wire grammar
remains strict. Typed parsers still accept human shorthand such as `09:30`;
they are not validators for another protocol's lexical domain.

Duration parsing and JSON decoding accept the full signed nanosecond range,
including both decomposed and single-seconds spellings of MinInt64. Component
magnitudes and their sum are checked against the sign-specific limit; values
outside it fail instead of wrapping. Human and wire grammars remain distinct.
