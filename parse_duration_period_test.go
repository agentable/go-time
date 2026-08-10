package gotime

import (
	"errors"
	"math"
	"testing"
)

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
