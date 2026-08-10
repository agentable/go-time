package gotime

import (
	"testing"

	"golang.org/x/text/language"
)

func TestParse_Interval_DateToDate(t *testing.T) {
	r := Parse("2026-03-27/2026-03-28")
	if r.Status != StatusInvalid {
		t.Fatalf("status = %v, want Invalid", r.Status)
	}
	if r.Error.Code != CodeIncompatibleTypes {
		t.Errorf("error code = %q, want %q", r.Error.Code, CodeIncompatibleTypes)
	}
}

func TestParse_Interval_DatetimeToDatetime(t *testing.T) {
	r := Parse("2026-03-27T00:00:00Z/2026-03-28T00:00:00Z")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	if r.Kind != KindInterval {
		t.Fatalf("kind = %v, want KindInterval", r.Kind)
	}
}

func TestParse_Interval_StartDuration(t *testing.T) {
	r := Parse("2026-03-27T00:00:00Z/PT12H")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	if r.Kind != KindInterval {
		t.Fatalf("kind = %v, want KindInterval", r.Kind)
	}
	iv, _ := r.Interval()
	length, err := iv.Length()
	if err != nil {
		t.Fatalf("Length() error = %v", err)
	}
	if length.InHours() != 12 {
		t.Errorf("length = %v hours, want 12", length.InHours())
	}
}

func TestParse_Interval_DurationEnd(t *testing.T) {
	r := Parse("PT12H/2026-03-28T00:00:00Z")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	iv, _ := r.Interval()
	length, err := iv.Length()
	if err != nil {
		t.Fatalf("Length() error = %v", err)
	}
	if length.InHours() != 12 {
		t.Errorf("length = %v hours, want 12", length.InHours())
	}
}

func TestParse_Interval_EndBeforeStart(t *testing.T) {
	r := Parse("2026-03-28T00:00:00Z/2026-03-27T00:00:00Z")
	if r.Status != StatusInvalid {
		t.Fatalf("status = %v, want Invalid", r.Status)
	}
	if r.Error.Code != CodeIntervalReversed {
		t.Errorf("error code = %q, want %q", r.Error.Code, CodeIntervalReversed)
	}
}

func TestParse_Interval_AmbiguousEndpoint(t *testing.T) {
	r := Parse(
		"2026-11-01T01:30:00/2026-11-01T03:00:00",
		WithZone(MustLoadZone("America/New_York")),
	)
	if r.Status != StatusAmbiguous {
		t.Fatalf("status = %v (err=%v), want Ambiguous", r.Status, r.Error)
	}
}

func TestParse_Interval_RejectsNaturalLanguageEndpoint(t *testing.T) {
	r := Parse(
		"tomorrow/2026-03-28T00:00:00Z",
		WithInputLocale(language.English),
		WithReference(fixedNow),
	)
	if r.Status != StatusInvalid {
		t.Fatalf("status = %v, want Invalid", r.Status)
	}
}
