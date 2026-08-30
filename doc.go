// Package gotime converts human time expressions into precise, computable
// value objects.
//
// # Parse entry points
//
// Two layers, picked by what you know up front:
//
//   - Typed helpers — [ParseInstant], [ParseDateTime], [ParseLocalDateTime],
//     [ParseDate], [ParseTime], [ParseDuration], [ParsePeriod],
//     [ParseInterval]. Each returns (T, error) and is the default path for
//     application code.
//   - [Parse] — diagnostic / dispatch entry. Returns a [ParseResult] with
//     Status / Kind / Candidates / Warnings. Reach for it when you do not
//     know the kind ahead of time, or when you need ambiguity candidates,
//     warnings, or zone metadata. Dispatch through Status, Kind, and the
//     comma-ok ParseResult accessors.
//
// # Current time
//
// [Now] / [NowIn] / [TodayIn] return the current moment or calendar date.
// There is no zero-argument Today: a calendar date requires an explicit zone.
//
// # Panic convention
//
// [MustLoadZone] panics only for invalid fixed IANA zone identifiers and is
// intended for package-level or test-time initialization.
//
// # Ownership and concurrency
//
// Value-receiver methods do not mutate their receiver. Pointer JSON decoders
// replace their target and require caller-provided exclusive access while
// decoding. Exported result slices are caller-owned; copying a containing
// struct aliases the slice backing arrays rather than deep-cloning them.
//
// # Formatting is out of scope
//
// Localization and display formatting are intentionally out of scope. Bridge
// gotime values to an external renderer through the [Instant.Std],
// [DateTime.Std], and [Duration.Std] adapters.
//
// # Example — typed helper (most common)
//
//	dt, err := gotime.ParseDateTime("2026-03-27T13:00:00",
//	    gotime.WithZone(gotime.MustLoadZone("Asia/Tokyo")),
//	)
//	if err != nil {
//	    return err
//	}
//	fmt.Println(dt.Std().Format("2006-01-02 15:04 MST"))
//
// # Example — natural language with locale
//
// Natural date/datetime expressions require an explicit reference instant.
// Product code can pass [Now] at the boundary; tests usually pass a fixed
// [Instant].
//
//	dt, err := gotime.ParseDateTime("tomorrow at 3pm",
//	    gotime.WithInputLocale(language.English),
//	    gotime.WithZone(gotime.MustLoadZone("America/New_York")),
//	    gotime.WithReference(gotime.Now()),
//	)
package gotime
