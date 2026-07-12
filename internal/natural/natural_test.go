package natural

import (
	"testing"
	"time"
)

func mustLoadLocation(t *testing.T, id string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(id)
	if err != nil {
		t.Fatalf("time.LoadLocation(%q): %v", id, err)
	}
	return loc
}

func equalCivil(r Result, want time.Time) bool {
	return r.Year == want.Year() &&
		r.Month == want.Month() &&
		r.Day == want.Day() &&
		r.Hour == want.Hour() &&
		r.Minute == want.Minute() &&
		r.Second == want.Second() &&
		r.Nanosecond == want.Nanosecond()
}

func civilTime(r Result) time.Time {
	return time.Date(r.Year, r.Month, r.Day, r.Hour, r.Minute, r.Second, r.Nanosecond, time.UTC)
}
