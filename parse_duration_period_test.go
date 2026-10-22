package gotime

import (
	"errors"
	"math"
	"testing"
)

func TestParseDurationPrecisionAndRange(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input string
		want  Duration
		err   error
	}{
		{"PT0.0000000001S", 0, ErrInvalidDuration},
		{"PT0.0000000000S", 0, ErrInvalidDuration},
		{"PT0.0000000001H", 0, ErrInvalidDuration},
		{"PT0.000000001S", Nanosecond, nil},
		{"PT9223372036.854775807S", Duration(maxInt64), nil},
		{"-PT9223372036.854775808S", Duration(minInt64), nil},
		{"PT9223372036.854775808S", 0, ErrOverflow},
		{"-PT9223372036.854775809S", 0, ErrOverflow},
	} {
		d, err := ParseDuration(tc.input)
		r := Parse(tc.input)
		if tc.err == nil {
			parsed, ok := r.Duration()
			if err != nil || d != tc.want || !ok || parsed != tc.want {
				t.Errorf("%s: %v %v", tc.input, d, err)
			}
			continue
		}
		var detail *TimeError
		if d != 0 || r.Status != StatusInvalid || !errors.Is(r.Error, tc.err) || !errors.Is(err, tc.err) || !errors.As(err, &detail) || detail.Hint == "" {
			t.Errorf("%s: status=%s error=%v", tc.input, r.Status, err)
		}
		if errors.Is(tc.err, ErrInvalidDuration) && errors.Is(err, ErrOverflow) {
			t.Errorf("precision error is also overflow: %v", err)
		}
	}
}

func TestParse_Duration(t *testing.T) {
	tests := []struct {
		input   string
		wantMin float64
	}{
		{"PT1H30M", 90},
		{"PT0S", 0},
		{"PT90M", 90},
		{"PT3600S", 60},
		{"-PT30M", -30},
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
			got := d.Minutes()
			if got != tt.wantMin {
				t.Errorf("Minutes() = %v, want %v", got, tt.wantMin)
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

func TestParsePeriodRejectsIncompleteTimeSection(t *testing.T) {
	for _, input := range []string{"P1DT", "P1YT", "-P2WT", "P+1Y-2MT", " P1DT ", "P", "PT", "P1DT1H"} {
		t.Run(input, func(t *testing.T) {
			r := Parse(input)
			if r.Status != StatusInvalid || !errors.Is(r.Error, ErrInvalidFormat) {
				t.Errorf("Parse(%q): status=%s error=%v, want Invalid and ErrInvalidFormat", input, r.Status, r.Error)
			}
			if r.Input != input || r.Error == nil || r.Error.Input != input || r.Error.Hint == "" {
				t.Errorf("Parse(%q): input=%q detail=%+v, want original input and actionable error", input, r.Input, r.Error)
			}
			p, err := ParsePeriod(input)
			if p != (Period{}) || !errors.Is(err, ErrInvalidFormat) {
				t.Errorf("ParsePeriod(%q) = %v, %v; want zero and ErrInvalidFormat", input, p, err)
			}
			var detail *TimeError
			if !errors.As(err, &detail) || detail.Input != input || detail.Hint == "" {
				t.Errorf("ParsePeriod(%q): detail=%+v, want original input and actionable error", input, detail)
			}
		})
	}
}

func TestParsePeriodCompleteComponents(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  Period
	}{
		{"P1D", Days(1)},
		{"P1Y", Years(1)},
		{"-P2W", Days(-14)},
		{"P+1Y-2M", Period{Years: 1, Months: -2}},
	} {
		t.Run(tc.input, func(t *testing.T) {
			p, err := ParsePeriod(tc.input)
			parsed, ok := Parse(tc.input).Period()
			if err != nil || p != tc.want || !ok || parsed != tc.want {
				t.Errorf("%q: typed=%v error=%v diagnostic=%v ok=%v, want %v", tc.input, p, err, parsed, ok, tc.want)
			}
		})
	}
	for _, tc := range []struct {
		input string
		want  Duration
	}{
		{"PT0S", 0},
		{"PT1H", Hour},
	} {
		t.Run(tc.input, func(t *testing.T) {
			d, err := ParseDuration(tc.input)
			parsed, ok := Parse(tc.input).Duration()
			if err != nil || d != tc.want || !ok || parsed != tc.want {
				t.Errorf("%q: typed=%v error=%v diagnostic=%v ok=%v, want %v", tc.input, d, err, parsed, ok, tc.want)
			}
		})
	}
}

func TestParse_Duration_RejectsExponentSeconds(t *testing.T) {
	for _, input := range []string{"PT1e3S", "PT1E-9S"} {
		t.Run(input, func(t *testing.T) {
			r := Parse(input)
			if r.Status != StatusInvalid {
				t.Fatalf("Parse(%q).Status = %v, want Invalid", input, r.Status)
			}
			if r.Error.Code != CodeInvalidFormat {
				t.Errorf("Parse(%q).Error.Code = %q, want %q", input, r.Error.Code, CodeInvalidFormat)
			}

			_, err := ParseDuration(input)
			if !errors.Is(err, ErrInvalidFormat) {
				t.Errorf("ParseDuration(%q) error = %v, want ErrInvalidFormat", input, err)
			}
		})
	}
}

func TestParse_Duration_Overflow(t *testing.T) {
	tests := []string{
		"PT999999999999999999999H",
		"PT2562048H",
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
	for _, input := range []string{"P2147483648D", "P2147483649D"} {
		t.Run(input, func(t *testing.T) {
			r := Parse(input)
			if r.Status != StatusInvalid {
				t.Fatalf("status = %v, want Invalid", r.Status)
			}
			if r.Error == nil || !errors.Is(r.Error, ErrOverflow) || r.Error.Code != CodeOverflow {
				t.Errorf("error = %v, want ErrOverflow with code %q", r.Error, CodeOverflow)
			}
		})
	}
}

func TestParse_Period_Minimum(t *testing.T) {
	t.Parallel()

	p, err := ParsePeriod("-P2147483648Y")
	if err != nil {
		t.Fatalf("ParsePeriod(minimum) error = %v", err)
	}
	if p != (Period{Years: math.MinInt32}) {
		t.Fatalf("ParsePeriod(minimum) = %+v, want Years=%d", p, int32(math.MinInt32))
	}
}

func TestParse_Period_RejectsLeadingAndComponentSigns(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"-P-1Y", "-P+1Y", "-P-1W+2D"} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			r := Parse(input)
			if r.Status != StatusInvalid || r.Error == nil {
				t.Fatalf("Parse(%q) = %#v, want invalid result", input, r)
			}
			if !errors.Is(r.Error, ErrInvalidPeriod) {
				t.Fatalf("Parse(%q) error = %v, want ErrInvalidPeriod", input, r.Error)
			}
			if _, err := ParsePeriod(input); !errors.Is(err, ErrInvalidPeriod) {
				t.Fatalf("ParsePeriod(%q) error = %v, want ErrInvalidPeriod", input, err)
			}
		})
	}
}
