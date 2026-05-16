package gotime

import "time"

// Now returns the current moment as an Instant (UTC, no monotonic reading).
func Now() Instant { return Instant{t: time.Now().UTC()} }

// NowIn returns the current moment projected into zone z as a DateTime.
func NowIn(z Zone) DateTime { return Now().In(z) }

// TodayIn returns the current calendar date in zone z.
//
// There is no zero-arg Today() helper: a calendar date is only meaningful
// relative to a zone, and inheriting time.Local from the runtime environment
// produces date drift across container/CI/server boundaries. Callers that
// want process-local behavior must spell it out: TodayIn(Local).
func TodayIn(z Zone) Date { return DateFromTime(time.Now().In(z.Location())) }
