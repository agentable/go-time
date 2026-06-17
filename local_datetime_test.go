package gotime

import (
	"errors"
	"testing"
	"time"

	"github.com/go-json-experiment/json"
)

func TestLocalDateTimeResolveNormal(t *testing.T) {
	d := mustDate(2026, time.March, 27)
	clock := mustTime(9, 30, 0)
	z := MustLoadZone("Asia/Shanghai")

	ldt := NewLocalDateTime(d, clock)
	resolution := ldt.Resolve(z)

	if resolution.Status != LocalResolved {
		t.Fatalf("Resolve status = %s, want %s", resolution.Status, LocalResolved)
	}
	if !resolution.Zone.Equal(z) {
		t.Fatalf("Resolve zone = %s, want %s", resolution.Zone, z)
	}
	if !resolution.Local.Date.Equal(d) || !resolution.Local.Time.Equal(clock) {
		t.Fatalf("Resolve local = %v, want date %v time %v", resolution.Local, d, clock)
	}
	if got := len(resolution.Candidates); got != 1 {
		t.Fatalf("Resolve candidates = %d, want 1", got)
	}

	dt, err := resolution.Only()
	if err != nil {
		t.Fatalf("Only() error = %v, want nil", err)
	}
	if got := dt.String(); got != "2026-03-27T09:30:00+08:00" {
		t.Fatalf("Only() = %s, want 2026-03-27T09:30:00+08:00", got)
	}
}

func TestLocalDateTimeResolveNonexistent(t *testing.T) {
	d := mustDate(2026, time.March, 8)
	clock := mustTime(2, 30, 0)
	z := MustLoadZone("America/New_York")

	resolution := NewLocalDateTime(d, clock).Resolve(z)

	if resolution.Status != LocalNonexistent {
		t.Fatalf("Resolve status = %s, want %s", resolution.Status, LocalNonexistent)
	}
	if got := len(resolution.Candidates); got != 0 {
		t.Fatalf("Resolve candidates = %d, want 0", got)
	}
	if _, err := resolution.Only(); !errors.Is(err, ErrNonexistentTime) {
		t.Fatalf("Only() error = %v, want ErrNonexistentTime", err)
	}
}

func TestLocalDateTimeResolveAmbiguous(t *testing.T) {
	d := mustDate(2026, time.November, 1)
	clock := mustTime(1, 30, 0)
	z := MustLoadZone("America/New_York")

	resolution := NewLocalDateTime(d, clock).Resolve(z)

	if resolution.Status != LocalAmbiguous {
		t.Fatalf("Resolve status = %s, want %s", resolution.Status, LocalAmbiguous)
	}
	if got := len(resolution.Candidates); got != 2 {
		t.Fatalf("Resolve candidates = %d, want 2", got)
	}
	if _, err := resolution.Only(); !errors.Is(err, ErrDuplicateTime) {
		t.Fatalf("Only() error = %v, want ErrDuplicateTime", err)
	}

	first, second := resolution.Candidates[0], resolution.Candidates[1]
	if !first.Before(second) {
		t.Fatalf("candidates are not chronological: %s then %s", first, second)
	}
	if !first.Date().Equal(d) || !second.Date().Equal(d) {
		t.Fatalf("candidate dates = %s and %s, want %s", first.Date(), second.Date(), d)
	}
	if !first.Clock().Equal(clock) || !second.Clock().Equal(clock) {
		t.Fatalf("candidate clocks = %s and %s, want %s", first.Clock(), second.Clock(), clock)
	}
	if first.Zone().OffsetAt(first.Instant()) == second.Zone().OffsetAt(second.Instant()) {
		t.Fatalf("candidate offsets both = %s, want distinct offsets", first.Zone().OffsetAt(first.Instant()))
	}
}

func TestLocalDateTimeResolveZeroZoneUsesUTC(t *testing.T) {
	d := mustDate(2026, time.March, 27)
	clock := mustTime(9, 30, 0)

	resolution := NewLocalDateTime(d, clock).Resolve(Zone{})

	if !resolution.Zone.Equal(UTC) {
		t.Fatalf("Resolve zero zone = %s, want UTC", resolution.Zone)
	}
	dt, err := resolution.Only()
	if err != nil {
		t.Fatalf("Only() error = %v, want nil", err)
	}
	if got := dt.String(); got != "2026-03-27T09:30:00Z" {
		t.Fatalf("Only() = %s, want 2026-03-27T09:30:00Z", got)
	}
}

func TestLocalDateTimeJSONRoundTrip(t *testing.T) {
	ldt := NewLocalDateTime(
		mustDate(2026, time.March, 27),
		mustTime(9, 30, 0),
	)

	b, err := json.Marshal(ldt)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v, want nil", err)
	}
	const want = `{"kind":"local_datetime","value":"2026-03-27T09:30:00","calendar":"iso8601"}`
	if string(b) != want {
		t.Fatalf("MarshalJSON() = %s, want %s", b, want)
	}

	var got LocalDateTime
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v, want nil", err)
	}
	if !got.Date.Equal(ldt.Date) || !got.Time.Equal(ldt.Time) {
		t.Fatalf("UnmarshalJSON() = %v, want %v", got, ldt)
	}
}

func TestLocalDateTimeJSONRoundTripFraction(t *testing.T) {
	ldt := NewLocalDateTime(
		mustDate(2026, time.March, 27),
		mustTimeNanos(9, 30, 0, 123_450_000),
	)

	b, err := json.Marshal(ldt)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v, want nil", err)
	}
	const want = `{"kind":"local_datetime","value":"2026-03-27T09:30:00.12345","calendar":"iso8601"}`
	if string(b) != want {
		t.Fatalf("MarshalJSON() = %s, want %s", b, want)
	}

	var got LocalDateTime
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v, want nil", err)
	}
	if !got.Date.Equal(ldt.Date) || !got.Time.Equal(ldt.Time) {
		t.Fatalf("UnmarshalJSON() = %v, want %v", got, ldt)
	}
}
