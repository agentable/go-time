package gotime

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/go-json-experiment/json"
)

func TestDurationMarshalJSON(t *testing.T) {
	d := Duration(float64(Hour) * 1.5)
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	want := `{"kind":"duration","iso":"PT1H30M"}`
	if string(b) != want {
		t.Errorf("got %s, want %s", b, want)
	}
}

func TestDurationMarshalJSON_Zero(t *testing.T) {
	d := Duration(0)
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	want := `{"kind":"duration","iso":"PT0S"}`
	if string(b) != want {
		t.Errorf("got %s, want %s", b, want)
	}
}

func TestDurationMarshalJSON_Negative(t *testing.T) {
	d := (-30 * Minute)
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if !strings.Contains(string(b), `"iso":"-PT30M"`) {
		t.Errorf("missing iso field: %s", b)
	}
}

func TestDurationUnmarshalJSON(t *testing.T) {
	var d Duration
	if err := json.Unmarshal([]byte(`{"kind":"duration","iso":"PT1H30M"}`), &d); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if d.InHours() != 1.5 {
		t.Errorf("got %v hours, want 1.5", d.InHours())
	}
}

func TestDurationUnmarshalJSON_Zero(t *testing.T) {
	var d Duration
	if err := json.Unmarshal([]byte(`{"kind":"duration","iso":"PT0S"}`), &d); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !d.IsZero() {
		t.Errorf("expected zero duration, got %v", d.ISO8601())
	}
}

func TestDurationUnmarshalJSON_Negative(t *testing.T) {
	var d Duration
	if err := json.Unmarshal([]byte(`{"kind":"duration","iso":"-PT30M"}`), &d); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if d.InMinutes() != -30 {
		t.Errorf("got %v minutes, want -30", d.InMinutes())
	}
}

func TestDurationJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		orig Duration
	}{
		{name: "whole units", orig: 90 * Minute},
		{name: "nanosecond", orig: Nanosecond},
		{name: "fractional second", orig: Second + Nanosecond},
		{name: "negative fractional second", orig: -Nanosecond},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.orig)
			if err != nil {
				t.Fatalf("Marshal(%v) error = %v", tc.orig, err)
			}
			var got Duration
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("Unmarshal(%s) error = %v", b, err)
			}
			again, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("Marshal(round-tripped %v) error = %v", got, err)
			}
			if !bytes.Equal(again, b) {
				t.Errorf("Marshal after round-trip = %s, want %s", again, b)
			}
			if got.Nanoseconds() != tc.orig.Nanoseconds() {
				t.Errorf("round-trip duration = %v, want %v", got.ISO8601(), tc.orig.ISO8601())
			}
		})
	}
}

func TestDurationUnmarshalJSON_Invalid(t *testing.T) {
	var d Duration
	err := json.Unmarshal([]byte(`{"kind":"duration","iso":"not-iso"}`), &d)
	if err == nil {
		t.Fatal("Unmarshal(invalid duration) error = nil, want error")
	}
	if !errors.Is(err, errInvalidISO8601Duration) {
		t.Errorf("Unmarshal(invalid duration) error = %v, want errInvalidISO8601Duration", err)
	}
}

func TestDurationUnmarshalJSON_InvalidComponents(t *testing.T) {
	tests := []struct {
		name string
		iso  string
	}{
		{name: "hour overflow", iso: "PT999999999999999999999H"},
		{name: "minute overflow", iso: "PT999999999999999999999M"},
		{name: "second exponent", iso: "PT1e3S"},
		{name: "negative second exponent", iso: "PT1E-9S"},
		{name: "second overflow", iso: "PT1e309S"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var d Duration
			err := json.Unmarshal([]byte(fmt.Sprintf(`{"kind":"duration","iso":%q}`, tc.iso)), &d)
			if err == nil {
				t.Fatalf("Unmarshal(%q) error = nil, want error", tc.iso)
			}
			if !errors.Is(err, errInvalidISO8601Duration) {
				t.Errorf("Unmarshal(%q) error = %v, want errInvalidISO8601Duration", tc.iso, err)
			}
		})
	}
}
