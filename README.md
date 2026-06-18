# Go Time

[![Go Version](https://img.shields.io/github/go-mod/go-version/agentable/go-time)](https://github.com/agentable/go-time)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

A Go time semantics library for parsing, computing, and serializing precise time value objects. Hands off to stdlib types — `time.Time`, `time.Duration` — for any rendering / formatting need.

## Features

- **Typed value objects**: Work with `Instant`, `DateTime`, `LocalDateTime`, `Date`, `Time`, `Duration`, `Period`, `Interval`, `Zone`
- **Two parse entry points**: `Parse` returns a tagged-sum `ParseResult`; typed `ParseInstant` / `ParseDateTime` / `ParseLocalDateTime` / … helpers return a typed `(value, error)` pair
- **One concept per type**: `Duration` is exact nanoseconds, `Period` is calendar Y/M/D — `P1Y` parses as Period, `PT1H` as Duration, mixed inputs are rejected
- **Stdlib bridges**: `.Std()`, `.Clock()`, `Duration.Decompose()` send values back to `time.Time` / `time.Duration` / structured clock slots so any formatter can consume them
- **Timezone-aware semantics**: Strict IANA zones, fuzzy zone resolution, explicit local-time resolution, floating-time detection (`ParseResult.HasZone`)
- **Calendar-safe computation**: `Duration.Add` is exact; `AddPeriod` is calendar-aware with end-of-month clamping and DST-stable wall-clock preservation
- **Stable JSON schemas**: Deterministic wire formats with strict decoding; no schema depends on `time.Now()` or runtime locale
- **Zero i18n footprint**: This package ships no locale data, no formatter types — display is the caller's choice (stdlib `time.Format`, logging libraries, templates, or app-owned renderers)

## Installation

```bash
go get github.com/agentable/go-time
```

Requires Go 1.26.3 or newer.

## Quick Start

Most code knows what type it expects — reach for the typed helpers
(`ParseDateTime`, `ParseLocalDateTime`, `ParseInstant`, `ParseDate`, …). They return `(value, error)`
and hide the tagged-sum dispatch:

```go
package main

import (
	"fmt"
	"log"

	gotime "github.com/agentable/go-time"
)

func main() {
	dt, err := gotime.ParseDateTime("2026-03-27T13:00:00",
		gotime.WithZone(gotime.MustLoadZone("Asia/Tokyo")),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dt.Std().Format("2006-01-02 15:04 MST"))
	// 2026-03-27 13:00 JST
}
```

### Natural language

Pass a locale and an explicit reference instant for natural date/datetime
phrases. Product code chooses "now" at the boundary; tests pin a fixed
reference:

```go
zone := gotime.MustLoadZone("America/New_York")
now := gotime.Now()

dt, err := gotime.ParseDateTime("tomorrow at 3pm",
	gotime.WithInputLocale(language.English),
	gotime.WithZone(zone),
	gotime.WithReference(now),
)
```

### Deterministic tests

Use a fixed reference instant when you need reproducible output:

```go
ref, _ := gotime.ParseInstant("2026-03-30T12:00:00Z")

dt, err := gotime.ParseDateTime("tomorrow at 3pm",
	gotime.WithInputLocale(language.English),
	gotime.WithZone(gotime.MustLoadZone("America/New_York")),
	gotime.WithReference(ref),
)
```

## Parsing

### Typed helpers — the default path

When the kind is known up front, the typed helpers return `(value, error)` and
translate `StatusAmbiguous` / `StatusInvalid` / `Kind` mismatch into a `*TimeError`:

```go
i,  err := gotime.ParseInstant("2026-03-27T04:00:00Z")
dt, err := gotime.ParseDateTime("2026-03-27T13:00:00", gotime.WithZone(gotime.MustLoadZone("Asia/Tokyo")))
ldt, err := gotime.ParseLocalDateTime("2026-03-27T13:00:00")
d,  err := gotime.ParseDate("2026-03-27")
t,  err := gotime.ParseTime("15:00")
du, err := gotime.ParseDuration("PT1H30M")
p,  err := gotime.ParsePeriod("P1Y3M")
iv, err := gotime.ParseInterval("2026-03-27T00:00:00Z/2026-03-28T00:00:00Z")
```

A mismatched kind (e.g. `ParseInstant("2026-03-27")`) returns `*TimeError`
wrapping `ErrIncompatibleTypes`.

### `Parse` — inspection / polymorphic dispatch

Reach for `Parse` when you do not know the kind ahead of time, or when you
need access to `Candidates`, `Warnings`, `HasZone`, or `Reference`. The fastest
way to dispatch is `Value()` + a Go type switch:

```go
r := gotime.Parse(input, opts...)

switch v := r.Value().(type) {
case gotime.DateTime:
	fmt.Println(v)
case gotime.LocalDateTime:
	fmt.Println(v)
case gotime.Date:
	fmt.Println(v)
case gotime.Duration:
	fmt.Println(v)
case nil:
	// Status is StatusAmbiguous or StatusInvalid.
	switch r.Status {
	case gotime.StatusAmbiguous:
		for _, c := range r.Candidates {
			fmt.Println(c.Kind)
		}
	case gotime.StatusInvalid:
		fmt.Println(r.Error.Code, r.Error.Hint)
	}
}
```

The comma-ok accessors (`r.DateTime()`, `r.LocalDateTime()`, `r.Date()`, …) are still available for
callers that use `Parse` for metadata but already know the target `Kind`
statically.

### Supported inputs

| Input kind | Examples | Routes to |
|------------|----------|-----------|
| RFC 3339 datetime with offset | `2026-03-27T13:00:00+09:00`, `20260327T130000+0900`, `2026-03-27T04:00:00Z` | `KindInstant` |
| Local datetime | `2026-03-27T13:00:00`, `20260327T130000` | `KindLocalDateTime`; `KindDateTime` only with `WithZone` |
| Date | `2026-03-27`, `2026-W13-5`, `2026-086`, `04/05/2026` (locale-disambiguated) | `KindDate` |
| Time | `15:00`, `08:30:45`, `3:30 PM` | `KindTime` |
| ISO duration (clock only) | `PT1H30M`, `PT0S` | `KindDuration` |
| ISO period (calendar only) | `P1Y3M7D`, `P2W`, `P5D` | `KindPeriod` |
| Mixed `P{date}T{time}` | `P1DT2H`, `P1Y2DT3H` | `StatusInvalid` + `CodeInvalidFormat` |
| Interval | `2026-03-27T00:00:00Z/2026-03-28T00:00:00Z`, `2026-03-27T09:00:00Z/PT9H` | `KindInterval` |
| Natural language | `tomorrow at 3pm`, `下周五下午三点`, `明日`, `다음 주 금요일 오후 3시` | varies |

`Duration` and `Period` are distinct types and cannot mix in a single ISO string: a 24-hour clock span is `PT24H` (Duration), a calendar day is `P1D` (Period).

### Parse options

| Option | Purpose |
|--------|---------|
| `WithInputLocale(tag language.Tag)` | Language hint for natural-language input. Also disambiguates slash-date order (e.g. `04/05/2026` is May 4 in `en-GB`, April 5 in `en-US`). |
| `WithZone(zone)` | Resolve floating datetimes into `DateTime`; without it they remain `LocalDateTime` |
| `WithReference(instant)` | Anchor natural date/datetime expressions such as `tomorrow` or `next Friday` |

Ambiguity is surfaced through `ParseResult.Candidates`, not pre-selected by an option — callers decide. There is no `WithStrategy` knob.

`WithInputLocale` takes [`golang.org/x/text/language.Tag`](https://pkg.go.dev/golang.org/x/text/language#Tag) — the de facto Go BCP-47 type. Only the language subtag is used; Unicode `-u-` extensions (hour cycle, calendar) belong to the rendering layer, not the parser.

`ParseResult.HasZone` reports whether the input itself included an explicit zone or offset. Floating datetime input without `WithZone` stays as `KindLocalDateTime`; adding `WithZone` resolves it into a zoned `DateTime`.

When the zone identifier comes from user input, resolve it at the call site:

```go
z, err := gotime.ResolveZone(userInput)
if err != nil {
	return err
}
r := gotime.Parse(input, gotime.WithZone(z))
```

## Formatting

**This package does not format.** Display is the caller's concern. Every value object exposes a small, stable bridge back to stdlib so any formatter can consume it.

### Stdlib `time.Format`

```go
fmt.Println(dt.Std().Format("2006-01-02 15:04 MST"))
// 2026-03-27 13:00 JST
```

### External renderers

```go
type DateTimeRenderer interface {
	Format(time.Time) string
}

func render(dt gotime.DateTime, renderer DateTimeRenderer) string {
	return renderer.Format(dt.Std())
}
```

### Duration → clock slots, Period → exported fields

```go
// Duration: modular arithmetic is non-trivial, so a helper does it for you.
c := d.Decompose() // DurationComponents{Hours, Minutes, Seconds, ...}

clockSlots := struct {
	Hours        int64
	Minutes      int64
	Seconds      int64
	Milliseconds int64
}{
	Hours:        c.Hours,
	Minutes:      c.Minutes,
	Seconds:      c.Seconds,
	Milliseconds: c.Milliseconds,
}

// Period: fields are already exported — no Decompose, no parallel struct.
calendarSlots := struct {
	Years  int32
	Months int32
	Days   int32
}{
	Years:  p.Years,
	Months: p.Months,
	Days:   p.Days,
}
```

### Bridge methods

| Method | Returns |
|--------|---------|
| `Instant.Std()` | `time.Time` (UTC) |
| `DateTime.Std()` | `time.Time` in the DateTime's zone |
| `DateTime.Clock()` | `gotime.Time` (clock-time accessor) |
| `Date.Std(z Zone)` | `time.Time` at 00:00 in `z` |
| `Time.Std(on Date, z Zone)` | `time.Time` |
| `Interval.StdRange()` | `(start, end time.Time)` |
| `Duration.Std()` | `time.Duration` |
| `Duration.Decompose()` | `DurationComponents` (clock slots) |
| `Period.Years` / `.Months` / `.Days` | `int32` fields (no Decompose) |

See [`SPECS/30-formatting.md`](SPECS/30-formatting.md) for the full contract.

## Core Types

| Type | Meaning |
|------|---------|
| `Instant` | Absolute UTC moment |
| `DateTime` | Zoned local date and time |
| `LocalDateTime` | Date plus clock time before zone resolution |
| `Date` | Calendar date without time |
| `Time` | Clock time without date |
| `Duration` | Exact elapsed nanoseconds (`type Duration time.Duration`) |
| `Period` | Calendar offset of `Years`/`Months`/`Days` |
| `Interval` | Half-open range `[start, end)` |
| `Zone` | IANA timezone identity for persisted zones |
| `DurationComponents` | Clock-slot decomposition of `Duration` (Hours…Nanoseconds) — bridge to external duration formatters |

## Duration vs Period

`Duration` is exact nanoseconds — composed from typed constants (`Nanosecond`, `Microsecond`, `Millisecond`, `Second`, `Minute`, `Hour`):

```go
quarter   := 15 * gotime.Minute
twoDaysExact := 48 * gotime.Hour  // 48 exact elapsed hours
```

`Period` is a calendar offset — built via struct literal or sugar constructors (`Years`, `Months`, `Days`). Calendar days are DST-safe; they preserve wall-clock time across the boundary:

```go
nextMonth   := gotime.Period{Months: 1}
twoCalDays  := gotime.Days(2)        // 2 calendar days (DST-safe)
twoWeeks    := gotime.Days(14)
```

There is intentionally no `gotime.Day` constant. `24 * gotime.Hour` is exact 24 hours; `gotime.Days(1)` is a calendar day. Conflating them is the bug the type split exists to prevent.

`DateTime` exposes:

- `dt.Add(d Duration)` — exact-time arithmetic; pass `-d` to move back
- `dt.AddPeriod(p Period)` — calendar arithmetic with end-of-month clamping; pass `p.Negate()` to move back
- `dt.Sub(other DateTime) Duration` — difference between two DateTimes (mirrors `time.Time.Sub`)

`Date.Add(p Period) Date` handles calendar arithmetic. For differences, use the name that matches your intent: `d.DaysUntil(other)` for signed calendar-day counts, or `d.PeriodUntil(other)` for a greedy years/months/days period. `Instant.Add(d Duration) Instant` and `Instant.Sub(other Instant) Duration` round out the exact-time trio. The compiler enforces the distinction between Duration and Period — `dt.Add(gotime.Months(1))` is a compile error.

## Common Operations

```go
zone := gotime.MustLoadZone("America/New_York")
tokyo := gotime.MustLoadZone("Asia/Tokyo")

now := gotime.Now()                  // Instant (UTC, no monotonic)
today := gotime.TodayIn(zone)        // Date — zone is required, never inferred
fromEpoch := gotime.UnixMillis(0)    // Instant from Unix epoch milliseconds

date, err := gotime.NewDate(2026, time.March, 27)
if err != nil {
	fmt.Println("date error:", err)
	return
}
clock, err := gotime.NewTime(9, 0, 0)
if err != nil {
	fmt.Println("time error:", err)
	return
}
dt, err := gotime.NewDateTime(date, clock, zone)
if err != nil {
	fmt.Println("datetime error:", err)
	return
}

local := gotime.NewLocalDateTime(date, clock)
resolution := local.Resolve(zone)
if resolution.Status == gotime.LocalAmbiguous {
	fmt.Println("choose between", resolution.Candidates)
}

exact := dt.Add(24 * gotime.Hour)      // exact 24-hour span (Duration)
calendar := dt.AddPeriod(gotime.Days(1)) // 1 calendar day, DST-safe (Period)
shifted := dt.In(tokyo)

iv, err := gotime.NewInterval(dt.Instant(), exact.Instant())
if err != nil {
	fmt.Println("interval error:", err)
	return
}

// Selection composes with stdlib slices.MinFunc / MaxFunc + Compare.
import "slices"
earliest := slices.MinFunc(
	[]gotime.Instant{now, fromEpoch, dt.Instant()},
	gotime.Instant.Compare,
)

fmt.Println(now, today, exact, calendar, shifted, iv.Length(), earliest)
```

## Timezones

Use `LoadZone` when you have a canonical IANA zone ID.
Use `ResolveZone` only when you accept messy real-world input and need to normalize it.
`MustLoadZone` is for `var` initialization with constant identifiers.

| Call | Use case |
|------|----------|
| `LoadZone("Asia/Tokyo")` | Strict IANA, returns error on miss |
| `MustLoadZone("Asia/Tokyo")` | Constant zones in `var` blocks |
| `ResolveZone("asia/tokyo")` | Forgiving case normalization |
| `ResolveZone("Eastern Standard Time")` | Windows zone names |
| `Zone{}` | Zero value, treated as UTC for stdlib interop |

```go
tokyo, err := gotime.LoadZone("Asia/Tokyo")
if err != nil {
	return err
}
fmt.Println(tokyo.ID(), tokyo.OffsetAt(gotime.Now()))
fmt.Println(len(gotime.Zones()), gotime.ZoneCatalogVersion())
```

`ZoneCatalogVersion()` identifies the tzdb version used to generate the `Zones()` catalog. It is catalog provenance, not a claim about the transition-rule data used by the Go runtime.

For interval rendering, project explicitly: `start, end := iv.StdRange()` returns two `time.Time` values you can pass to any range-formatter.

When a date plus clock time may cross a DST transition, resolve it explicitly:

```go
local := gotime.NewLocalDateTime(date, clock)
resolution := local.Resolve(zone)
dt, err := resolution.Only() // ErrNonexistentTime or ErrDuplicateTime if not unique
```

## Errors

go-time pairs sentinel errors (for `errors.Is`) with a typed `*TimeError` (for `errors.As`).

```go
var te *gotime.TimeError
if errors.As(err, &te) {
	log.Printf("code=%s hint=%s", te.Code, te.Hint)
}

if errors.Is(err, gotime.ErrAmbiguousDate) {
	// control-flow branch
}
```

`*TimeError` unwraps its `Err` sentinel for control flow; `Code` remains JSON/log metadata.
See `SPECS/60-errors.md` for the full code list.

## JSON

All value objects implement stable JSON encoding via `github.com/go-json-experiment/json`. Decoding is strict: wrong `kind`, missing required fields, unknown fields, unsupported calendars, and contradictory derived fields are rejected.

```go
b, err := json.Marshal(dt)
if err != nil {
	return err
}
fmt.Println(string(b))
```

Representative shapes:

```json
{"kind":"datetime","value":"2026-03-27T09:00:00-04:00","zone":"America/New_York","calendar":"iso8601"}
{"kind":"local_datetime","value":"2026-03-27T09:00:00","calendar":"iso8601"}
{"kind":"duration","iso":"PT1H30M"}
{"kind":"period","iso":"P1Y3M7D"}
{"kind":"zone","id":"America/New_York"}
```

- `Duration` / `Period` carry **only** `kind` + `iso` — no `components` / `years` / `months` / `days` mirror fields. Run `Duration.Decompose()` for clock slots; read `Period.Years` / `.Months` / `.Days` directly.
- `Zone` JSON contains only `{"kind":"zone","id":"…"}`. Time-dependent display data (offset and abbreviation) lives in `Zone.Snapshot(at Instant)` — never on the marshalled value, so the same `Zone` encodes byte-identical regardless of when you call `Marshal`.

## API Reference

- Package docs: [pkg.go.dev/github.com/agentable/go-time](https://pkg.go.dev/github.com/agentable/go-time)
- Project contracts: [SPECS/](SPECS/)

## Development

```bash
task test
task lint
task fmt
task vet
task verify
```

For development guidelines and agent workflow, see [AGENTS.md](AGENTS.md).

## Contributing

Open an issue before large changes so the public API and specs can stay aligned.

## License

This software is licensed under the **MIT License**.
See the [LICENSE](./LICENSE) file for full terms.
