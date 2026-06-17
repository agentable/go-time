// Package zone provides internal IANA timezone data and DST projection utilities.
package zone

import (
	"slices"
	"time"
)

// DSTStatus describes the DST classification of a projected local time.
type DSTStatus int

const (
	// DSTNormal means the local time maps to exactly one UTC instant.
	DSTNormal DSTStatus = iota
	// DSTNonexistent means the local time falls in a spring-forward gap (no such wall time exists).
	DSTNonexistent
	// DSTAmbiguous means the local time falls in a fall-back overlap (two UTC instants map to it).
	DSTAmbiguous
)

// LocalTimeResult holds the outcome of projecting a local wall-clock time into a timezone.
type LocalTimeResult struct {
	// Status classifies the local time as normal, nonexistent, or ambiguous.
	Status DSTStatus
	// Times holds the matching instants in chronological order.
	Times []time.Time
}

// ProjectLocalTime projects a local wall-clock time into loc and classifies the DST status.
//
// Algorithm:
//  1. Construct t = time.Date(year, month, day, hour, minute, second, 0, loc).
//  2. Gap detection: if t's actual local components differ from requested, the wall time
//     does not exist. Compare the date too, because some zones skip whole calendar days.
//  3. Overlap detection: collect nearby UTC offsets and reproject the same wall time under
//     each offset. This catches non-hour transitions such as Australia/Lord_Howe's 30-minute
//     fall-back overlap without assuming every DST boundary is exactly one hour.
//  4. Otherwise → DSTNormal with a single instant.
func ProjectLocalTime(loc *time.Location, year int, month time.Month, day, hour, minute, second int) LocalTimeResult {
	if loc == nil {
		loc = time.UTC
	}

	t := time.Date(year, month, day, hour, minute, second, 0, loc)
	if !sameLocalTime(t, year, month, day, hour, minute, second) {
		return LocalTimeResult{Status: DSTNonexistent}
	}

	candidates := []time.Time{t}
	_, off0 := t.Zone()
	utc := t.UTC()
	for _, delta := range []time.Duration{
		-24 * time.Hour, -12 * time.Hour, -6 * time.Hour,
		-3 * time.Hour, -2 * time.Hour, -time.Hour,
		-30 * time.Minute, -15 * time.Minute,
		15 * time.Minute, 30 * time.Minute,
		time.Hour, 2 * time.Hour, 3 * time.Hour,
		6 * time.Hour, 12 * time.Hour, 24 * time.Hour,
	} {
		probe := utc.Add(delta).In(loc)
		_, off := probe.Zone()
		if off == off0 {
			continue
		}
		alt := utc.Add(time.Duration(off0-off) * time.Second).In(loc)
		if sameLocalTime(alt, year, month, day, hour, minute, second) {
			candidates = appendUniqueTime(candidates, alt)
		}
	}

	if len(candidates) > 1 {
		slices.SortFunc(candidates, func(a, b time.Time) int { return a.Compare(b) })
		return LocalTimeResult{
			Status: DSTAmbiguous,
			Times:  candidates,
		}
	}

	return LocalTimeResult{
		Status: DSTNormal,
		Times:  candidates,
	}
}

func sameLocalTime(t time.Time, year int, month time.Month, day, hour, minute, second int) bool {
	return t.Year() == year &&
		t.Month() == month &&
		t.Day() == day &&
		t.Hour() == hour &&
		t.Minute() == minute &&
		t.Second() == second
}

func appendUniqueTime(times []time.Time, t time.Time) []time.Time {
	for _, existing := range times {
		if existing.Equal(t) {
			return times
		}
	}
	return append(times, t)
}
