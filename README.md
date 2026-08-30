# go-time
[![Go Version](https://img.shields.io/github/go-mod/go-version/agentable/go-time)](https://github.com/agentable/go-time)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

[![Go CI](https://github.com/agentable/go-time/actions/workflows/ci.yml/badge.svg)](https://github.com/agentable/go-time/actions/workflows/ci.yml)

A Go time semantics library that turns standard formats and controlled human expressions into precise, typed, and computable values

## Features

- **Typed time semantics**: Represent instants, zoned and local datetimes, dates, clock times, durations, periods, intervals, and zones as distinct values with non-mutating value methods
- **Two parsing paths**: Use typed `Parse*` functions for expected input or `Parse` when the kind or interpretation may be ambiguous
- **Explicit human context**: Supply language, reference time, and timezone when interpreting natural-language input
- **Safe calendar arithmetic**: Keep exact `Duration` math separate from calendar `Period` math, including end-of-month and DST behavior
- **Timezone-aware resolution**: Preserve nonexistent and duplicate local times instead of silently normalizing them
- **Stable JSON**: Serialize value objects and parse results with deterministic wire representations
- **Stdlib interoperability**: Hand values to formatters and other Go APIs through `time.Time` and `time.Duration`

## Installation

```bash
go get github.com/agentable/go-time
```

Requires **Go 1.27.0 or newer**.

## Quick Start

Use a typed parser when your application expects a specific value. This example
parses a local deadline in a configured IANA timezone, then inspects its absolute
instant and local projection.

```go
package main

import (
	"fmt"
	"log"

	gotime "github.com/agentable/go-time"
)

func main() {
	zone, err := gotime.LoadZone("America/New_York")
	if err != nil {
		log.Fatal(err)
	}

	deadline, err := gotime.ParseDateTime(
		"2026-11-05T09:30:00",
		gotime.WithZone(zone),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(deadline.Instant())
	fmt.Println(deadline.Std().Format("2006-01-02 15:04 MST"))
}
```

Output:

```text
2026-11-05T14:30:00Z
2026-11-05 09:30 EST
```

`ParseDateTime` resolves the local wall time in the supplied zone. Use
`Instant()` for storage and ordering, and use `Std()` when handing the value to
stdlib formatters or other Go APIs. Parsing does not persist the deadline or
trigger any action.

## Parse User Input

### Choose a typed parser

Typed parsers return `(value, error)` and are the default choice when the input field already has a known meaning.

| Expected value | Parser | Example input |
|---|---|---|
| Absolute instant | `ParseInstant` | `2026-11-05T14:30:00Z` |
| Zoned local datetime | `ParseDateTime` | `2026-11-05T09:30:00` with `WithZone` |
| Floating local datetime | `ParseLocalDateTime` | `2026-11-05T09:30:00` |
| Calendar date | `ParseDate` | `2026-11-05` |
| Clock time | `ParseTime` | `09:30` |
| Exact duration | `ParseDuration` | `PT1H30M` |
| Calendar period | `ParsePeriod` | `P1M` |
| Absolute interval | `ParseInterval` | `2026-11-05T14:30:00Z/2026-11-05T16:00:00Z` |

Each parser rejects a different resolved kind. For example, `ParseInstant("2026-11-05")` returns an error that matches `ErrIncompatibleTypes`.

### Parse natural language deterministically

Relative date and datetime phrases need three caller-owned inputs: a language, a reference instant, and the timezone whose calendar defines the phrase. Pin the reference in tests; use `gotime.Now()` at the product boundary when runtime behavior should follow the current clock.

```go
package main

import (
	"fmt"
	"log"

	gotime "github.com/agentable/go-time"
	"golang.org/x/text/language"
)

func main() {
	zone, err := gotime.LoadZone("America/New_York")
	if err != nil {
		log.Fatal(err)
	}

	reference, err := gotime.ParseInstant("2026-03-30T12:00:00Z")
	if err != nil {
		log.Fatal(err)
	}

	deadline, err := gotime.ParseDateTime(
		"tomorrow at 3pm",
		gotime.WithInputLocale(language.English),
		gotime.WithReference(reference),
		gotime.WithZone(zone),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(deadline.Std().Format("2006-01-02 15:04 MST"))
}
```

Output:

```text
2026-03-31 15:00 EDT
```

`WithInputLocale` accepts `language.Tag`. See the [parsing specification](SPECS/20-parsing.md) for supported language families and input forms.

### Supply parsing context

Add only the context required by the input your application accepts.

| Option | Use it for |
|---|---|
| `WithInputLocale(tag)` | Enable controlled natural-language parsing or specify a known slash-date convention |
| `WithReference(instant)` | Anchor relative expressions such as `tomorrow` and `next Friday` |
| `WithZone(zone)` | Resolve local datetimes and define the calendar used by relative expressions |

Keep these values caller-owned: load the zone from application configuration, pass `gotime.Now()` at the runtime boundary, and pin a fixed reference in tests.

### Preserve ambiguous input

Use `Parse` when the input kind is unknown or the application needs candidates, warnings, or explicit-zone metadata. This is the path for user-facing ambiguity workflows.

```go
result := gotime.Parse("04/05/2026")

switch result.Status {
case gotime.StatusResolved:
	date, ok := result.Date()
	if ok {
		fmt.Println("resolved:", date)
	}
case gotime.StatusAmbiguous:
	for _, candidate := range result.Candidates {
		date, ok := candidate.Date()
		if ok {
			fmt.Println("candidate:", date)
		}
	}
case gotime.StatusInvalid:
	fmt.Println(result.Error.Code, result.Error.Hint)
}
```

Output:

```text
candidate: 2026-04-05
candidate: 2026-05-04
```

If the application already knows the user's input convention, pass it explicitly:

```go
date, err := gotime.ParseDate(
	"04/05/2026",
	gotime.WithInputLocale(language.MustParse("en-GB")),
)
if err != nil {
	return err
}
fmt.Println(date) // 2026-05-04
```

## Compute With Time Values

### Choose exact or calendar arithmetic

Use `Duration` for elapsed time and `Period` for calendar movement. Across a DST boundary, those operations can produce different wall-clock results by design.

```go
zone, err := gotime.LoadZone("America/New_York")
if err != nil {
	return err
}

start, err := gotime.ParseDateTime(
	"2026-03-07T09:00:00",
	gotime.WithZone(zone),
)
if err != nil {
	return err
}

exact, err := start.Add(24 * gotime.Hour)
if err != nil {
	return err
}

resolution, err := start.AddPeriod(gotime.Days(1))
if err != nil {
	return err
}
calendar, err := resolution.Only()
if err != nil {
	return err
}

fmt.Println(exact.Std().Format("2006-01-02 15:04 MST"))
fmt.Println(calendar.Std().Format("2006-01-02 15:04 MST"))
// 2026-03-08 10:00 EDT
// 2026-03-08 09:00 EDT
```

Typed constants cover exact units from `Nanosecond` through `Hour`. Calendar periods use `Period` fields or constructors such as `Years`, `Months`, and `Days`:

```go
retryDelay := 15 * gotime.Minute
billingStep := gotime.Period{Months: 1}
nextWeek := gotime.Days(7)
```

### Measure exact and calendar differences

Difference APIs return errors when the scalar or endpoint cannot represent the
requested result; they never return a saturated duration or normalize an
invalid date.

```go
opened, err := gotime.ParseInstant("2026-03-27T09:00:00Z")
if err != nil {
	return err
}
closed, err := gotime.ParseInstant("2026-03-27T10:30:00Z")
if err != nil {
	return err
}
elapsed, err := closed.Sub(opened)
if err != nil {
	return err
}
fmt.Println(elapsed) // 1h30m0s

first, err := gotime.ParseDate("2026-03-27")
if err != nil {
	return err
}
last, err := gotime.ParseDate("2026-04-02")
if err != nil {
	return err
}
days, err := first.DaysUntil(last)
if err != nil {
	return err
}
fmt.Println(days) // 6
```

Use exact `Sub` methods for timeline elapsed time and `DaysUntil` for signed
calendar-day counts. Product-specific age, billing, or Y/M/D decomposition
belongs in the caller.

### Query calendar facts

Calendar-derived facts report invalid `Date` values instead of normalizing
them into another date.

```go
date, err := gotime.ParseDate("2026-02-14")
if err != nil {
	return err
}

weekday, err := date.Weekday()
if err != nil {
	return err
}
days, err := date.DaysInMonth()
if err != nil {
	return err
}

fmt.Println(weekday, days) // Saturday 28
```

`ISOWeek` and `YearDay` use the same checked pattern. `DaysInMonth` needs a
valid year and month only, while `IsLeapYear` remains a year-only `bool` query.

### Build a half-open time window

Create intervals from two instants or from one instant plus an exact duration. Intervals are useful for comparisons and handoff to range-based APIs.

```go
window, err := gotime.NewIntervalStartingAt(
	deadline.Instant(),
	90 * gotime.Minute,
)
if err != nil {
	return err
}

length, err := window.Length()
if err != nil {
	return err
}

fmt.Println(length)                        // 1h30m0s
fmt.Println(window.Contains(window.End())) // false: the end is exclusive

start, end := window.StdRange()
fmt.Println(start, end)
```

Use `Overlaps`, `Adjacent`, `Intersect`, `Union`, `Shift`, and `Expand` for common interval operations.

## Work With Time Zones

Choose zone loading based on where the identifier came from.

| Input source | API | Example |
|---|---|---|
| Configuration or stored IANA ID | `LoadZone` | `America/New_York` |
| Source-code constant | `MustLoadZone` | `var tokyo = gotime.MustLoadZone("Asia/Tokyo")` |
| User or migration input | `ResolveZone` | `Eastern Standard Time`, `asia/tokyo` |

Never call `MustLoadZone` with user input. Resolve user-provided values before passing the resulting `Zone` to a parser:

```go
zone, err := gotime.ResolveZone(userZone)
if err != nil {
	return err
}

deadline, err := gotime.ParseDateTime(userTime, gotime.WithZone(zone))
if err != nil {
	return err
}
fmt.Println(deadline.Instant())
```

`gotime.UTC` is a named value for reading, comparison, and arguments; assigning
to it does not configure a process-wide default. `Zones()` returns a sorted,
caller-owned copy, so changing the returned slice does not affect later calls.

Use `LocalDateTime.Resolve` when a local wall time may fall in a DST gap or overlap and the application needs to inspect that state directly:

```go
zone, err := gotime.LoadZone("America/New_York")
if err != nil {
	return err
}

local, err := gotime.ParseLocalDateTime("2026-11-01T01:30:00")
if err != nil {
	return err
}

resolution := local.Resolve(zone)
switch resolution.Status {
case gotime.LocalResolved:
	datetime, err := resolution.Only()
	if err != nil {
		return err
	}
	fmt.Println(datetime.Instant())
case gotime.LocalAmbiguous:
	for _, candidate := range resolution.Candidates {
		fmt.Println(candidate.Instant())
	}
case gotime.LocalNonexistent:
	// Ask for another local time.
case gotime.LocalInvalid:
	// Reject invalid date or clock components.
}
```

Output for this duplicate local time:

```text
2026-11-01T05:30:00Z
2026-11-01T06:30:00Z
```

The application chooses between these candidates; go-time does not select one implicitly.

## Hand Values to Other APIs

go-time does not format values for display. Convert semantic values at the integration boundary:

| Value | Bridge |
|---|---|
| `Instant` | `instant.Std()` returns UTC `time.Time` |
| `Instant` | `instant.UnixNano()` / `instant.UnixMilli()` return a checked epoch scalar |
| `DateTime` | `datetime.Std()` returns zoned `time.Time` |
| `Duration` | `duration.Std()` returns `time.Duration` |
| `Duration` | `duration.Decompose()` returns clock component slots |
| `Interval` | `interval.StdRange()` returns two `time.Time` values |
| `Period` | Read the exported `Years`, `Months`, and `Days` fields |

```go
fmt.Println(deadline.Std().Format(time.RFC1123))

epochMillis, err := deadline.Instant().UnixMilli()
if err != nil {
	return err
}
fmt.Println(epochMillis)

components := (90 * gotime.Minute).Decompose()
fmt.Println(components.Hours, components.Minutes) // 1 30
```

Epoch projections return `ErrOverflow` when the selected `int64` precision
cannot represent the instant. Keep the `Instant` or use `Std()` when a scalar
projection is not required.

## Serialize Values

Marshal value objects and parse results with Go 1.27.0 `encoding/json/v2`.

```go
payload, err := json.Marshal(deadline)
if err != nil {
	return err
}
fmt.Println(string(payload))
// {"kind":"datetime","instant":"2026-11-05T14:30:00Z","zone":"America/New_York"}

var restored gotime.DateTime
if err := json.Unmarshal(payload, &restored); err != nil {
	return err
}
fmt.Println(restored.Zone().ID()) // America/New_York
```

Decoding validates the value at the boundary. See [SPECS/10-domain-model.md](SPECS/10-domain-model.md) for the complete wire contract instead of mirroring schemas in application code.

`ParseResult` and `TimeError` JSON are diagnostic output, not runtime recovery
formats. Marshal them when sending inspection data to a log pipeline or API:

```go
result := gotime.Parse(userInput, opts...)
diagnostic, err := json.Marshal(result)
if err != nil {
	return err
}
sendDiagnostic(diagnostic)
```

Keep the original result when code needs `HasZone`, typed accessors, ambiguity
identity, or `errors.Is`. To obtain a new runtime result later, parse the
original input again with explicit options. A serialized `TimeError` keeps its
stable `code` and details, but not the Go sentinel or underlying cause chain.

## Share Values Safely

Value methods do not modify their receiver, so independent values can be read
concurrently. JSON decoding is different: it replaces the pointer receiver.
Keep that receiver private until decoding finishes.

```go
var date gotime.Date
if err := json.Unmarshal(payload, &date); err != nil {
	return err
}
publish(date)
```

`ParseResult.Warnings`, `ParseResult.Candidates`, and
`LocalResolution.Candidates` are caller-owned slices. Copying the containing
struct still shares each slice backing array. Clone the slices you need to
mutate independently, and coordinate concurrent access whenever an alias may
write.

## Handle Errors

Use sentinel errors for control flow and `*TimeError` for structured details.

```go
_, err := gotime.ParseDateTime(userTime, gotime.WithZone(zone))
if err != nil {
	if errors.Is(err, gotime.ErrDuplicateTime) {
		// Re-run with Parse to inspect candidates or ask the user to clarify.
	}

	var timeErr *gotime.TimeError
	if errors.As(err, &timeErr) {
		log.Printf("code=%s hint=%s", timeErr.Code, timeErr.Hint)
	}
	return err
}
```

Treat `TimeError.Input` and `TimeError.Message` as caller-provided data and apply the application's redaction policy before logging them.

## API Overview

| Task | Primary API |
|---|---|
| Read the current instant or date | `Now`, `NowIn`, `TodayIn` |
| Parse a known type | `ParseInstant`, `ParseDateTime`, `ParseLocalDateTime`, `ParseDate`, `ParseTime`, `ParseDuration`, `ParsePeriod`, `ParseInterval` |
| Parse unknown or ambiguous input | `Parse`, `ParseResult` |
| Construct civil values | `NewDate`, `NewTime`, `NewLocalDateTime`, `NewDateTime` |
| Convert from Unix or stdlib values | `UnixSeconds`, `UnixMillis`, `UnixNanos`, `InstantFromTime`, `DateTimeFromTime`, `DateFromTime`, `TimeFromTime` |
| Project an instant to Unix time | checked `Instant.UnixNano`, `Instant.UnixMilli` |
| Compare and calculate | `Compare`, `Add`, `AddPeriod`, `Sub`, `DaysUntil`, checked date queries |
| Work with ranges | `NewInterval`, `NewIntervalStartingAt`, `NewIntervalEndingAt`, interval methods |
| Load time zones | `LoadZone`, `MustLoadZone`, `ResolveZone`, `Zones`, `ZoneCatalogVersion` |

For complete signatures and package documentation, use [pkg.go.dev](https://pkg.go.dev/github.com/agentable/go-time). For behavior contracts, use [SPECS/](SPECS/).

## Development

```bash
task test      # Run all tests with race detection
task lint      # Run golangci-lint and verify module tidiness
task fmt       # Format Go code
task vet       # Run go vet
task verify    # Run the full verification suite
```

For development guidelines and agent workflow, see [AGENTS.md](AGENTS.md).

## Contributing

Open an issue before large changes so the public API and specifications can stay aligned.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
