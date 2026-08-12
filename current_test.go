package gotime

import (
	"testing"
	"time"
)

func TestNow_NonZero(t *testing.T) {
	before := time.Now()
	i := Now()
	after := time.Now()
	if i.IsZero() {
		t.Error("Now() returned zero Instant")
	}
	if got := i.Std(); got.Before(before) || got.After(after) {
		t.Errorf("Now() = %v, want within [%v, %v]", got, before, after)
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
	before := time.Now().In(z.Location())
	got := TodayIn(z)
	after := time.Now().In(z.Location())
	beforeDate, err := DateFromTime(before)
	if err != nil {
		t.Fatalf("DateFromTime(%v) error = %v", before, err)
	}
	afterDate, err := DateFromTime(after)
	if err != nil {
		t.Fatalf("DateFromTime(%v) error = %v", after, err)
	}
	if !got.Equal(beforeDate) && !got.Equal(afterDate) {
		t.Errorf("TodayIn(America/New_York) = %v, want %v or %v", got, beforeDate, afterDate)
	}
}

func TestNowInAndTodayIn_ZeroZoneUseUTC(t *testing.T) {
	beforeNow := time.Now().UTC()
	datetime := NowIn(Zone{})
	afterNow := time.Now().UTC()

	beforeToday := time.Now().UTC()
	today := TodayIn(Zone{})
	afterToday := time.Now().UTC()

	if got := datetime.Zone().ID(); got != "UTC" {
		t.Errorf("NowIn(Zone{}).Zone().ID() = %q, want UTC", got)
	}
	if got := datetime.Std(); got.Before(beforeNow) || got.After(afterNow) {
		t.Errorf("NowIn(Zone{}) = %v, want within [%v, %v]", got, beforeNow, afterNow)
	}
	beforeDate, err := DateFromTime(beforeToday)
	if err != nil {
		t.Fatalf("DateFromTime(%v) error = %v", beforeToday, err)
	}
	afterDate, err := DateFromTime(afterToday)
	if err != nil {
		t.Fatalf("DateFromTime(%v) error = %v", afterToday, err)
	}
	if !today.Equal(beforeDate) && !today.Equal(afterDate) {
		t.Errorf("TodayIn(Zone{}) = %v, want %v or %v", today, beforeDate, afterDate)
	}
}
