package gotime

import (
	"strings"
	"testing"

	"github.com/go-json-experiment/json"
)

func TestTimeMarshalJSON(t *testing.T) {
	tm := mustTime(13, 30, 45)
	b, err := json.Marshal(tm)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	want := `{"kind":"time","value":"13:30:45","precision":"second"}`
	if string(b) != want {
		t.Errorf("got %s, want %s", b, want)
	}
}

func TestTimeMarshalJSON_WithNanoseconds(t *testing.T) {
	tm := mustTimeNanos(13, 30, 45, 123456789)
	b, err := json.Marshal(tm)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if !strings.Contains(string(b), `"kind":"time"`) {
		t.Errorf("missing kind field: %s", b)
	}
	if !strings.Contains(string(b), "123456789") {
		t.Errorf("missing nanoseconds in output: %s", b)
	}
	if !strings.Contains(string(b), `"precision":"nanosecond"`) {
		t.Errorf("expected nanosecond precision: %s", b)
	}
}

func TestTimeMarshalJSON_MillisecondPrecision(t *testing.T) {
	tm := mustTimeNanos(13, 30, 45, 500_000_000) // 500ms
	b, err := json.Marshal(tm)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if !strings.Contains(string(b), `"precision":"millisecond"`) {
		t.Errorf("expected millisecond precision: %s", b)
	}
}

func TestTimeMarshalJSON_MicrosecondPrecision(t *testing.T) {
	tm := mustTimeNanos(13, 30, 45, 500_000) // 500µs
	b, err := json.Marshal(tm)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if !strings.Contains(string(b), `"precision":"microsecond"`) {
		t.Errorf("expected microsecond precision: %s", b)
	}
}

func TestTimeUnmarshalJSON(t *testing.T) {
	var tm Time
	if err := json.Unmarshal([]byte(`{"kind":"time","value":"13:30:45","precision":"second"}`), &tm); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if tm.Hour() != 13 || tm.Minute() != 30 || tm.Second() != 45 {
		t.Errorf("got %v, want 13:30:45", tm)
	}
}

func TestTimeJSONRoundTrip(t *testing.T) {
	orig := mustTime(9, 0, 0)
	b, _ := json.Marshal(orig)
	var got Time
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if !orig.Equal(got) {
		t.Errorf("round-trip mismatch: got %v, want %v", got, orig)
	}
}

func TestTimeJSONRoundTrip_WithNanoseconds(t *testing.T) {
	orig := mustTimeNanos(13, 30, 45, 123456789)
	b, _ := json.Marshal(orig)
	var got Time
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if !orig.Equal(got) {
		t.Errorf("round-trip mismatch: got %v, want %v", got, orig)
	}
}
