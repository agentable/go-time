package gotime

import (
	"testing"

	"github.com/go-json-experiment/json"
	"github.com/google/go-cmp/cmp"
)

func TestPeriod_ComponentArithmetic(t *testing.T) {
	t.Parallel()

	base := Period{Years: 1, Months: -2, Days: 3}
	other := Period{Years: -4, Months: 5, Days: -6}

	tests := []struct {
		name string
		got  Period
		want Period
	}{
		{name: "negate", got: base.Negate(), want: Period{Years: -1, Months: 2, Days: -3}},
		{name: "add", got: base.Add(other), want: Period{Years: -3, Months: 3, Days: -3}},
		{name: "subtract", got: base.Sub(other), want: Period{Years: 5, Months: -7, Days: 9}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if diff := cmp.Diff(tc.want, tc.got); diff != "" {
				t.Errorf("Period mismatch (-want +got):\n%s", diff)
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

func TestPeriod_RFC5545(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		p    Period
		want string
	}{
		{name: "positive whole weeks", p: Days(14), want: "P2W"},
		{name: "negative whole weeks", p: Days(-21), want: "-P3W"},
		{name: "non-week period falls back to ISO", p: Period{Years: 1, Months: 2, Days: 3}, want: "P1Y2M3D"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.p.RFC5545(); got != tc.want {
				t.Errorf("RFC5545() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPeriod_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		p    Period
		want string
	}{
		{name: "zero", p: Period{}, want: "0d"},
		{name: "positive fields", p: Period{Years: 1, Months: 2, Days: 3}, want: "1y2mo3d"},
		{name: "all negative", p: Period{Months: -2, Days: -3}, want: "-2mo3d"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.p.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPeriod_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	orig := Period{Years: 1, Months: 3, Days: 14}
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var got Period
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if diff := cmp.Diff(orig, got); diff != "" {
		t.Errorf("Period round-trip mismatch (-want +got):\n%s", diff)
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

	var got Period
	if err := json.Unmarshal([]byte(`{"kind":"period","iso":"P"}`), &got); err == nil {
		t.Fatal("json.Unmarshal invalid period succeeded, want error")
	}
}
