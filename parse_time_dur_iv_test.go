package gotime

import (
	"testing"
)

// ── Time ──────────────────────────────────────────────────────────────────

func TestParse_Time_24h(t *testing.T) {
	tests := []struct {
		input string
		h, m  int
		s     int
	}{
		{"15:00", 15, 0, 0},
		{"08:30:45", 8, 30, 45},
		{"00:00", 0, 0, 0},
		{"23:59:59", 23, 59, 59},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r := Parse(tt.input)
			if r.Status != StatusResolved {
				t.Fatalf("status = %v, want Resolved", r.Status)
			}
			if r.Kind != KindTime {
				t.Fatalf("kind = %v, want KindTime", r.Kind)
			}
			ti, _ := r.Time()
			if ti.Hour() != tt.h || ti.Minute() != tt.m || ti.Second() != tt.s {
				t.Errorf("time = %v, want %02d:%02d:%02d", ti, tt.h, tt.m, tt.s)
			}
		})
	}
}

func TestParse_Time_12h(t *testing.T) {
	tests := []struct {
		input string
		h, m  int
	}{
		{"3pm", 15, 0},
		{"3:30 PM", 15, 30},
		{"3:30PM", 15, 30},
		{"12am", 0, 0},
		{"12pm", 12, 0},
		{"1AM", 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r := Parse(tt.input)
			if r.Status != StatusResolved {
				t.Fatalf("status = %v, want Resolved", r.Status)
			}
			if r.Kind != KindTime {
				t.Fatalf("kind = %v, want KindTime", r.Kind)
			}
			ti, _ := r.Time()
			if ti.Hour() != tt.h || ti.Minute() != tt.m {
				t.Errorf("time = %v, want %02d:%02d", ti, tt.h, tt.m)
			}
		})
	}
}

func TestParse_Time_Invalid(t *testing.T) {
	r := Parse("25:00")
	if r.Status != StatusInvalid {
		t.Fatalf("status = %v, want Invalid", r.Status)
	}
	if r.Error.Code != CodeInvalidTime {
		t.Errorf("error code = %q, want %q", r.Error.Code, CodeInvalidTime)
	}
}

func TestParse_DateTime_FractionalNanoseconds(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  int
	}{
		{input: "2026-03-27T13:00:00.1", want: 100_000_000},
		{input: "2026-03-27T13:00:00.123456789", want: 123_456_789},
		{input: "20260327T130000.1234567899", want: 123_456_789},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			r := Parse(tc.input)
			if r.Status != StatusResolved {
				t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
			}
			var got int
			switch r.Kind {
			case KindDateTime:
				dt, _ := r.DateTime()
				got = dt.Clock().Nanosecond()
			case KindInstant:
				i, _ := r.Instant()
				got = i.Std().Nanosecond()
			default:
				t.Fatalf("kind = %v, want KindDateTime or KindInstant", r.Kind)
			}
			if got != tc.want {
				t.Fatalf("nanosecond = %d, want %d", got, tc.want)
			}
		})
	}
}

// ── Duration ──────────────────────────────────────────────────────────────

func TestParse_Duration(t *testing.T) {
	tests := []struct {
		input   string
		wantMin float64
	}{
		{"PT1H30M", 90},
		{"PT0S", 0},
		{"PT90M", 90},
		{"PT3600S", 60},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r := Parse(tt.input)
			if r.Status != StatusResolved {
				t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
			}
			if r.Kind != KindDuration {
				t.Fatalf("kind = %v, want KindDuration", r.Kind)
			}
			d, _ := r.Duration()
			got := d.InMinutes()
			if got != tt.wantMin {
				t.Errorf("InMinutes() = %v, want %v", got, tt.wantMin)
			}
		})
	}
}

func TestParse_Period_Days(t *testing.T) {
	// P5D = 5 calendar days (Period, not Duration).
	r := Parse("P5D")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v, want Resolved", r.Status)
	}
	if r.Kind != KindPeriod {
		t.Fatalf("kind = %v, want KindPeriod (P-with-date-components is Period, not Duration)", r.Kind)
	}
	p, _ := r.Period()
	if p.Days != 5 {
		t.Errorf("Days = %v, want 5", p.Days)
	}
}

func TestParse_Period_Years(t *testing.T) {
	// P1Y = 1 calendar year — routes to Period, not Duration with 365.25-day approximation.
	r := Parse("P1Y")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v, want Resolved", r.Status)
	}
	if r.Kind != KindPeriod {
		t.Fatalf("kind = %v, want KindPeriod", r.Kind)
	}
	p, _ := r.Period()
	if p.Years != 1 {
		t.Errorf("Years = %v, want 1", p.Years)
	}
}

func TestParse_MixedISO_Rejected(t *testing.T) {
	// P1DT2H mixes calendar (D) and clock (H) components — no single gotime
	// type can carry both, so the parser must refuse it cleanly.
	tests := []string{"P1DT2H", "P1Y2DT3H", "P1MT30M"}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			r := Parse(input)
			if r.Status != StatusInvalid {
				t.Fatalf("status = %v, want Invalid for mixed ISO input", r.Status)
			}
			if r.Error.Code != CodeInvalidFormat {
				t.Errorf("error code = %q, want %q", r.Error.Code, CodeInvalidFormat)
			}
		})
	}
}

func TestParse_Duration_Invalid(t *testing.T) {
	tests := []string{"PT", "P"}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			r := Parse(input)
			if r.Status != StatusInvalid {
				t.Fatalf("status = %v, want Invalid", r.Status)
			}
			if r.Error.Code != CodeInvalidFormat {
				t.Errorf("error code = %q, want %q", r.Error.Code, CodeInvalidFormat)
			}
		})
	}
}

func TestParse_Duration_Overflow(t *testing.T) {
	tests := []string{
		"PT999999999999999999999H",
		"PT2562047H48M",
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			r := Parse(input)
			if r.Status != StatusInvalid {
				t.Fatalf("status = %v, want Invalid", r.Status)
			}
			if r.Error.Code != CodeOverflow {
				t.Errorf("error code = %q, want %q", r.Error.Code, CodeOverflow)
			}
		})
	}
}

func TestParse_Period_Overflow(t *testing.T) {
	r := Parse("P2147483648D")
	if r.Status != StatusInvalid {
		t.Fatalf("status = %v, want Invalid", r.Status)
	}
	if r.Error.Code != CodeOverflow {
		t.Errorf("error code = %q, want %q", r.Error.Code, CodeOverflow)
	}
}

// ── Interval ──────────────────────────────────────────────────────────────

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
	if iv.Length().InHours() != 12 {
		t.Errorf("length = %v hours, want 12", iv.Length().InHours())
	}
}

func TestParse_Interval_DurationEnd(t *testing.T) {
	r := Parse("PT12H/2026-03-28T00:00:00Z")
	if r.Status != StatusResolved {
		t.Fatalf("status = %v (err=%v), want Resolved", r.Status, r.Error)
	}
	iv, _ := r.Interval()
	if iv.Length().InHours() != 12 {
		t.Errorf("length = %v hours, want 12", iv.Length().InHours())
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
