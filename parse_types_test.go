package gotime

import (
	"testing"

	"golang.org/x/text/language"
)

func TestParseResult_ZeroValue(t *testing.T) {
	var r ParseResult
	if r.Status != "" {
		t.Errorf("zero ParseResult.Status = %q, want empty", r.Status)
	}
	if r.Kind != "" {
		t.Errorf("zero ParseResult.Kind = %q, want empty", r.Kind)
	}
	if r.Warnings != nil {
		t.Errorf("zero ParseResult.Warnings = %v, want nil", r.Warnings)
	}
	if r.Candidates != nil {
		t.Errorf("zero ParseResult.Candidates = %v, want nil", r.Candidates)
	}
	if r.Error != nil {
		t.Errorf("zero ParseResult.Error = %v, want nil", r.Error)
	}
}

func TestParseResult_Accessors_ZeroValue(t *testing.T) {
	var r ParseResult
	if _, ok := r.DateTime(); ok {
		t.Error("zero ParseResult.DateTime() should report ok=false")
	}
	if _, ok := r.Date(); ok {
		t.Error("zero ParseResult.Date() should report ok=false")
	}
	if _, ok := r.Time(); ok {
		t.Error("zero ParseResult.Time() should report ok=false")
	}
	if _, ok := r.Duration(); ok {
		t.Error("zero ParseResult.Duration() should report ok=false")
	}
	if v := r.Value(); v != nil {
		t.Errorf("zero ParseResult.Value() = %v, want nil", v)
	}
}

func TestParseResult_Value_ByKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  any
	}{
		{
			name:  "datetime",
			input: "2026-03-27T13:00:00+09:00",
			want:  DateTime{},
		},
		{
			name:  "date",
			input: "2026-03-27",
			want:  Date{},
		},
		{
			name:  "time",
			input: "13:00:00",
			want:  Time{},
		},
		{
			name:  "duration",
			input: "PT1H30M",
			want:  Duration(0),
		},
		{
			name:  "period",
			input: "P1Y2M3D",
			want:  Period{},
		},
		{
			name:  "interval",
			input: "2026-03-27T00:00:00Z/2026-03-28T00:00:00Z",
			want:  Interval{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := Parse(tc.input)
			if r.Status != StatusResolved {
				t.Fatalf("Parse(%q).Status = %q, want resolved", tc.input, r.Status)
			}
			v := r.Value()
			gotType := typeName(v)
			wantType := typeName(tc.want)
			if gotType != wantType {
				t.Errorf("Value() type = %s, want %s", gotType, wantType)
			}
		})
	}
}

func TestParseResult_Value_NilWhenNotResolved(t *testing.T) {
	t.Parallel()

	r := Parse("nonsense gibberish 12345!@#")
	if r.Status == StatusResolved {
		t.Fatalf("expected non-resolved status, got %q", r.Status)
	}
	if v := r.Value(); v != nil {
		t.Errorf("non-resolved Value() = %v, want nil", v)
	}
}

func typeName(v any) string {
	switch v.(type) {
	case Instant:
		return "Instant"
	case DateTime:
		return "DateTime"
	case Date:
		return "Date"
	case Time:
		return "Time"
	case Duration:
		return "Duration"
	case Period:
		return "Period"
	case Interval:
		return "Interval"
	case nil:
		return "nil"
	default:
		return "unknown"
	}
}

func TestWithInputLocale(t *testing.T) {
	cfg := &config{}
	tag := language.MustParse("zh-Hans")
	WithInputLocale(tag)(cfg)
	if cfg.lang != tag {
		t.Errorf("WithInputLocale set lang = %q, want %q", cfg.lang, tag)
	}
}

func TestWithZone(t *testing.T) {
	cfg := &config{}
	zone := MustLoadZone("Asia/Tokyo")
	WithZone(zone)(cfg)
	if !cfg.zone.Equal(zone) {
		t.Errorf("WithZone set zone = %q, want %q", cfg.zone.ID(), zone.ID())
	}
}

func TestWithReference(t *testing.T) {
	now := Now()
	cfg := &config{}
	WithReference(now)(cfg)
	if !cfg.relativeTo.Equal(now) {
		t.Error("WithReference did not set relativeTo correctly")
	}
}

func TestOptions_Composable(t *testing.T) {
	cfg := &config{}
	enUS := language.MustParse("en-US")
	for _, opt := range []Option{
		WithInputLocale(enUS),
		WithZone(MustLoadZone("America/New_York")),
	} {
		opt(cfg)
	}
	if cfg.lang != enUS {
		t.Errorf("lang = %q, want %q", cfg.lang, enUS)
	}
	if cfg.zone.ID() != "America/New_York" {
		t.Errorf("zone = %q, want %q", cfg.zone.ID(), "America/New_York")
	}
}

func TestParseResult_HasZone(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		hasZone bool
	}{
		{"RFC3339 UTC", "2026-03-27T04:00:00Z", true},
		{"RFC3339 offset", "2026-03-27T13:00:00+09:00", true},
		{"naive datetime", "2026-03-27T13:00:00", false},
		{"date only", "2026-03-27", false},
		{"compact UTC", "20260327T130000Z", true},
		{"compact no offset", "20260327T130000", false},
		{"time only", "13:00", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := Parse(tc.input)
			if r.Status == StatusInvalid {
				t.Skipf("input %q not parseable: %v", tc.input, r.Error)
			}
			if r.HasZone != tc.hasZone {
				t.Errorf("Parse(%q).HasZone = %v, want %v", tc.input, r.HasZone, tc.hasZone)
			}
		})
	}
}

func TestOptions_LastWriterWins(t *testing.T) {
	cfg := &config{}
	WithInputLocale(language.MustParse("en-US"))(cfg)
	zh := language.MustParse("zh-Hans")
	WithInputLocale(zh)(cfg)
	if cfg.lang != zh {
		t.Errorf("lang = %q, want %q (last writer wins)", cfg.lang, zh)
	}
}
