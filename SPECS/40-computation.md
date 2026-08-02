# Computation

## Overview

go-time separates exact arithmetic from calendar arithmetic.

- `Duration` is exact nanoseconds (`type Duration time.Duration`).
- `Period` is a calendar offset (`Years`, `Months`, `Days`).

The method names carry the distinction:

```go
next, err := dt.Add(2 * gotime.Hour)                // exact
resolution, err := dt.AddPeriod(gotime.Days(1))    // calendar
```

`dt.Add(gotime.Months(1))` does not compile.

## Exact Arithmetic

```go
func (i Instant) Add(d Duration) Instant
func (dt DateTime) Add(d Duration) (DateTime, error)
```

Move backward by passing a negative duration:

```go
previous, err := dt.Add(-30 * gotime.Minute)
```

`DateTime.Add` returns `ErrOverflow` if the exact result would leave the civil
year domain. `Instant.Add` remains total because `Instant` carries no civil
projection.

There is no `Sub(Duration)` form. `Sub` means exact difference between two timeline values.

## Calendar Arithmetic

```go
func (dt DateTime) AddPeriod(p Period) (LocalResolution, error)
func (d Date) Add(p Period) (Date, error)
```

Move backward with a checked negation:

```go
backward, err := gotime.Months(1).Negate()
if err != nil {
    return err
}
resolution, err := dt.AddPeriod(backward)
```

Month and year arithmetic uses end-of-month clamping:

```go
jan31, _ := gotime.NewDate(2026, time.January, 31)
leap, _ := gotime.NewDate(2024, time.February, 29)

feb28, err := jan31.Add(gotime.Months(1)) // 2026-02-28
next, err := leap.Add(gotime.Years(1))     // 2025-02-28
```

`Date.Add` returns `ErrInvalidDate` for an invalid receiver and `ErrOverflow`
instead of leaving the civil year domain.

`DateTime.AddPeriod` preserves local wall-clock intent across DST transitions.
The target may resolve to zero, one, or multiple candidates; callers use
`LocalResolution.Only()` only when exactly one result is required. A civil-year
overflow returns `ErrOverflow` before a resolution is exposed.
`DateTime.Add(Duration)` preserves elapsed nanoseconds, so wall-clock time may
shift across DST boundaries.

## Difference

```go
func (i Instant) Sub(other Instant) (Duration, error)
func (dt DateTime) Sub(other DateTime) (Duration, error)
func (d Date) DaysUntil(other Date) (int, error)
func (p Period) Add(other Period) (Period, error)
func (p Period) Sub(other Period) (Period, error)
func (p Period) Negate() (Period, error)
func (p Period) Abs() (Period, error)
```

Period component arithmetic returns `ErrOverflow` rather than wrapping an
`int32` component. ISO rendering remains total and supports the full signed
component domain.

Exact timeline differences also return `ErrOverflow` when the true nanosecond
span does not fit `Duration`; they never expose `time.Time.Sub` saturation as
an exact result. `Interval.Length` follows the same rule while preserving long
interval endpoints.

`DaysUntil` returns the signed number of calendar-day boundaries between two
valid dates. It matches `ErrInvalidDate` with an actionable hint when either
endpoint is invalid. Calendar decomposition into years, months, and days is
product policy; go-time does not choose a balancing or end-of-month convention
for callers.

## Comparison

Core ordered values provide `Compare`, plus convenience predicates where implemented:

```go
order := a.Compare(b) // -1, 0, 1
a.Before(b)
a.After(b)
a.Equal(b)
```

Selection and clamping compose with stdlib instead of adding methods to every value object:

```go
earliest := slices.MinFunc(times, gotime.Instant.Compare)
latest := slices.MaxFunc(times, gotime.Instant.Compare)

clamped := v
if v.Compare(lo) < 0 { clamped = lo }
if v.Compare(hi) > 0 { clamped = hi }
```

There are no `Min`, `Max`, or `Clamp` methods.

## Intervals

Intervals are half-open: `[start, end)`.

```go
iv, err := gotime.NewInterval(start, end)
iv, err := gotime.NewIntervalStartingAt(start, 9 * gotime.Hour)
iv, err := gotime.NewIntervalEndingAt(end, 9 * gotime.Hour)
```

Length-based interval constructors reject negative durations with `ErrInvalidDuration`.

Current interval operations:

| Method | Meaning |
|---|---|
| `Start()` | Inclusive start instant. |
| `End()` | Exclusive end instant. |
| `Length()` | Exact duration or `ErrOverflow` when the scalar does not fit. |
| `StdRange()` | UTC stdlib range for external formatting. |
| `Contains(Instant)` | Includes start, excludes end. |
| `Overlaps(Interval)` | True only when intervals share time. Touching endpoints do not overlap. |
| `Adjacent(Interval)` | True when one end equals the other's start. |
| `Intersect(Interval)` | Returns overlap and `ok`. |
| `Union(Interval)` | Merges overlapping or adjacent intervals; disjoint intervals return `ErrIntervalsDisjoint`. |
| `Shift(Duration)` | Moves start and end. |
| `Expand(before, after Duration)` | Moves start backward and end forward; rejects negative expansion durations. |

`NewInterval` returns `ErrIntervalReversed` when `end < start`. `Expand` returns `(Interval, error)` so it can preserve the same invariant as constructors. Zero-length intervals are allowed.

## Duration and Period Display Hooks

`Duration.String()` follows stdlib:

```go
(90 * gotime.Minute).String() // "1h30m0s"
```

`Duration.ISO8601()` and `Period.ISO8601()` provide stable machine strings. Duration sub-second precision uses decimal seconds, never scientific notation. RFC 5545 rendering belongs to calendar integration code outside go-time.

## External Protocol Text

- **Decision**: `ISO8601()` is the only core machine-text helper for `Duration` and `Period`.
- **Why**: ISO text is already the canonical wire representation. Protocol-specific duration dialects have extra validity rules that belong to integrations that own those protocols.
- **Rejected**: Public `RFC5545()` methods on core value objects, especially for calendar periods with year/month fields.
- **Contract Impact**: Core values stay protocol-neutral. Calendar, scheduling, or interchange adapters translate from go-time values outside this module.

## Forbidden

- Do not overload `Add` with both duration and period semantics.
- Do not define `Instant.AddPeriod`.
- Do not let `Date.Add` accept `Duration`.
- Do not let month arithmetic overflow into the following month.
- Do not put zone context on `Interval`.
- Do not add `Min`, `Max`, or `Clamp` methods.
- Do not introduce `StartOf*` or `EndOf*` families.
- Do not add protocol-named rendering methods to core value objects.
- Do not add iteration APIs such as `Each`, `EachDate`, or `Interval.Step` without a separate spec that owns cadence, inclusivity, and DST semantics.

## Acceptance Criteria

- Exact and calendar arithmetic remain separated by type signatures.
- Month/year arithmetic clamps at end of month.
- `DaysUntil` is antisymmetric for valid dates and rejects either invalid
  endpoint with `ErrInvalidDate`.
- Interval operations preserve half-open `[start, end)` semantics.
- No public `RFC5545` method exists on `Duration` or `Period`.
