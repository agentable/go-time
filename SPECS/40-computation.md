# Computation

## Overview

go-time separates exact arithmetic from calendar arithmetic.

- `Duration` is exact nanoseconds (`type Duration time.Duration`).
- `Period` is a calendar offset (`Years`, `Months`, `Days`).

The method names carry the distinction:

```go
dt.Add(2 * gotime.Hour)        // exact
dt.AddPeriod(gotime.Days(1))   // calendar
```

`dt.Add(gotime.Months(1))` does not compile.

## Exact Arithmetic

```go
func (i Instant) Add(d Duration) Instant
func (dt DateTime) Add(d Duration) DateTime
```

Move backward by passing a negative duration:

```go
dt.Add(-30 * gotime.Minute)
```

There is no `Sub(Duration)` form. `Sub` means difference between two values.

## Calendar Arithmetic

```go
func (dt DateTime) AddPeriod(p Period) DateTime
func (d Date) Add(p Period) Date
```

Move backward with `p.Negate()`:

```go
dt.AddPeriod(gotime.Months(1).Negate())
```

Month and year arithmetic uses end-of-month clamping:

```go
jan31, _ := gotime.NewDate(2026, time.January, 31)
leap, _ := gotime.NewDate(2024, time.February, 29)

jan31.Add(gotime.Months(1)) // 2026-02-28
leap.Add(gotime.Years(1))   // 2025-02-28
```

`DateTime.AddPeriod` preserves local wall-clock time across DST transitions. `DateTime.Add(Duration)` preserves elapsed nanoseconds, so wall-clock time may shift across DST boundaries.

## Difference

```go
func (i Instant) Sub(other Instant) Duration
func (dt DateTime) Sub(other DateTime) Duration
func (d Date) Sub(other Date) Period
func (p Period) Sub(other Period) Period
```

`Sub` is the one verb for difference. There is no parallel `Diff`.

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
iv := gotime.IntervalOf(start, 9 * gotime.Hour)
```

Current interval operations:

| Method | Meaning |
|---|---|
| `Start()` | Inclusive start instant. |
| `End()` | Exclusive end instant. |
| `Length()` | Exact duration. |
| `StdRange()` | UTC stdlib range for external formatting. |
| `Contains(Instant)` | Includes start, excludes end. |
| `Overlaps(Interval)` | True only when intervals share time. Touching endpoints do not overlap. |
| `Adjacent(Interval)` | True when one end equals the other's start. |
| `Intersect(Interval)` | Returns overlap and `ok`. |
| `Union(Interval)` | Merges overlapping or adjacent intervals; disjoint intervals return `ErrIntervalsDisjoint`. |
| `Shift(Duration)` | Moves start and end. |
| `Expand(before, after Duration)` | Moves start backward and end forward. |

`NewInterval` returns `ErrIntervalReversed` when `end < start`. Zero-length intervals are allowed.

## Duration and Period Display Hooks

`Duration.String()` follows stdlib:

```go
(90 * gotime.Minute).String() // "1h30m0s"
```

`Duration.ISO8601()` and `Period.ISO8601()` provide stable machine strings. `RFC5545()` exists on both types and does not return an error.

## Not Implemented

The current implementation does not expose `Instant.Truncate`, `Instant.Round`, `Each`, `EachDate`, or `Interval.Step`. Specs must not claim those APIs until code exists.

## Forbidden

- Do not overload `Add` with both duration and period semantics.
- Do not define `Instant.AddPeriod`.
- Do not let `Date.Add` accept `Duration`.
- Do not let month arithmetic overflow into the following month.
- Do not put zone context on `Interval`.
- Do not add `Min`, `Max`, or `Clamp` methods.
- Do not introduce `StartOf*` or `EndOf*` families.
