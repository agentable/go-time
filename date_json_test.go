package gotime

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/go-json-experiment/json"
)

func TestDateMarshalJSON(t *testing.T) {
	d := mustDate(2026, 3, 27)
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	want := `{"kind":"date","value":"2026-03-27"}`
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
	for _, orig := range []Date{
		mustDate(0, time.January, 1),
		mustDate(2026, time.March, 27),
		mustDate(9999, time.December, 31),
	} {
		b, err := json.Marshal(orig)
		if err != nil {
			t.Fatalf("Marshal(%v) error = %v", orig, err)
		}
		var got Date
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("Unmarshal(%s) error = %v", b, err)
		}
		if !orig.Equal(got) {
			t.Errorf("round-trip mismatch: got %v, want %v", got, orig)
		}
		again, err := json.Marshal(got)
		if err != nil || !bytes.Equal(again, b) {
			t.Fatalf("second Marshal(%v) = %s, %v; want %s", got, again, err, b)
		}
	}
}

func TestDateMarshalJSON_RejectsInvalidValue(t *testing.T) {
	_, err := json.Marshal(Date{})
	if !errors.Is(err, ErrInvalidDate) {
		t.Fatalf("Marshal(Date{}) error = %v, want ErrInvalidDate", err)
	}
	var te *TimeError
	if !errors.As(err, &te) || te.Hint == "" {
		t.Fatalf("Marshal(Date{}) error = %#v, want TimeError with hint", err)
	}
}

func TestDateUnmarshalJSON_InvalidValue(t *testing.T) {
	var d Date
	err := json.Unmarshal([]byte(`{"kind":"date","value":"not-a-date"}`), &d)
	if err == nil {
		t.Error("expected error for invalid date value, got nil")
	}
}

func TestNewDateRejectsYearsOutsideWireDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		year int
	}{
		{name: "negative", year: -1},
		{name: "too large", year: 10000},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewDate(tc.year, time.January, 1)
			if !errors.Is(err, ErrInvalidDate) {
				t.Fatalf("NewDate(%d, January, 1) error = %v, want ErrInvalidDate", tc.year, err)
			}
		})
	}
}
