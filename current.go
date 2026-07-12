package gotime

import "time"

// Now returns the current moment as an Instant (UTC, no monotonic reading).
func Now() Instant { return Instant{t: time.Now().UTC()} }

// NowIn returns the current moment projected into zone z as a DateTime.
func NowIn(z Zone) DateTime { return dateTimeFromTimeTrusted(time.Now(), z) }

// TodayIn returns the current calendar date in zone z.
//
// There is no zero-arg Today() helper: a calendar date is only meaningful
// relative to a zone. Applications that need the host's zone must discover or
// configure its IANA identifier and pass the resulting Zone explicitly.
func TodayIn(z Zone) Date { return dateFromTimeTrusted(time.Now().In(z.Location())) }
