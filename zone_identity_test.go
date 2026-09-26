package gotime

import (
	"encoding/json/v2"
	"errors"
	"testing"
)

func TestZoneRejectsLocalIdentity(t *testing.T) {
	t.Parallel()
	for _, load := range []func(string) (Zone, error){LoadZone, ResolveZone} {
		_, err := load("Local")
		var detail *TimeError
		if !errors.Is(err, ErrInvalidZone) || !errors.As(err, &detail) || detail.Hint == "" {
			t.Errorf("Local: %v", err)
		}
	}
	z := MustLoadZone("Asia/Tokyo")
	original := z
	err := json.Unmarshal([]byte(`{"kind":"zone","id":"Local"}`), &z)
	if !errors.Is(err, ErrInvalidZone) || !z.Equal(original) {
		t.Errorf("zone decode: %v %v", z, err)
	}
	i, _ := ParseInstant("2026-01-01T00:00:00Z")
	dt, err := i.In(original)
	if err != nil {
		t.Fatal(err)
	}
	before := dt
	err = json.Unmarshal([]byte(`{"kind":"datetime","instant":"2026-01-01T00:00:00Z","zone":"Local"}`), &dt)
	if !errors.Is(err, ErrInvalidZone) || !dt.Equal(before) || !dt.Zone().Equal(before.Zone()) {
		t.Errorf("datetime decode: %v %v", dt, err)
	}
	for _, id := range []string{"UTC", "US/Eastern", "Etc/UTC"} {
		if _, err := LoadZone(id); err != nil {
			t.Errorf("%s: %v", id, err)
		}
	}
}
