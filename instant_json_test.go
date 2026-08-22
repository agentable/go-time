package gotime

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"encoding/json/v2"
)

func TestInstantMarshalJSON(t *testing.T) {
	i := UnixNanos(0)
	b, err := json.Marshal(i)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	want := `{"kind":"instant","iso":"1970-01-01T00:00:00Z"}`
	if string(b) != want {
		t.Errorf("got %s, want %s", b, want)
	}
}

func TestInstantUnmarshalJSON(t *testing.T) {
	var i Instant
	if err := json.Unmarshal([]byte(`{"kind":"instant","iso":"1970-01-01T00:00:00Z"}`), &i); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	got, err := i.UnixNano()
	if err != nil {
		t.Fatalf("UnixNano() error = %v", err)
	}
	if got != 0 {
		t.Errorf("got UnixNano=%d, want 0", got)
	}
}

func TestInstantUnmarshalJSON_NormalizesOffsetInput(t *testing.T) {
	t.Parallel()

	var instant Instant
	input := []byte(`{"kind":"instant","iso":"1970-01-01T09:00:00+09:00"}`)
	if err := json.Unmarshal(input, &instant); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !instant.Equal(UnixNanos(0)) {
		t.Fatalf("Unmarshal() = %v, want Unix epoch", instant)
	}
	got, err := json.Marshal(instant)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	want := `{"kind":"instant","iso":"1970-01-01T00:00:00Z"}`
	if string(got) != want {
		t.Fatalf("Marshal() = %s, want %s", got, want)
	}
}

func TestInstantJSONRoundTrip(t *testing.T) {
	orig := UnixNanos(1_000_000_007) // nanosecond precision
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var got Instant
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if !orig.Equal(got) {
		t.Errorf("round-trip mismatch: got %v, want %v", got, orig)
	}
	again, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("second Marshal() error = %v", err)
	}
	if !bytes.Equal(again, b) {
		t.Fatalf("Marshal -> Unmarshal -> Marshal = %s, want %s", again, b)
	}
}

func TestInstantMarshalJSON_RejectsUnparseableYear(t *testing.T) {
	i := InstantFromTime(time.Date(10_000, time.January, 1, 0, 0, 0, 0, time.UTC))
	_, err := json.Marshal(i)
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("Marshal(year 10000) error = %v, want ErrOverflow", err)
	}
	var te *TimeError
	if !errors.As(err, &te) || te.Hint == "" {
		t.Fatalf("Marshal(year 10000) error = %#v, want TimeError with hint", err)
	}
}

func TestInstantUnmarshalJSON_Invalid(t *testing.T) {
	var i Instant
	err := json.Unmarshal([]byte(`{"kind":"instant","iso":"not-a-time"}`), &i)
	if err == nil {
		t.Error("expected error for invalid instant, got nil")
	}
}
