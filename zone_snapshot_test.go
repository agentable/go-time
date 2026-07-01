package gotime

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestZone_Snapshot(t *testing.T) {
	t.Parallel()

	zone := MustLoadZone("America/New_York")
	instant := InstantFromTime(time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC))

	got := zone.Snapshot(instant)
	want := ZoneSnapshot{
		ID:           "America/New_York",
		Offset:       "-04:00",
		Abbreviation: "EDT",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Snapshot() mismatch (-want +got):\n%s", diff)
	}
}

func TestZone_ZeroSnapshot(t *testing.T) {
	t.Parallel()

	var zone Zone
	instant := InstantFromTime(time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC))

	got := zone.Snapshot(instant)
	want := ZoneSnapshot{ID: "UTC", Offset: "+00:00", Abbreviation: "UTC"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Snapshot() mismatch (-want +got):\n%s", diff)
	}
}
