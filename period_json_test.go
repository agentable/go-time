package gotime

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPeriodMarshalJSON_Months(t *testing.T) {
	p := Months(3)
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	want := `{"kind":"period","iso":"P3M"}`
	if string(b) != want {
		t.Errorf("got %s, want %s", b, want)
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
