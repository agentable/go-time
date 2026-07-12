package gotime

import (
	"errors"
	"testing"
	"time"

	"github.com/go-json-experiment/json"
)

func TestIntervalMarshalJSON(t *testing.T) {
	start := UnixNanos(0)
	end := UnixNanos(3_600_000_000_000) // +1h
	iv := mustInterval(t, start, end)
	b, err := json.Marshal(iv)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	want := `{"kind":"interval","start":"1970-01-01T00:00:00Z","end":"1970-01-01T01:00:00Z"}`
	if string(b) != want {
		t.Errorf("got %s, want %s", b, want)
	}
}

func TestIntervalUnmarshalJSON(t *testing.T) {
	var iv Interval
	input := `{"kind":"interval","start":"1970-01-01T00:00:00Z","end":"1970-01-01T01:00:00Z"}`
	if err := json.Unmarshal([]byte(input), &iv); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if iv.Start().UnixNano() != 0 {
		t.Errorf("start = %d, want 0", iv.Start().UnixNano())
	}
	if iv.End().UnixNano() != 3_600_000_000_000 {
		t.Errorf("end = %d, want 3600000000000", iv.End().UnixNano())
	}
}

func TestIntervalJSONRoundTrip(t *testing.T) {
	start := UnixNanos(1_000_000_000)
	end := UnixNanos(2_000_000_000)
	orig := mustInterval(t, start, end)
	b, _ := json.Marshal(orig)
	var got Interval
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if !orig.Start().Equal(got.Start()) || !orig.End().Equal(got.End()) {
		t.Errorf("round-trip mismatch: got %v, want %v", got, orig)
	}
}

func TestIntervalMarshalJSON_RejectsInvalidPrivateValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		iv   Interval
		want error
	}{
		{
			name: "reversed",
			iv:   Interval{start: UnixSeconds(1), end: UnixSeconds(0)},
			want: ErrIntervalReversed,
		},
		{
			name: "endpoint outside wire domain",
			iv: Interval{
				start: InstantFromTime(time.Date(10_000, time.January, 1, 0, 0, 0, 0, time.UTC)),
				end:   InstantFromTime(time.Date(10_000, time.January, 2, 0, 0, 0, 0, time.UTC)),
			},
			want: ErrOverflow,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := json.Marshal(tc.iv)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Marshal() error = %v, want %v", err, tc.want)
			}
			var te *TimeError
			if !errors.As(err, &te) || te.Hint == "" {
				t.Fatalf("Marshal() error = %#v, want TimeError with hint", err)
			}
		})
	}
}

func TestIntervalUnmarshalJSON_Invalid(t *testing.T) {
	var iv Interval
	err := json.Unmarshal([]byte(`{"kind":"interval"}`), &iv)
	if err == nil {
		t.Error("expected error for missing start/end, got nil")
	}
}

func TestIntervalUnmarshalJSON_RejectsInvalidFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		json string
	}{
		{name: "invalid start", json: `{"kind":"interval","start":"not-a-time","end":"1970-01-01T01:00:00Z"}`},
		{name: "invalid end", json: `{"kind":"interval","start":"1970-01-01T00:00:00Z","end":"not-a-time"}`},
		{name: "end before start", json: `{"kind":"interval","start":"1970-01-01T01:00:00Z","end":"1970-01-01T00:00:00Z"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var iv Interval
			err := json.Unmarshal([]byte(tc.json), &iv)
			if err == nil {
				t.Fatal("Unmarshal() error = nil, want error")
			}
		})
	}
}

func TestIntervalUnmarshalJSON_WrapsEndBeforeStartSentinel(t *testing.T) {
	t.Parallel()

	var iv Interval
	err := json.Unmarshal([]byte(`{"kind":"interval","start":"1970-01-01T01:00:00Z","end":"1970-01-01T00:00:00Z"}`), &iv)
	if !errors.Is(err, ErrIntervalReversed) {
		t.Fatalf("Unmarshal() error = %v, want ErrIntervalReversed", err)
	}
}
