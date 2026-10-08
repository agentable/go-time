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
	// DSTAmbiguous means the local time falls in a fall-back overlap (multiple UTC instants map to it).
	DSTAmbiguous
)

// LocalTimeResult holds the outcome of projecting a local wall-clock time into a timezone.
type LocalTimeResult struct {
	// Status classifies the local time as normal, nonexistent, or ambiguous.
	Status DSTStatus
	// Times holds the matching instants in chronological order.
	Times []time.Time
}

// ProjectLocalTime enumerates every possible offset from the held snapshot.
// For civil time L, every solution has the form L-offset. Forward projection
// rejects offsets that are not active there, including historical and footer
// offsets. No transition spacing or maximum candidate count is assumed.
func ProjectLocalTime(rules *Rules, year int, month time.Month, day, hour, minute, second int) LocalTimeResult {
	loc := rules.Location()
	offsets := []int{0}
	if rules != nil {
		offsets = rules.offsets
	}
	civil := time.Date(year, month, day, hour, minute, second, 0, time.UTC)
	var candidates []time.Time
	for _, offset := range offsets {
		candidate := civil.Add(-time.Duration(offset) * time.Second).In(loc)
		if sameLocalTime(candidate, year, month, day, hour, minute, second) {
			candidates = append(candidates, candidate)
		}
	}
	slices.SortFunc(candidates, time.Time.Compare)
	status := DSTNormal
	switch len(candidates) {
	case 0:
		status = DSTNonexistent
	case 1:
	default:
		status = DSTAmbiguous
	}
	return LocalTimeResult{Status: status, Times: candidates}
}

func sameLocalTime(t time.Time, year int, month time.Month, day, hour, minute, second int) bool {
	return t.Year() == year &&
		t.Month() == month &&
		t.Day() == day &&
		t.Hour() == hour &&
		t.Minute() == minute &&
		t.Second() == second
}
