package gotime

import (
	"testing"

	"github.com/go-json-experiment/json"
)

func TestDateMarshalJSON(t *testing.T) {
	d := mustDate(2026, 3, 27)
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	want := `{"kind":"date","value":"2026-03-27","calendar":"iso8601"}`
	if string(b) != want {
		t.Errorf("got %s, want %s", b, want)
	}
}

func TestDateUnmarshalJSON(t *testing.T) {
	var d Date
	if err := json.Unmarshal([]byte(`{"kind":"date","value":"2026-03-27"}`), &d); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	want := mustDate(2026, 3, 27)
	if !d.Equal(want) {
		t.Errorf("got %v, want %v", d, want)
	}
}

func TestDateJSONRoundTrip(t *testing.T) {
	orig := mustDate(2026, 3, 27)
	b, _ := json.Marshal(orig)
	var got Date
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("round-trip unmarshal: %v", err)
	}
	if !orig.Equal(got) {
		t.Errorf("round-trip mismatch: got %v, want %v", got, orig)
	}
}

func TestDateUnmarshalJSON_InvalidValue(t *testing.T) {
	var d Date
	err := json.Unmarshal([]byte(`{"kind":"date","value":"not-a-date"}`), &d)
	if err == nil {
		t.Error("expected error for invalid date value, got nil")
	}
}
