package gotime

import (
	"testing"
	"time"
)

func TestNow_NonZero(t *testing.T) {
	i := Now()
	if i.IsZero() {
		t.Error("Now() returned zero Instant")
	}
	// Should be within a second of time.Now().
	delta := time.Since(i.Std())
	if delta < 0 || delta > time.Second {
		t.Errorf("Now() delta = %v, expected < 1s", delta)
	}
}

func TestNowIn_UTCZone(t *testing.T) {
	dt := NowIn(UTC)
	if dt.Zone().ID() != "UTC" {
		t.Errorf("NowIn(UTC).Zone().ID() = %q, want \"UTC\"", dt.Zone().ID())
	}
	if dt.IsZero() {
		t.Error("NowIn(UTC) returned zero DateTime")
	}
}

func TestNowIn_NonUTCZone(t *testing.T) {
	z := MustLoadZone("Asia/Tokyo")
	dt := NowIn(z)
	if dt.Zone().ID() != "Asia/Tokyo" {
		t.Errorf("NowIn(Asia/Tokyo).Zone().ID() = %q, want \"Asia/Tokyo\"", dt.Zone().ID())
	}
}

func TestTodayIn_Zone(t *testing.T) {
	z := MustLoadZone("America/New_York")
	got := TodayIn(z)
	want := DateFromTime(time.Now().In(z.Location()))
	if !got.Equal(want) {
		t.Errorf("TodayIn(America/New_York) = %v, want %v", got, want)
	}
}
