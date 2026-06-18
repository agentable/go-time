# Domain Model

## Overview

go-time has nine immutable value objects. Each represents exactly one time concept and exposes only the operations that make sense for that concept.

The most important split is `Duration` vs `Period`: exact elapsed time and calendar offsets are different types, so the compiler prevents accidental mixing.

## Contract Decisions

### Closed Value JSON

- **Decision**: Value-object JSON is a closed contract. Decoders reject unknown fields, missing required fields, wrong `kind`, unsupported calendars, and contradictory derived fields.
- **Why**: Persisted time payloads must fail at the boundary when they carry two competing facts. Silent acceptance makes old or malformed data look authoritative.
- **Rejected**: Lenient decode for compatibility, mirror component fields, and display fields embedded in value JSON.
- **Contract Impact**: A wire payload either maps to one semantic value or returns a typed error. There is no best-effort merge.

### Constructible Domain

- **Decision**: Checked constructors and parse/unmarshal paths share the stable wire domain. `Date` years are `0000..9999`, matching the fixed-width ISO calendar shape used by parsing and JSON.
- **Why**: A caller should not be able to create a checked calendar value that the public wire contract cannot carry.
- **Rejected**: Extended years without a new wire grammar, and constructor normalization of invalid dates.
- **Contract Impact**: Invalid calendar components fail construction instead of being normalized.

## Value Objects

### Instant

An absolute UTC moment with nanosecond precision.

Use it for storage, logs, ordering, and cross-system transfer.

JSON:

```json
{"kind":"instant","iso":"2026-03-27T04:00:00Z","epoch_ms":1774584000000}
```

Key API: `InstantFromTime`, `UnixSeconds`, `UnixMillis`, `UnixNanos`, `Std`, `UnixNano`, `UnixMilli`, `In`, `Add(Duration)`, `Sub(Instant)`, `Compare`.

`iso` owns the instant value. `epoch_ms` is a mandatory cross-check field for systems that index by epoch milliseconds; unmarshal rejects payloads where it does not match `iso`.

### DateTime

A calendar date and clock time resolved in a specific `Zone`.

JSON:

```json
{"kind":"datetime","value":"2026-03-27T13:00:00+09:00","zone":"Asia/Tokyo","calendar":"iso8601"}
```

Key API: `NewDateTime`, `DateTimeFromTime`, `Date`, `Clock`, `Std`, `Zone`, `Instant`, `In`, `Add(Duration)`, `AddPeriod(Period)`, `Sub(DateTime)`, `Compare`.

`NewDateTime` is the convenience path for `NewLocalDateTime(d, t).Resolve(z).Only()`. DST gaps return `ErrNonexistentTime`; DST duplicates return `ErrDuplicateTime`. Use `DateTimeFromTime` when the caller already has a stdlib `time.Time` with the intended offset.

When unmarshaling JSON, the offset embedded in `value` must match the IANA `zone` at that instant. A mismatch returns `ErrInvalidZone`; the wire format must not contain two conflicting sources of truth.

`Add(Duration)` is exact arithmetic. `AddPeriod(Period)` is calendar arithmetic and preserves local wall-clock time across DST transitions.

### LocalDateTime

A calendar date and clock time without timezone or offset.

JSON:

```json
{"kind":"local_datetime","value":"2026-03-27T13:00:00","calendar":"iso8601"}
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
{"kind":"date","value":"2026-03-27","calendar":"iso8601"}
```

Key API: `NewDate`, `DateFromTime`, `Year`, `Month`, `Day`, `Std(Zone)`, `Weekday`, `ISOWeek`, `YearDay`, `DaysInMonth`, `IsLeapYear`, `Add(Period)`, `DaysUntil(Date)`, `PeriodUntil(Date)`, `Compare`.

`NewDate` returns an error for invalid calendar components. It accepts the stable wire-domain year range `0000..9999` and never normalizes values such as February 31.

`Date` accepts `Period`, never `Duration`.

### Time

A clock time without date or timezone.

JSON:

```json
{"kind":"time","value":"15:00:00","precision":"second"}
```

Sub-second values use the finest precision represented by the nanosecond field:

```json
{"kind":"time","value":"15:00:00.123000000","precision":"millisecond"}
```

Key API: `NewTime`, `NewTimeNanos`, `TimeFromTime`, `Hour`, `Minute`, `Second`, `Nanosecond`, `Std(Date, Zone)`.

`NewTime` and `NewTimeNanos` return errors for out-of-range clock components. They never normalize values such as 24:00:00 or nanosecond 1,000,000,000.

`value` owns the clock time. `precision` is a mandatory cross-check field describing the finest represented clock slot; unmarshal rejects mismatches.

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

Key API: `LoadZone`, `MustLoadZone`, `ResolveZone`, `Zones`, `ID`, `Location`, `Snapshot`, `OffsetAt`, `Abbreviation`.

`Zone.Location()` is total: a zero `Zone` falls back to UTC. Time-dependent display data is explicit:

```json
{"id":"Asia/Tokyo","offset":"+09:00","abbreviation":"JST"}
```

## Relationships

| Relationship | Meaning |
|---|---|
| `Instant.In(Zone) -> DateTime` | Project an absolute moment into a zone. |
| `NewLocalDateTime(Date, Time) -> LocalDateTime` | Combine calendar and clock without choosing a zone. |
| `LocalDateTime.Resolve(Zone) -> LocalResolution` | Project local wall time into a zone, preserving DST ambiguity. |
| `LocalResolution.Only() -> DateTime` | Demand exactly one resolved candidate. |
| `DateTime.Instant() -> Instant` | Convert a zoned local time to the absolute timeline. |
| `DateTime.In(Zone) -> DateTime` | Same moment, different zone. |
| `Time.Std(Date, Zone) -> time.Time` | Combine date, clock, and zone for stdlib interop. |
| `Date.Std(Zone) -> time.Time` | Project a date to midnight in a zone. |
| `DateTime.Sub(DateTime) -> Duration` | Exact elapsed difference. |
| `Date.DaysUntil(Date) -> int` | Signed calendar-day count. |
| `Date.PeriodUntil(Date) -> Period` | Signed greedy calendar difference, preferring years, then months, then days. |
| `Interval.Length() -> Duration` | Exact interval length. |

## Wire Format Invariance

JSON shapes are part of the long-term contract.

- Fields are closed. Adding a field is a deliberate wire-format change, not an accidental extension point.
- Existing field names and meanings must not change.
- Each payload has one semantic source of truth. Required cross-check fields such as `epoch_ms` and `precision` must match that source exactly and never introduce additional semantics.
- `Duration` and `Period` JSON use `iso`; derived components stay outside the wire payload.
- Marshal output must not depend on `time.Now()`, process locale, mutable globals, or ambient zone state.
- Unmarshal rejects wrong `kind`, missing required fields, unknown fields, unsupported calendars, and contradictory derived fields such as mismatched `epoch_ms` or `precision`.

## Acceptance Criteria

- JSON tests reject unknown fields, missing required fields, wrong `kind`, unsupported calendars, and contradictory derived fields.
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
