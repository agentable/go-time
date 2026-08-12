package gotime

import (
	"bytes"
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
	got, err := resolution.Only()
	if !got.IsZero() {
		t.Errorf("Only() = %v, want zero DateTime", got)
	}
	if !errors.Is(err, ErrNonexistentTime) {
		t.Fatalf("Only() error = %v, want ErrNonexistentTime", err)
	}
	var timeErr *TimeError
	if !errors.As(err, &timeErr) || timeErr.Hint == "" {
		t.Fatalf("Only() error = %#v, want TimeError with hint", err)
	}
}

func TestLocalDateTimeResolveSkippedDate(t *testing.T) {
	d := mustDate(2011, time.December, 30)
	clock := mustTime(12, 0, 0)
	z := MustLoadZone("Pacific/Apia")

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
	got, err := resolution.Only()
	if !got.IsZero() {
		t.Errorf("Only() = %v, want zero DateTime", got)
	}
	if !errors.Is(err, ErrDuplicateTime) {
		t.Fatalf("Only() error = %v, want ErrDuplicateTime", err)
	}
	var timeErr *TimeError
	if !errors.As(err, &timeErr) || timeErr.Hint == "" {
		t.Fatalf("Only() error = %#v, want TimeError with hint", err)
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
	_, firstOffset := first.Instant().Std().In(first.Zone().Location()).Zone()
	_, secondOffset := second.Instant().Std().In(second.Zone().Location()).Zone()
	if firstOffset == secondOffset {
		t.Fatalf("candidate offsets both = %d, want distinct offsets", firstOffset)
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

func TestLocalResolutionOnlyRejectsInvalidPublicStates(t *testing.T) {
	validLocal := NewLocalDateTime(
		mustDate(2026, time.March, 27),
		mustTime(9, 30, 0),
	)
	candidate := mustDateTime(validLocal.Date, validLocal.Time, UTC)

	tests := []struct {
		name       string
		resolution LocalResolution
		want       error
		wantCode   ErrorCode
	}{
		{
			name:       "resolved without candidate",
			resolution: LocalResolution{Status: LocalResolved, Local: validLocal},
			want:       ErrInvalidTime,
			wantCode:   CodeInvalidTime,
		},
		{
			name: "resolved with multiple candidates",
			resolution: LocalResolution{
				Status:     LocalResolved,
				Local:      validLocal,
				Candidates: []DateTime{candidate, candidate},
			},
			want:     ErrInvalidTime,
			wantCode: CodeInvalidTime,
		},
		{
			name: "invalid date components",
			resolution: LocalResolution{
				Status: LocalInvalid,
				Local:  NewLocalDateTime(Date{}, validLocal.Time),
			},
			want:     ErrInvalidDate,
			wantCode: CodeInvalidDate,
		},
		{
			name: "invalid time components",
			resolution: LocalResolution{
				Status: LocalInvalid,
				Local:  NewLocalDateTime(validLocal.Date, Time{hour: 24}),
			},
			want:     ErrInvalidTime,
			wantCode: CodeInvalidTime,
		},
		{
			name:       "invalid status with valid local time",
			resolution: LocalResolution{Status: LocalInvalid, Local: validLocal},
			want:       ErrInvalidTime,
			wantCode:   CodeInvalidTime,
		},
		{
			name:       "unknown status",
			resolution: LocalResolution{Status: LocalResolutionStatus("unknown"), Local: validLocal},
			want:       ErrInvalidTime,
			wantCode:   CodeInvalidTime,
		},
		{
			name:       "zero value",
			resolution: LocalResolution{},
			want:       ErrInvalidTime,
			wantCode:   CodeInvalidTime,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.resolution.Only()
			if !got.IsZero() {
				t.Errorf("LocalResolution.Only() = %v, want zero DateTime", got)
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("LocalResolution.Only() error = %v, want %v", err, tc.want)
			}
			var timeErr *TimeError
			if !errors.As(err, &timeErr) || timeErr.Hint == "" {
				t.Fatalf("LocalResolution.Only() error = %#v, want TimeError with hint", err)
			}
			if timeErr.Code != tc.wantCode {
				t.Errorf("LocalResolution.Only() error code = %q, want %q", timeErr.Code, tc.wantCode)
			}
		})
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
	const want = `{"kind":"local_datetime","value":"2026-03-27T09:30:00"}`
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
	again, err := json.Marshal(got)
	if err != nil || !bytes.Equal(again, b) {
		t.Fatalf("second Marshal(%v) = %s, %v; want %s", got, again, err, b)
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
	const want = `{"kind":"local_datetime","value":"2026-03-27T09:30:00.12345"}`
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
	again, err := json.Marshal(got)
	if err != nil || !bytes.Equal(again, b) {
		t.Fatalf("second Marshal(%v) = %s, %v; want %s", got, again, err, b)
	}
}

func TestLocalDateTimeJSONRoundTripCivilBoundaries(t *testing.T) {
	for _, ldt := range []LocalDateTime{
		NewLocalDateTime(mustDate(0, time.January, 1), mustTimeNanos(0, 0, 0, 1)),
		NewLocalDateTime(mustDate(9999, time.December, 31), mustTimeNanos(23, 59, 59, 999_999_999)),
	} {
		b, err := json.Marshal(ldt)
		if err != nil {
			t.Fatalf("Marshal(%v) error = %v", ldt, err)
		}
		var got LocalDateTime
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("Unmarshal(%s) error = %v", b, err)
		}
		again, err := json.Marshal(got)
		if err != nil || !bytes.Equal(again, b) {
			t.Fatalf("second Marshal(%v) = %s, %v; want %s", got, again, err, b)
		}
	}
}

func TestLocalDateTimeMarshalJSON_RejectsInvalidComponents(t *testing.T) {
	tests := []struct {
		name string
		ldt  LocalDateTime
		want error
	}{
		{
			name: "invalid date",
			ldt:  NewLocalDateTime(Date{}, mustTime(9, 30, 0)),
			want: ErrInvalidDate,
		},
		{
			name: "invalid time",
			ldt:  NewLocalDateTime(mustDate(2026, time.March, 27), Time{hour: 24}),
			want: ErrInvalidTime,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := json.Marshal(tc.ldt)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Marshal(%v) error = %v, want %v", tc.ldt, err, tc.want)
			}
			var te *TimeError
			if !errors.As(err, &te) || te.Hint == "" {
				t.Fatalf("Marshal(%v) error = %#v, want TimeError with hint", tc.ldt, err)
			}
		})
	}
}
