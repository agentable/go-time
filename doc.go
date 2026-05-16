// Package gotime converts human time expressions into precise, computable
// value objects.
//
// # Parse entry points
//
// Two layers, picked by what you know up front:
//
//   - Typed helpers — [ParseInstant], [ParseDateTime], [ParseDate],
//     [ParseTime], [ParseDuration], [ParsePeriod], [ParseInterval]. Each
//     returns (T, error) and is the default path for application code.
//   - [Parse] — inspection / dispatch entry. Returns a [ParseResult] with
//     Status / Kind / Candidates / Warnings. Reach for it when you do not
//     know the kind ahead of time, or when you need ambiguity candidates,
//     warnings, or zone metadata. Use [ParseResult.Value] with a Go type
//     switch for polymorphic dispatch.
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
// # Formatting is out of scope
//
// Localization and display formatting are intentionally out of scope. Bridge
// gotime values to a formatter (for example github.com/agentable/go-intl) via
// the [Instant.Std], [DateTime.Std], and [Duration.Std] adapters.
//
// # Example — typed helper (most common)
//
//	dt, err := gotime.ParseDateTime("2026-03-27T13:00:00+09:00")
//	if err != nil {
//	    return err
//	}
//	fmt.Println(dt.Std().Format("2006-01-02 15:04 MST"))
//
// # Example — natural language with locale
//
// Relative expressions ("tomorrow", "in 2 hours") default to time.Now();
// pass [WithReference] only when you need a fixed reference for testing.
//
//	dt, err := gotime.ParseDateTime("tomorrow at 3pm",
//	    gotime.WithInputLocale(language.English),
//	    gotime.WithZone(gotime.MustLoadZone("America/New_York")),
//	)
package gotime
