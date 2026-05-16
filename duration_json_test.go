package gotime

import (
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
	if err := json.Unmarshal([]byte(`{"kind":"duration","iso":"PT1H30M","parts":{"hours":1,"minutes":30}}`), &d); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if d.InHours() != 1.5 {
		t.Errorf("got %v hours, want 1.5", d.InHours())
	}
}

func TestDurationUnmarshalJSON_Zero(t *testing.T) {
	var d Duration
	if err := json.Unmarshal([]byte(`{"kind":"duration","iso":"PT0S","parts":{}}`), &d); err != nil {
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
	orig := (90 * Minute)
	b, _ := json.Marshal(orig)
	var got Duration
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if orig.Nanoseconds() != got.Nanoseconds() {
		t.Errorf("round-trip mismatch: got %v, want %v", got.ISO8601(), orig.ISO8601())
	}
}

func TestDurationUnmarshalJSON_Invalid(t *testing.T) {
	var d Duration
	err := json.Unmarshal([]byte(`{"kind":"duration","iso":"not-iso"}`), &d)
	if err == nil {
		t.Error("expected error for invalid duration, got nil")
	}
}
