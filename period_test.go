package gotime

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/go-json-experiment/json"
	"github.com/google/go-cmp/cmp"
)

func TestNewPeriod(t *testing.T) {
	t.Parallel()

	got := NewPeriod(1, -2, 3)
	want := Period{Years: 1, Months: -2, Days: 3}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("NewPeriod() mismatch (-want +got):\n%s", diff)
	}
}

func TestPeriod_ConstructorsUseInt32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  Period
		want Period
	}{
		{name: "years", got: Years(math.MaxInt32), want: Period{Years: math.MaxInt32}},
		{name: "months", got: Months(math.MinInt32), want: Period{Months: math.MinInt32}},
		{name: "days", got: Days(math.MaxInt32), want: Period{Days: math.MaxInt32}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tc.want, tc.got); diff != "" {
				t.Errorf("Period constructor mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPeriod_ComponentArithmetic(t *testing.T) {
	t.Parallel()

	base := Period{Years: 1, Months: -2, Days: 3}
	other := Period{Years: -4, Months: 5, Days: -6}
	got, err := base.Sub(other)
	if err != nil {
		t.Fatalf("Sub() error = %v", err)
	}
	want := Period{Years: 5, Months: -7, Days: 9}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Sub() mismatch (-want +got):\n%s", diff)
	}
}

func TestPeriod_Add(t *testing.T) {
	t.Parallel()

	got, err := (Period{Years: 1, Months: -2, Days: 3}).Add(Period{Years: -4, Months: 5, Days: -6})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	want := Period{Years: -3, Months: 3, Days: -3}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Add() mismatch (-want +got):\n%s", diff)
	}
}

func TestPeriod_Negate(t *testing.T) {
	t.Parallel()

	got, err := (Period{Years: 1, Months: -2, Days: 3}).Negate()
	if err != nil {
		t.Fatalf("Negate() error = %v", err)
	}
	want := Period{Years: -1, Months: 2, Days: -3}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Negate() mismatch (-want +got):\n%s", diff)
	}
}

func TestPeriod_NegateOverflow(t *testing.T) {
	t.Parallel()

	_, err := (Period{Years: math.MinInt32}).Negate()
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("Period{Years: math.MinInt32}.Negate() error = %v, want ErrOverflow", err)
	}
	var te *TimeError
	if !errors.As(err, &te) {
		t.Fatalf("Negate() error type = %T, want *TimeError", err)
	}
	if te.Hint == "" {
		t.Fatal("Negate() error Hint is empty")
	}
}

func TestPeriod_AddOverflow(t *testing.T) {
	t.Parallel()

	_, err := (Period{Months: math.MaxInt32}).Add(Period{Months: 1})
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("Period{Months: math.MaxInt32}.Add(Months(1)) error = %v, want ErrOverflow", err)
	}
	var te *TimeError
	if !errors.As(err, &te) || te.Hint == "" {
		t.Fatalf("Add() error = %#v, want *TimeError with Hint", err)
	}
}

func TestPeriod_SubOverflow(t *testing.T) {
	t.Parallel()

	_, err := (Period{Days: math.MinInt32}).Sub(Period{Days: 1})
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("Period{Days: math.MinInt32}.Sub(Days(1)) error = %v, want ErrOverflow", err)
	}
	var te *TimeError
	if !errors.As(err, &te) || te.Hint == "" {
		t.Fatalf("Sub() error = %#v, want *TimeError with Hint", err)
	}
}

func TestPeriod_AbsOverflow(t *testing.T) {
	t.Parallel()

	_, err := (Period{Months: math.MinInt32}).Abs()
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("Period{Months: math.MinInt32}.Abs() error = %v, want ErrOverflow", err)
	}
	var te *TimeError
	if !errors.As(err, &te) || te.Hint == "" {
		t.Fatalf("Abs() error = %#v, want *TimeError with Hint", err)
	}
}

func TestPeriod_Abs(t *testing.T) {
	t.Parallel()

	got, err := (Period{Years: -1, Months: 2, Days: -3}).Abs()
	if err != nil {
		t.Fatalf("Abs() error = %v", err)
	}
	want := Period{Years: 1, Months: 2, Days: 3}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Abs() mismatch (-want +got):\n%s", diff)
	}
}

func TestPeriod_IsNegative(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		p    Period
		want bool
	}{
		{name: "zero", p: Period{}, want: false},
		{name: "all positive", p: Period{Years: 1, Months: 2, Days: 3}, want: false},
		{name: "negative year", p: Period{Years: -1}, want: true},
		{name: "mixed signs", p: Period{Years: 1, Months: -2, Days: 3}, want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.p.IsNegative(); got != tc.want {
				t.Errorf("IsNegative() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPeriod_ISO8601(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		p    Period
		want string
	}{
		{name: "zero", p: Period{}, want: "P0D"},
		{name: "date fields", p: Period{Years: 1, Months: 2, Days: 3}, want: "P1Y2M3D"},
		{name: "all negative", p: Period{Years: -1, Months: -2, Days: -3}, want: "-P1Y2M3D"},
		{name: "mixed signs", p: Period{Years: 1, Months: -2, Days: 3}, want: "P+1Y-2M+3D"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.p.ISO8601(); got != tc.want {
				t.Errorf("ISO8601() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPeriod_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		p    Period
	}{
		{name: "zero", p: Period{}},
		{name: "positive fields", p: Period{Years: 1, Months: 2, Days: 3}},
		{name: "all negative", p: Period{Months: -2, Days: -3}},
		{name: "mixed signs", p: Period{Years: 1, Months: -2, Days: 3}},
		{name: "minimum component", p: Period{Years: math.MinInt32}},
		{name: "mixed minimum component", p: Period{Years: math.MinInt32, Months: 1}},
	}
	seen := make(map[string]Period, len(tests))
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.p.String()
			if want := tc.p.ISO8601(); got != want {
				t.Errorf("String() = %q, want ISO8601() %q", got, want)
			}
			parsed, err := ParsePeriod(got)
			if err != nil {
				t.Fatalf("ParsePeriod(String()) error = %v", err)
			}
			if diff := cmp.Diff(tc.p, parsed); diff != "" {
				t.Errorf("ParsePeriod(String()) mismatch (-want +got):\n%s", diff)
			}
			if previous, ok := seen[got]; ok {
				t.Fatalf("String() collision for %#v and %#v: %q", previous, tc.p, got)
			}
			seen[got] = tc.p
		})
	}
}

func TestPeriod_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		orig Period
	}{
		{name: "minimum", orig: Period{Years: math.MinInt32, Months: math.MinInt32, Days: math.MinInt32}},
		{name: "maximum", orig: Period{Years: math.MaxInt32, Months: math.MaxInt32, Days: math.MaxInt32}},
		{name: "positive", orig: Period{Years: 1, Months: 3, Days: 14}},
		{name: "mixed signs", orig: Period{Years: 1, Months: -2, Days: 3}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b, err := json.Marshal(tc.orig)
			if err != nil {
				t.Fatalf("json.Marshal: %v", err)
			}

			var got Period
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("json.Unmarshal: %v", err)
			}

			if diff := cmp.Diff(tc.orig, got); diff != "" {
				t.Errorf("Period round-trip mismatch (-want +got):\n%s", diff)
			}
			again, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("json.Marshal(round-tripped): %v", err)
			}
			if !bytes.Equal(again, b) {
				t.Fatalf("Marshal after round-trip = %s, want %s", again, b)
			}
		})
	}
}

func TestPeriod_UnmarshalJSONWeeks(t *testing.T) {
	t.Parallel()

	var got Period
	if err := json.Unmarshal([]byte(`{"kind":"period","iso":"-P2W"}`), &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	want := Period{Days: -14}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Period mismatch (-want +got):\n%s", diff)
	}
}

func TestPeriod_UnmarshalJSONInvalidISO(t *testing.T) {
	t.Parallel()

	for _, iso := range []string{"P", "-P-1Y", "-P+1Y", "-P-1W+2D"} {
		t.Run(iso, func(t *testing.T) {
			t.Parallel()

			var got Period
			input := []byte(fmt.Sprintf(`{"kind":"period","iso":%q}`, iso))
			err := json.Unmarshal(input, &got)
			if !errors.Is(err, ErrInvalidPeriod) {
				t.Fatalf("json.Unmarshal(%q) error = %v, want ErrInvalidPeriod", iso, err)
			}
			if !errors.Is(err, errInvalidISO8601Period) {
				t.Fatalf("json.Unmarshal(%q) error = %v, want errInvalidISO8601Period", iso, err)
			}
		})
	}
}
