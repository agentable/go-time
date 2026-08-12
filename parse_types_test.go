package gotime

import "testing"

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
	t.Parallel()

	var r ParseResult
	if _, ok := r.DateTime(); ok {
		t.Error("zero ParseResult.DateTime() should report ok=false")
	}
	if _, ok := r.LocalDateTime(); ok {
		t.Error("zero ParseResult.LocalDateTime() should report ok=false")
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
	if _, ok := r.Instant(); ok {
		t.Error("zero ParseResult.Instant() should report ok=false")
	}
	if _, ok := r.Period(); ok {
		t.Error("zero ParseResult.Period() should report ok=false")
	}
	if _, ok := r.Interval(); ok {
		t.Error("zero ParseResult.Interval() should report ok=false")
	}
}

func TestParseResult_Accessors_ByKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		opts     []Option
		wantKind Kind
		assert   func(*testing.T, ParseResult)
	}{
		{
			name:     "instant",
			input:    "2026-03-27T04:00:00Z",
			wantKind: KindInstant,
			assert: func(t *testing.T, r ParseResult) {
				t.Helper()
				if _, ok := r.Instant(); !ok {
					t.Fatal("Instant() ok=false, want true")
				}
			},
		},
		{
			name:     "datetime",
			input:    "2026-03-27T13:00:00",
			opts:     []Option{WithZone(MustLoadZone("Asia/Tokyo"))},
			wantKind: KindDateTime,
			assert: func(t *testing.T, r ParseResult) {
				t.Helper()
				if _, ok := r.DateTime(); !ok {
					t.Fatal("DateTime() ok=false, want true")
				}
			},
		},
		{
			name:     "local datetime",
			input:    "20260327T13",
			wantKind: KindLocalDateTime,
			assert: func(t *testing.T, r ParseResult) {
				t.Helper()
				if _, ok := r.LocalDateTime(); !ok {
					t.Fatal("LocalDateTime() ok=false, want true")
				}
			},
		},
		{
			name:     "date",
			input:    "2026-03-27",
			wantKind: KindDate,
			assert: func(t *testing.T, r ParseResult) {
				t.Helper()
				if _, ok := r.Date(); !ok {
					t.Fatal("Date() ok=false, want true")
				}
			},
		},
		{
			name:     "time",
			input:    "13:00:00",
			wantKind: KindTime,
			assert: func(t *testing.T, r ParseResult) {
				t.Helper()
				if _, ok := r.Time(); !ok {
					t.Fatal("Time() ok=false, want true")
				}
			},
		},
		{
			name:     "duration",
			input:    "PT1H30M",
			wantKind: KindDuration,
			assert: func(t *testing.T, r ParseResult) {
				t.Helper()
				if _, ok := r.Duration(); !ok {
					t.Fatal("Duration() ok=false, want true")
				}
			},
		},
		{
			name:     "period",
			input:    "P1Y2M3D",
			wantKind: KindPeriod,
			assert: func(t *testing.T, r ParseResult) {
				t.Helper()
				if _, ok := r.Period(); !ok {
					t.Fatal("Period() ok=false, want true")
				}
			},
		},
		{
			name:     "interval",
			input:    "2026-03-27T00:00:00Z/2026-03-28T00:00:00Z",
			wantKind: KindInterval,
			assert: func(t *testing.T, r ParseResult) {
				t.Helper()
				if _, ok := r.Interval(); !ok {
					t.Fatal("Interval() ok=false, want true")
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := Parse(tc.input, tc.opts...)
			if r.Status != StatusResolved {
				t.Fatalf("Parse(%q).Status = %q, want resolved", tc.input, r.Status)
			}
			if r.Kind != tc.wantKind {
				t.Fatalf("Parse(%q).Kind = %q, want %q", tc.input, r.Kind, tc.wantKind)
			}
			tc.assert(t, r)
		})
	}
}

func TestParseResult_Accessors_NonResolvedReturnFalse(t *testing.T) {
	t.Parallel()

	r := Parse("nonsense gibberish 12345!@#")
	if r.Status == StatusResolved {
		t.Fatalf("expected non-resolved status, got %q", r.Status)
	}
	if _, ok := r.Date(); ok {
		t.Fatal("Date() ok=true, want false")
	}
}

func TestParseResult_Accessors_AmbiguousReturnFalse(t *testing.T) {
	t.Parallel()

	r := Parse("04/05/2026")
	if r.Status != StatusAmbiguous || r.Kind != KindDate {
		t.Fatalf("Parse status/kind = %q/%q, want ambiguous/date", r.Status, r.Kind)
	}
	if _, ok := r.Date(); ok {
		t.Fatal("ambiguous Date() ok=true, want false")
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
				t.Fatalf("Parse(%q).Status = invalid, error = %#v; want non-invalid", tc.input, r.Error)
			}
			if r.HasZone != tc.hasZone {
				t.Errorf("Parse(%q).HasZone = %v, want %v", tc.input, r.HasZone, tc.hasZone)
			}
		})
	}
}
