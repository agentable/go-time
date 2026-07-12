package gotime

import (
	"bytes"
	"errors"
	"testing"

	"github.com/go-json-experiment/json"
)

func TestTimeMarshalJSON(t *testing.T) {
	tm := mustTime(13, 30, 45)
	b, err := json.Marshal(tm)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	want := `{"kind":"time","value":"13:30:45"}`
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
	const want = `{"kind":"time","value":"13:30:45.123456789"}`
	if string(b) != want {
		t.Errorf("Marshal() = %s, want %s", b, want)
	}
}

func TestTimeMarshalJSON_MillisecondPrecision(t *testing.T) {
	tm := mustTimeNanos(13, 30, 45, 500_000_000) // 500ms
	b, err := json.Marshal(tm)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	const want = `{"kind":"time","value":"13:30:45.5"}`
	if string(b) != want {
		t.Errorf("Marshal() = %s, want %s", b, want)
	}
}

func TestTimeMarshalJSON_MicrosecondPrecision(t *testing.T) {
	tm := mustTimeNanos(13, 30, 45, 500_000) // 500µs
	b, err := json.Marshal(tm)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	const want = `{"kind":"time","value":"13:30:45.0005"}`
	if string(b) != want {
		t.Errorf("Marshal() = %s, want %s", b, want)
	}
}

func TestTimeUnmarshalJSON(t *testing.T) {
	var tm Time
	if err := json.Unmarshal([]byte(`{"kind":"time","value":"13:30:45"}`), &tm); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if tm.Hour() != 13 || tm.Minute() != 30 || tm.Second() != 45 {
		t.Errorf("got %v, want 13:30:45", tm)
	}
}

func TestTimeJSONRoundTrip(t *testing.T) {
	orig := mustTime(9, 0, 0)
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var got Time
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if !orig.Equal(got) {
		t.Errorf("round-trip mismatch: got %v, want %v", got, orig)
	}
	again, err := json.Marshal(got)
	if err != nil || !bytes.Equal(again, b) {
		t.Fatalf("second Marshal() = %s, %v; want %s", again, err, b)
	}
}

func TestTimeJSONRoundTrip_WithNanoseconds(t *testing.T) {
	orig := mustTimeNanos(13, 30, 45, 123456789)
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var got Time
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if !orig.Equal(got) {
		t.Errorf("round-trip mismatch: got %v, want %v", got, orig)
	}
	again, err := json.Marshal(got)
	if err != nil || !bytes.Equal(again, b) {
		t.Fatalf("second Marshal() = %s, %v; want %s", again, err, b)
	}
}

func TestTimeMarshalJSON_RejectsInvalidValue(t *testing.T) {
	_, err := json.Marshal(Time{hour: 24})
	if !errors.Is(err, ErrInvalidTime) {
		t.Fatalf("Marshal(invalid Time) error = %v, want ErrInvalidTime", err)
	}
	var te *TimeError
	if !errors.As(err, &te) || te.Hint == "" {
		t.Fatalf("Marshal(invalid Time) error = %#v, want TimeError with hint", err)
	}
}

func TestTimeUnmarshalJSON_RejectsNonCanonicalValue(t *testing.T) {
	var tm Time
	err := json.Unmarshal([]byte(`{"kind":"time","value":"13:30:45.500000000"}`), &tm)
	if !errors.Is(err, ErrInvalidFormat) {
		t.Fatalf("Unmarshal(non-canonical Time) error = %v, want ErrInvalidFormat", err)
	}
}
