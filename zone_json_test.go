package gotime

import (
	"errors"
	"testing"
	"time"

	"github.com/go-json-experiment/json"
)

func TestZoneMarshalJSON(t *testing.T) {
	z := MustLoadZone("Asia/Tokyo")
	b, err := json.Marshal(z)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	// Output is deterministic — no time-dependent fields.
	want := `{"kind":"zone","id":"Asia/Tokyo"}`
	if string(b) != want {
		t.Errorf("got %s, want %s", b, want)
	}
}

func TestZoneMarshalJSON_UTC(t *testing.T) {
	b, err := json.Marshal(UTC)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	want := `{"kind":"zone","id":"UTC"}`
	if string(b) != want {
		t.Errorf("got %s, want %s", b, want)
	}
}

func TestZoneMarshalJSON_FixedOffsetRejectsZoneWire(t *testing.T) {
	z := Zone{id: "+08:00", loc: time.FixedZone("+08:00", 8*3600)}
	_, err := json.Marshal(z)
	if !errors.Is(err, ErrInvalidZone) {
		t.Fatalf("Marshal error = %v, want ErrInvalidZone", err)
	}
}

func TestZoneUnmarshalJSON(t *testing.T) {
	var z Zone
	if err := json.Unmarshal([]byte(`{"kind":"zone","id":"Asia/Tokyo"}`), &z); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if z.ID() != "Asia/Tokyo" {
		t.Errorf("got %q, want %q", z.ID(), "Asia/Tokyo")
	}
}

func TestZoneJSONRoundTrip(t *testing.T) {
	orig := UTC
	b, _ := json.Marshal(orig)
	var got Zone
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if !orig.Equal(got) {
		t.Errorf("round-trip mismatch: got %v, want %v", got, orig)
	}
}

func TestZoneUnmarshalJSON_Invalid(t *testing.T) {
	var z Zone
	err := json.Unmarshal([]byte(`{"kind":"zone","id":"XYZ/Nowhere"}`), &z)
	if err == nil {
		t.Error("expected error for invalid zone, got nil")
	}
}
