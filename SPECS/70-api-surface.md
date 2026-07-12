# API Surface

## Overview

All public API lives in the top-level `gotime` package. The design favors concrete immutable values, typed parsing, explicit arithmetic, and stdlib bridges.

There is no public subpackage API and no formatting layer.

## Surface Decisions

### Delete Misleading Convenience

- **Decision**: Public names must describe contracts the package fully owns.
- **Why**: A small foundation API ages better than compatibility shims or protocol-named helpers that overpromise.
- **Rejected**: Keeping `RFC5545()` as a cosmetic alias, keeping `Zone.IsDST()` with heuristic behavior, and adding placeholder strategy or formatter surfaces.
- **Contract Impact**: Removed or forbidden APIs stay out unless a future spec owns their full semantics and verification.

### Prefer Stdlib Bridges Over Product Policy

- **Decision**: go-time exits to stdlib values and simple slots; product policy stays above the module.
- **Why**: Formatting, recurrence, scheduling, locale display, and ambiguity UX depend on product context.
- **Rejected**: Formatter types, global defaults, calendar products, and mutable policy objects.
- **Contract Impact**: New public APIs must either be primitive value semantics or explicit bridges, not product workflow.

## Current Time

```go
func Now() Instant
func NowIn(z Zone) DateTime
func TodayIn(z Zone) Date
```

There is no `Today()`. A calendar date depends on a zone; callers must choose one explicitly.

## Construction

```go
func NewDate(year int, month time.Month, day int) (Date, error)
func NewTime(hour, minute, second int) (Time, error)
func NewTimeNanos(hour, minute, second, nanosecond int) (Time, error)
func NewLocalDateTime(d Date, t Time) LocalDateTime
func NewDateTime(d Date, t Time, z Zone) (DateTime, error)

func InstantFromTime(t time.Time) Instant
func DateTimeFromTime(t time.Time, z Zone) (DateTime, error)
func DateFromTime(t time.Time) (Date, error)
func TimeFromTime(t time.Time) Time

func UnixSeconds(s int64) Instant
func UnixMillis(ms int64) Instant
func UnixNanos(ns int64) Instant

func NewInterval(start, end Instant) (Interval, error)
func NewIntervalStartingAt(start Instant, length Duration) (Interval, error)
func NewIntervalEndingAt(end Instant, length Duration) (Interval, error)

func NewPeriod(years, months, days int32) Period
func Years(n int32) Period
func Months(n int32) Period
func Days(n int32) Period
```

Constructors and projections that can cross the civil domain return errors.
They never normalize invalid dates, invalid clock times, DST gaps, duplicate
local times, or years outside `0000..9999`. Use `DateTimeFromTime` when the
caller already has a stdlib `time.Time` with the intended offset.

```go
func (i Instant) In(z Zone) (DateTime, error)
func (dt DateTime) In(z Zone) (DateTime, error)
```

Both validate after projection into `z` and match `ErrOverflow` if the target
civil year is outside `0000..9999`. `NowIn` and `TodayIn` remain total because
the process clock is inside the civil domain.

`Period` fields are exported, so struct literals are first-class:

```go
gotime.Period{Years: 1, Months: 3, Days: 7}
```

## Local Date-Time Resolution

```go
type LocalDateTime struct {
    Date Date
    Time Time
}

type LocalResolutionStatus string

const (
    LocalInvalid     LocalResolutionStatus = "invalid"
    LocalResolved    LocalResolutionStatus = "resolved"
    LocalNonexistent LocalResolutionStatus = "nonexistent"
    LocalAmbiguous   LocalResolutionStatus = "ambiguous"
)

type LocalResolution struct {
    Status     LocalResolutionStatus
    Zone       Zone
    Local      LocalDateTime
    Candidates []DateTime
}

func (ldt LocalDateTime) Resolve(z Zone) LocalResolution
func (r LocalResolution) Only() (DateTime, error)
```

`LocalDateTime` carries no zone. `Resolve` exposes DST gaps and overlaps without choosing for the caller. `Only` is the deliberate narrowing operation for code that requires exactly one `DateTime`.

## Duration Constants

```go
const (
    Nanosecond Duration = 1
    Microsecond
    Millisecond
    Second
    Minute
    Hour
)
```

There is no `Day` duration constant. Use `24 * gotime.Hour` for exact 24-hour time, and `gotime.Days(n)` for calendar days.

## Parsing

Typed parsers:

```go
func ParseInstant(input string, opts ...Option) (Instant, error)
func ParseDateTime(input string, opts ...Option) (DateTime, error)
func ParseLocalDateTime(input string, opts ...Option) (LocalDateTime, error)
func ParseDate(input string, opts ...Option) (Date, error)
func ParseTime(input string, opts ...Option) (Time, error)
func ParseDuration(input string, opts ...Option) (Duration, error)
func ParsePeriod(input string, opts ...Option) (Period, error)
func ParseInterval(input string, opts ...Option) (Interval, error)
```

Diagnostic parser:

```go
func Parse(input string, opts ...Option) ParseResult
```

Comma-ok accessors:

```go
func (r ParseResult) Instant() (Instant, bool)
func (r ParseResult) DateTime() (DateTime, bool)
func (r ParseResult) LocalDateTime() (LocalDateTime, bool)
func (r ParseResult) Date() (Date, bool)
func (r ParseResult) Time() (Time, bool)
func (r ParseResult) Duration() (Duration, bool)
func (r ParseResult) Period() (Period, bool)
func (r ParseResult) Interval() (Interval, bool)
```

Options:

```go
func WithInputLocale(tag language.Tag) Option
func WithZone(zone Zone) Option
func WithReference(t Instant) Option
```

Reference-dependent natural dates and datetimes require both `WithReference`
and `WithZone`; missing context returns `ErrInvalidFormat` or `ErrInvalidZone`.
Formal floating datetimes may still omit `WithZone` and remain
`LocalDateTime`. Exact natural durations and periods require neither option.

There is no `WithStrategy`, `WithLocale`, or `WithZoneID`.

Parse warning codes are part of the public inspection model:

```go
const (
    WarnAssumedZone WarningCode = "assumed_zone"
    WarnTruncatedPrecision WarningCode = "truncated_precision"
    WarnInferredCalendar WarningCode = "inferred_calendar"
    WarnDuplicateTime WarningCode = "duplicate_time"
)
```

## Arithmetic And Comparison

```go
func (i Instant) Add(d Duration) Instant
func (i Instant) Sub(other Instant) Duration
func (i Instant) Compare(other Instant) int

func (dt DateTime) Add(d Duration) (DateTime, error)
func (dt DateTime) AddPeriod(p Period) (LocalResolution, error)
func (dt DateTime) Sub(other DateTime) Duration
func (dt DateTime) Compare(other DateTime) int

func (d Date) Add(p Period) (Date, error)
func (d Date) DaysUntil(other Date) int
func (d Date) PeriodUntil(other Date) (Period, error)
func (d Date) Compare(other Date) int

func (p Period) Add(other Period) (Period, error)
func (p Period) Sub(other Period) (Period, error)
func (p Period) Negate() (Period, error)
func (p Period) Abs() (Period, error)
```

Use `Compare` with stdlib helpers for selection. There are no `Min`, `Max`, or `Clamp` methods.

## Intervals

```go
func (iv Interval) Start() Instant
func (iv Interval) End() Instant
func (iv Interval) Length() Duration
func (iv Interval) StdRange() (start, end time.Time)
func (iv Interval) Contains(i Instant) bool
func (iv Interval) Overlaps(other Interval) bool
func (iv Interval) Adjacent(other Interval) bool
func (iv Interval) Intersect(other Interval) (Interval, bool)
func (iv Interval) Union(other Interval) (Interval, error)
func (iv Interval) Shift(d Duration) Interval
func (iv Interval) Expand(before, after Duration) (Interval, error)
```

Interval iteration is not part of this API surface. Cadence, inclusivity, and DST behavior belong to a separate scheduling or recurrence contract, not to `Interval`.

## Zones

```go
var UTC Zone

func LoadZone(id string) (Zone, error)
func MustLoadZone(id string) Zone
func ResolveZone(id string) (Zone, error)
func Zones() []string
func ZoneCatalogVersion() string

func (z Zone) ID() string
func (z Zone) Location() *time.Location
func (z Zone) String() string
func (z Zone) Equal(other Zone) bool
func (z Zone) IsZero() bool
```

## Stdlib Bridges

```go
func (i Instant) Std() time.Time
func (dt DateTime) Std() time.Time
func (iv Interval) StdRange() (start, end time.Time)
func (d Duration) Std() time.Duration
func (d Duration) Decompose() DurationComponents
```

`DateTime.Clock()` returns a go-time `Time`, not a stdlib type.
Unresolved `Date` and `Time` values cross to stdlib only after
`NewLocalDateTime(date, clock).Resolve(zone)` produces a chosen `DateTime`.

## JSON

All value objects and `ParseResult` use `github.com/go-json-experiment/json`. Do not add `encoding/json` imports.

Stable value JSON:

- `Instant`: `{"kind":"instant","iso":"..."}`
- `DateTime`: `{"kind":"datetime","instant":"...Z","zone":"..."}`
- `LocalDateTime`: `{"kind":"local_datetime","value":"YYYY-MM-DDTHH:MM:SS"}`
- `Date`: `{"kind":"date","value":"YYYY-MM-DD"}`
- `Time`: `{"kind":"time","value":"HH:MM:SS[.fraction]"}`
- `Duration`: `{"kind":"duration","iso":"..."}`
- `Period`: `{"kind":"period","iso":"..."}`
- `Interval`: `{"kind":"interval","start":"...","end":"..."}`
- `Zone`: `{"kind":"zone","id":"..."}`

Decoding is strict for value-object JSON: unknown fields, missing required
fields, wrong `kind`, non-canonical values, and invalid identities are errors.

`Duration` ISO strings use canonical decimal seconds for sub-second precision; scientific notation is outside the wire domain. A zero `Zone` encodes with `id:"UTC"` to match its total UTC projection behavior.

## Removed Or Forbidden API

- Formatter and locale-display types.
- display i18n, CLDR, message-format, display-locale, and formatter dependencies.
- `gotime.Today()`.
- `gotime.Day` duration constant.
- `Duration.InDays()`.
- `Duration.RFC5545()`.
- `Period.RFC5545()`.
- `Period.Decompose()`.
- `Zone.IsDST()`.
- `WithStrategy`, `Strategy*`, `WithLocale`, `WithZoneID`.
- `Min`, `Max`, `Clamp`.
- `StartOf*`, `EndOf*`.
- `Interval.Step`, `Each`, `EachDate`.
- `ParseResult.Value()`.
- `ParseResult.Locale`, `ParseResult.Strategy`.
- Public `Must*` APIs other than `MustLoadZone`.

## Acceptance Criteria

- `go doc` exposes only top-level `gotime` APIs; no public subpackage becomes part of the contract.
- API inventory contains no formatter, recurrence, strategy, `RFC5545`, or `IsDST` surface.
- `Zone` exposes point-in-time offset and abbreviation, but no DST boolean.
- JSON shapes match the list above and decode strictly.
