package gotime

import (
	"bytes"
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

func TestZoneMarshalJSON_ZeroNormalizesToUTC(t *testing.T) {
	b, err := json.Marshal(Zone{})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	want := `{"kind":"zone","id":"UTC"}`
	if string(b) != want {
		t.Errorf("Marshal(Zone{}) = %s, want %s", b, want)
	}
}

func TestZoneMarshalJSON_FixedOffsetRejectsZoneWire(t *testing.T) {
	z := Zone{id: "+08:00", loc: time.FixedZone("+08:00", 8*3600)}
	_, err := json.Marshal(z)
	if !errors.Is(err, ErrInvalidZone) {
		t.Fatalf("Marshal error = %v, want ErrInvalidZone", err)
	}
}

func TestZoneMarshalJSON_RejectsUnknownPrivateID(t *testing.T) {
	z := Zone{id: "Mars/Olympus", loc: time.UTC}
	_, err := json.Marshal(z)
	if !errors.Is(err, ErrInvalidZone) {
		t.Fatalf("Marshal error = %v, want ErrInvalidZone", err)
	}
	var te *TimeError
	if !errors.As(err, &te) || te.Hint == "" {
		t.Fatalf("Marshal error = %#v, want TimeError with hint", err)
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
	tests := []struct {
		name string
		orig Zone
	}{
		{name: "named IANA zone", orig: MustLoadZone("Asia/Tokyo")},
		{name: "UTC", orig: UTC},
		{name: "zero zone", orig: Zone{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.orig)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			var got Zone
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("Unmarshal(%s) error = %v", b, err)
			}
			if !tc.orig.Equal(got) {
				t.Errorf("round-trip mismatch: got %v, want %v", got, tc.orig)
			}
			again, err := json.Marshal(got)
			if err != nil || !bytes.Equal(again, b) {
				t.Fatalf("second Marshal(%v) = %s, %v; want %s", got, again, err, b)
			}
		})
	}
}

func TestZoneUnmarshalJSON_Invalid(t *testing.T) {
	var z Zone
	err := json.Unmarshal([]byte(`{"kind":"zone","id":"XYZ/Nowhere"}`), &z)
	if err == nil {
		t.Error("expected error for invalid zone, got nil")
	}
}
