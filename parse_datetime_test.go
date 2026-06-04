package gotime

import (
	"testing"
	"time"
)

func TestParse_RFC3339_UTC(t *testing.T) {
	r := Parse("2026-03-27T04:00:00Z")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v, want Resolved", r.Status)
	}
	if r.Kind != KindInstant {
		t.Fatalf("kind = %v, want KindInstant", r.Kind)
	}
	i, _ := r.Instant()
	if i.IsZero() {
		t.Fatal("Instant must not be zero")
	}
	got := i.Std()
	want := time.Date(2026, 3, 27, 4, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("Instant = %v, want %v", got, want)
	}
}

func TestParse_DateTime_WithOffset(t *testing.T) {
	r := Parse("2026-03-27T13:00:00+09:00")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v, want Resolved", r.Status)
	}
	if r.Kind != KindDateTime {
		t.Fatalf("kind = %v, want KindDateTime", r.Kind)
	}
	dt, _ := r.DateTime()
	if dt.Zone().ID() == "" {
		t.Error("zone must not be empty")
	}
	// The UTC equivalent should be 04:00
	if dt.Instant().Std().Hour() != 4 {
		t.Errorf("UTC hour = %d, want 4", dt.Instant().Std().Hour())
	}
}

func TestParse_DateTime_SubSecond(t *testing.T) {
	r := Parse("2026-03-27T13:00:00.123456+09:00")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v, want Resolved", r.Status)
	}
	if r.Kind != KindDateTime {
		t.Fatalf("kind = %v, want KindDateTime", r.Kind)
	}
	dt, _ := r.DateTime()
	ns := dt.Instant().Std().Nanosecond()
	if ns != 123456000 {
		t.Errorf("nanoseconds = %d, want 123456000", ns)
	}
}

func TestParse_DateTime_CompactFormat(t *testing.T) {
	r := Parse("20260327T130000+0900")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	if r.Kind != KindDateTime {
		t.Fatalf("kind = %v, want KindDateTime", r.Kind)
	}
	dt, _ := r.DateTime()
	utc := dt.Instant().Std()
	if utc.Hour() != 4 {
		t.Errorf("UTC hour = %d, want 4", utc.Hour())
	}
}

func TestParse_DateTime_CompactLocal(t *testing.T) {
	r := Parse("20260327T130000")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	if r.Kind != KindDateTime {
		t.Fatalf("kind = %v, want KindDateTime", r.Kind)
	}
	if r.HasZone {
		t.Fatal("HasZone = true, want false")
	}
	dt, _ := r.DateTime()
	if dt.Clock().Hour() != 13 {
		t.Errorf("hour = %d, want 13", dt.Clock().Hour())
	}
}

func TestParse_DateTime_NoOffset_DefaultsUTC(t *testing.T) {
	r := Parse("2026-03-27T13:00:00")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v, want Resolved", r.Status)
	}
	if r.Kind != KindDateTime {
		t.Fatalf("kind = %v, want KindDateTime", r.Kind)
	}
	dt, _ := r.DateTime()
	if dt.Zone().ID() != "UTC" {
		t.Errorf("zone = %q, want UTC", dt.Zone().ID())
	}
}

func TestParse_DateTime_NoOffset_WithZoneOption(t *testing.T) {
	r := Parse("2026-03-27T13:00:00", WithZone(MustLoadZone("Asia/Tokyo")))
	if r.Status != StatusResolved {
		t.Fatalf("status = %v, want Resolved", r.Status)
	}
	dt, _ := r.DateTime()
	if dt.Zone().ID() != "Asia/Tokyo" {
		t.Errorf("zone = %q, want Asia/Tokyo", dt.Zone().ID())
	}
	want := time.Date(2026, time.March, 27, 4, 0, 0, 0, time.UTC)
	if got := dt.Instant().Std(); !got.Equal(want) {
		t.Errorf("instant = %v, want %v", got, want)
	}
	if !hasWarning(r.Warnings, WarnAssumedZone) {
		t.Fatalf("warnings = %v, want WarnAssumedZone", r.Warnings)
	}
}

func TestParse_DateTime_ExplicitOffsetIgnoresZoneOption(t *testing.T) {
	t.Parallel()

	r := Parse("2026-03-27T13:00:00+09:00", WithZone(MustLoadZone("America/New_York")))
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	if !r.HasZone {
		t.Fatal("HasZone = false, want true")
	}
	if !r.Zone.IsZero() {
		t.Fatalf("result Zone = %q, want zero because input carried an offset", r.Zone.ID())
	}
	if hasWarning(r.Warnings, WarnAssumedZone) {
		t.Fatalf("warnings = %v, do not want WarnAssumedZone", r.Warnings)
	}
	dt, ok := r.DateTime()
	if !ok {
		t.Fatal("DateTime() ok=false, want true")
	}
	if dt.Zone().ID() != "+09:00" {
		t.Fatalf("zone = %q, want +09:00", dt.Zone().ID())
	}
	if dt.Clock().Hour() != 13 {
		t.Fatalf("clock hour = %d, want 13", dt.Clock().Hour())
	}
	want := time.Date(2026, time.March, 27, 4, 0, 0, 0, time.UTC)
	if got := dt.Instant().Std(); !got.Equal(want) {
		t.Fatalf("instant = %v, want %v", got, want)
	}
}

func TestParse_DateTime_NoOffset_WithFixedOffsetZoneOption(t *testing.T) {
	z, err := ResolveZone("UTC+8")
	if err != nil {
		t.Fatalf("ResolveZone(UTC+8): %v", err)
	}
	r := Parse("2026-03-27T13:00:00", WithZone(z))
	if r.Status != StatusResolved {
		t.Fatalf("status = %v, want Resolved", r.Status)
	}
	dt, _ := r.DateTime()
	if dt.Zone().ID() != "+08:00" {
		t.Errorf("zone = %q, want +08:00", dt.Zone().ID())
	}
	if dt.Clock().Hour() != 13 {
		t.Errorf("hour = %d, want 13", dt.Clock().Hour())
	}
}

func TestParse_DateTime_TruncatedPrecisionWarning(t *testing.T) {
	r := Parse("2026-03-27T13:00:00.1234567899+09:00")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	if !hasWarning(r.Warnings, WarnTruncatedPrecision) {
		t.Fatalf("warnings = %v, want WarnTruncatedPrecision", r.Warnings)
	}
	dt, _ := r.DateTime()
	if got := dt.Instant().Std().Nanosecond(); got != 123456789 {
		t.Errorf("nanosecond = %d, want 123456789", got)
	}
}

func TestParse_DateTime_DuplicateTimeCandidatesWarn(t *testing.T) {
	r := Parse("2026-11-01T01:30:00", WithZone(MustLoadZone("America/New_York")))
	if r.Status != StatusAmbiguous {
		t.Fatalf("status = %v, want Ambiguous", r.Status)
	}
	if len(r.Candidates) != 2 {
		t.Fatalf("len(Candidates) = %d, want 2", len(r.Candidates))
	}
	for _, c := range r.Candidates {
		if !hasWarning(c.Warnings, WarnDuplicateTime) {
			t.Fatalf("candidate warnings = %v, want WarnDuplicateTime", c.Warnings)
		}
		if hasWarning(c.Warnings, WarnInferredCalendar) {
			t.Fatalf("candidate warnings = %v, do not want WarnInferredCalendar", c.Warnings)
		}
	}
}

func TestParse_DateTime_DuplicateTime_NonHourDSTTransition(t *testing.T) {
	r := Parse("2026-04-05T01:45:00", WithZone(MustLoadZone("Australia/Lord_Howe")))
	if r.Status != StatusAmbiguous {
		t.Fatalf("status = %v, want Ambiguous", r.Status)
	}
	if r.Kind != KindDateTime {
		t.Fatalf("kind = %v, want KindDateTime", r.Kind)
	}
	if len(r.Candidates) != 2 {
		t.Fatalf("len(Candidates) = %d, want 2", len(r.Candidates))
	}
	for _, c := range r.Candidates {
		dt, ok := c.DateTime()
		if !ok {
			t.Fatalf("candidate kind = %v, want DateTime", c.Kind)
		}
		if dt.Clock().Hour() != 1 || dt.Clock().Minute() != 45 {
			t.Fatalf("candidate clock = %v, want 01:45", dt.Clock())
		}
		if !hasWarning(c.Warnings, WarnDuplicateTime) {
			t.Fatalf("candidate warnings = %v, want WarnDuplicateTime", c.Warnings)
		}
	}
}

func TestParse_DateTime_NonexistentLocalTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		zone  Zone
	}{
		{name: "new york spring forward", input: "2026-03-08T02:30:00", zone: MustLoadZone("America/New_York")},
		{name: "lord howe half hour gap", input: "2026-10-04T02:15:00", zone: MustLoadZone("Australia/Lord_Howe")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := Parse(tc.input, WithZone(tc.zone))
			if r.Status != StatusInvalid {
				t.Fatalf("status = %v, want Invalid", r.Status)
			}
			if r.Error == nil {
				t.Fatal("Error is nil, want TimeError")
			}
			if r.Error.Code != CodeNonexistentTime {
				t.Fatalf("error code = %q, want %q", r.Error.Code, CodeNonexistentTime)
			}
			if r.Error.Hint == "" {
				t.Fatal("error Hint is empty")
			}
		})
	}
}

func TestParse_DateTime_InvalidTime(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"2026-03-27T25:00:00Z"},
		{"2026-03-27T13:60:00Z"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r := Parse(tt.input)
			if r.Status != StatusInvalid {
				t.Fatalf("status = %v, want Invalid for %q", r.Status, tt.input)
			}
		})
	}
}

func hasWarning(warnings []Warning, code WarningCode) bool {
	for _, w := range warnings {
		if w.Code == code {
			return true
		}
	}
	return false
}

func TestParse_DateTime_InvalidOffset(t *testing.T) {
	tests := []string{
		"2026-03-27T13:00:00+99:99",
		"20260327T130000+9999",
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			r := Parse(input)
			if r.Status != StatusInvalid {
				t.Fatalf("status = %v, want Invalid", r.Status)
			}
			if r.Error.Code != CodeInvalidZone {
				t.Errorf("error code = %q, want %q", r.Error.Code, CodeInvalidZone)
			}
		})
	}
}
