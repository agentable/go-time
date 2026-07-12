# Domain Model

## Overview

go-time has nine immutable value objects. Each represents exactly one time concept and exposes only the operations that make sense for that concept.

The most important split is `Duration` vs `Period`: exact elapsed time and calendar offsets are different types, so the compiler prevents accidental mixing.

## Contract Decisions

### Closed Value JSON

- **Decision**: Value-object JSON is a closed contract. Decoders reject unknown
  fields, missing required fields, wrong `kind`, non-canonical values, and
  invalid identities.
- **Why**: Persisted time payloads must fail at the boundary when they carry two competing facts. Silent acceptance makes old or malformed data look authoritative.
- **Rejected**: Lenient decode for compatibility, mirror component fields, and display fields embedded in value JSON.
- **Contract Impact**: A wire payload either maps to one semantic value or returns a typed error. There is no best-effort merge.

### Constructible Domain

- **Decision**: Checked constructors and parse/unmarshal paths share the stable wire domain. `Date` years are `0000..9999`, matching the fixed-width ISO calendar shape used by parsing and JSON.
- **Why**: A caller should not be able to create a checked calendar value that the public wire contract cannot carry.
- **Rejected**: Extended years without a new wire grammar, and constructor normalization of invalid dates.
- **Contract Impact**: Invalid calendar components fail construction instead of being normalized.

All public stdlib and zone projections validate the resulting civil year.
`DateFromTime`, `DateTimeFromTime`, `Instant.In`, and `DateTime.In` return
`ErrOverflow` rather than producing a `Date` or `DateTime` outside
`0000..9999`. `NowIn` and `TodayIn` remain total because the process clock is
already inside that domain.

## Value Objects

### Instant

An absolute UTC moment with nanosecond precision.

Use it for storage, logs, ordering, and cross-system transfer.

JSON:

```json
{"kind":"instant","iso":"2026-03-27T04:00:00Z"}
```

Key API: `InstantFromTime`, `UnixSeconds`, `UnixMillis`, `UnixNanos`, `Std`, `UnixNano`, `UnixMilli`, `In`, `Add(Duration)`, `Sub(Instant)`, `Compare`.

`iso` is the only instant wire fact. Callers that need epoch milliseconds use
`UnixMilli()` after decoding. Marshal rejects an instant whose UTC year cannot
be represented by the matching RFC3339 decoder.

### DateTime

A calendar date and clock time resolved in a specific `Zone`.

JSON:

```json
{"kind":"datetime","instant":"2026-03-27T04:00:00Z","zone":"Asia/Tokyo"}
```

Key API: `NewDateTime`, `DateTimeFromTime`, `Date`, `Clock`, `Std`, `Zone`, `Instant`, `In`, `Add(Duration)`, `AddPeriod(Period)`, `Sub(DateTime)`, `Compare`.

`NewDateTime` is the convenience path for `NewLocalDateTime(d, t).Resolve(z).Only()`. DST gaps return `ErrNonexistentTime`; DST duplicates return `ErrDuplicateTime`. Use `DateTimeFromTime` when the caller already has a stdlib `time.Time` with the intended offset.

`instant` owns the absolute time and `zone` owns the projection identity.
Unmarshal establishes the instant directly, then projects it through the named
zone. It does not store or revalidate a redundant local offset against current
timezone rules.

`Add(Duration)` is exact arithmetic. `AddPeriod(Period)` is checked calendar
arithmetic: it preserves local wall-clock intent, applies end-of-month clamping,
and returns a `LocalResolution` so a target DST gap or overlap is not silently
normalized or selected.
`Add(Duration)` returns `ErrOverflow` if the exact result would project outside
the `DateTime` civil year domain; `Instant.Add(Duration)` remains total because
an `Instant` has no civil projection.

### LocalDateTime

A calendar date and clock time without timezone or offset.

JSON:

```json
{"kind":"local_datetime","value":"2026-03-27T13:00:00"}
```

Key API: `NewLocalDateTime`, `Resolve`, `String`.

`LocalDateTime` cannot identify an instant until resolved in a `Zone`. `Resolve` returns a `LocalResolution`:

- `LocalResolved` carries exactly one chronological `DateTime` candidate.
- `LocalNonexistent` carries no candidates for DST gaps.
- `LocalAmbiguous` carries multiple chronological candidates for DST overlaps.
- `LocalInvalid` reports invalid zero or corrupted components.

`LocalResolution.Only()` returns the single `DateTime` or a typed error (`ErrNonexistentTime`, `ErrDuplicateTime`, `ErrInvalidDate`, or `ErrInvalidTime`).

### Date

A calendar date without time or timezone.

JSON:

```json
{"kind":"date","value":"2026-03-27"}
```

Key API: `NewDate`, `DateFromTime`, `Year`, `Month`, `Day`, `Weekday`, `ISOWeek`, `YearDay`, `DaysInMonth`, `IsLeapYear`, `Add(Period)`, `DaysUntil(Date)`, `PeriodUntil(Date)`, `Compare`.

`NewDate` returns an error for invalid calendar components. It accepts the stable wire-domain year range `0000..9999` and never normalizes values such as February 31.

`Date` accepts `Period`, never `Duration`.

### Time

A clock time without date or timezone.

JSON:

```json
{"kind":"time","value":"15:00:00"}
```

Sub-second values use the canonical trimmed decimal representation:

```json
{"kind":"time","value":"15:00:00.123"}
```

Key API: `NewTime`, `NewTimeNanos`, `TimeFromTime`, `Hour`, `Minute`, `Second`, `Nanosecond`.

`NewTime` and `NewTimeNanos` return errors for out-of-range clock components. They never normalize values such as 24:00:00 or nanosecond 1,000,000,000.

`value` is the only clock-time wire fact. Decode rejects non-canonical trailing
zeros rather than preserving input precision provenance.

### Duration

An exact elapsed span represented as `type Duration time.Duration`.

Constants mirror stdlib through `Hour`; there is intentionally no `Day` constant.

```go
5 * gotime.Minute
24 * gotime.Hour
```

JSON:

```json
{"kind":"duration","iso":"PT1H30M"}
```

Key API: `Std`, `Nanoseconds`, `Milliseconds`, `InSeconds`, `InMinutes`, `InHours`, `Abs`, `String`, `ISO8601`, `Decompose`.

`Duration.String()` matches `time.Duration.String()` byte-for-byte. `Duration.Decompose()` exposes clock slots only: hours, minutes, seconds, milliseconds, microseconds, nanoseconds.

`Duration.ISO8601()` emits canonical decimal seconds for sub-second precision. Scientific notation is not part of the wire domain.

### Period

A calendar offset in years, months, and calendar days.

```go
gotime.Period{Years: 1, Months: 3, Days: 7}
gotime.Months(1)
gotime.Days(7)
```

JSON:

```json
{"kind":"period","iso":"P1Y3M7D"}
```

Key API: exported `Years`, `Months`, `Days` fields, `NewPeriod`, `Years`, `Months`, `Days` constructors, `Negate`, `Abs`, `Add`, `Sub`, `ISO8601`, `String`.

Month/year arithmetic clamps to the end of the target month. `Period` has no `Decompose`; callers read the exported fields directly.
Component arithmetic is checked: `Add`, `Sub`, `Negate`, and `Abs` return
`ErrOverflow` when the exact result does not fit `int32`.

### Interval

A half-open UTC interval `[start, end)` bounded by two `Instant` values.

JSON:

```json
{"kind":"interval","start":"2026-03-27T00:00:00Z","end":"2026-03-27T09:00:00Z"}
```

Key API: `NewInterval`, `NewIntervalStartingAt`, `NewIntervalEndingAt`, `Start`, `End`, `Length`, `StdRange`, `Contains`, `Overlaps`, `Adjacent`, `Intersect`, `Union`, `Shift`, `Expand`.

`Interval` carries no zone. Projection for display happens outside the package via `StdRange`.

### Zone

A timezone identity backed by `time.Location`.

JSON:

```json
{"kind":"zone","id":"Asia/Tokyo"}
```

Key API: `LoadZone`, `MustLoadZone`, `ResolveZone`, `Zones`, `ID`, `Location`.

`Zone.Location()` is total: a zero `Zone` falls back to UTC. Time-dependent
abbreviations and numeric offsets come from the stdlib bridge:

```go
name, offsetSeconds := instant.Std().In(zone.Location()).Zone()
```

Zero `Zone` values have UTC semantics on `ID`, `String`, `Equal`, `Location`,
and JSON.

## Relationships

| Relationship | Meaning |
|---|---|
| `Instant.In(Zone) -> (DateTime, error)` | Project an absolute moment into a zone; reject a civil-year overflow. |
| `NewLocalDateTime(Date, Time) -> LocalDateTime` | Combine calendar and clock without choosing a zone. |
| `LocalDateTime.Resolve(Zone) -> LocalResolution` | Project local wall time into a zone, preserving DST ambiguity. |
| `LocalResolution.Only() -> DateTime` | Demand exactly one resolved candidate. |
| `DateTime.Instant() -> Instant` | Convert a zoned local time to the absolute timeline. |
| `DateTime.In(Zone) -> (DateTime, error)` | Same moment, different zone; reject a civil-year overflow. |
| `DateTime.Sub(DateTime) -> Duration` | Exact elapsed difference. |
| `Date.DaysUntil(Date) -> int` | Signed calendar-day count. |
| `Date.PeriodUntil(Date) -> (Period, error)` | Signed greedy calendar difference; reject invalid endpoints. |
| `Interval.Length() -> Duration` | Exact interval length. |

## Wire Format Invariance

JSON shapes are part of the long-term contract.

- Fields are closed. Adding a field is a deliberate wire-format change, not an accidental extension point.
- Existing field names and meanings must not change.
- Each payload has one semantic source of truth.
- `Duration` and `Period` JSON use `iso`; derived components stay outside the wire payload.
- Marshal output must not depend on `time.Now()`, process locale, mutable globals, or ambient zone state.
- Unmarshal rejects wrong `kind`, missing required fields, unknown fields,
  non-canonical values, and invalid identities.

## Acceptance Criteria

- JSON tests reject unknown fields, missing required fields, wrong `kind`,
  non-canonical values, and invalid identities.
- `Marshal -> Unmarshal -> Marshal` remains byte-stable for accepted value payloads.
- Date construction rejects years outside `0000..9999` and invalid calendar components.
- Zone JSON remains `{"kind":"zone","id":"..."}` and never carries offset, abbreviation, or DST display fields.

## Forbidden

- Do not merge `Instant` and `DateTime`.
- Do not make value objects mutable.
- Do not couple `Zone` and locale.
- Do not give `Duration` calendar semantics.
- Do not put zone context on `Interval`.
- Do not serialize derived display fields from `Zone.MarshalJSON`.
- Do not replace typed values with `string` or `any`.
