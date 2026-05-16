package gotime

import (
	"testing"

	"github.com/go-json-experiment/json"
)

func TestInstantMarshalJSON(t *testing.T) {
	i := UnixNanos(0)
	b, err := json.Marshal(i)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	want := `{"kind":"instant","iso":"1970-01-01T00:00:00Z","epoch_ms":0}`
	if string(b) != want {
		t.Errorf("got %s, want %s", b, want)
	}
}

func TestInstantUnmarshalJSON(t *testing.T) {
	var i Instant
	if err := json.Unmarshal([]byte(`{"kind":"instant","iso":"1970-01-01T00:00:00Z","epoch_ms":0}`), &i); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if i.UnixNano() != 0 {
		t.Errorf("got UnixNano=%d, want 0", i.UnixNano())
	}
}

func TestInstantJSONRoundTrip(t *testing.T) {
	orig := UnixNanos(1_000_000_007) // nanosecond precision
	b, _ := json.Marshal(orig)
	var got Instant
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if !orig.Equal(got) {
		t.Errorf("round-trip mismatch: got %v, want %v", got, orig)
	}
}

func TestInstantUnmarshalJSON_Invalid(t *testing.T) {
	var i Instant
	err := json.Unmarshal([]byte(`{"kind":"instant","iso":"not-a-time"}`), &i)
	if err == nil {
		t.Error("expected error for invalid instant, got nil")
	}
}
