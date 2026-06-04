package gotime

import (
	"testing"
	"time"

	"github.com/go-json-experiment/json"
)

func TestDateTimeMarshalJSON(t *testing.T) {
	z := MustLoadZone("Asia/Tokyo")
	d := mustDate(2026, 3, 27)
	tm := mustTime(13, 0, 0)
	dt := mustDateTime(d, tm, z)

	b, err := json.Marshal(dt)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	want := `{"kind":"datetime","value":"2026-03-27T13:00:00+09:00","zone":"Asia/Tokyo","calendar":"iso8601"}`
	if string(b) != want {
		t.Errorf("got %s, want %s", b, want)
	}
}

func TestDateTimeUnmarshalJSON(t *testing.T) {
	var dt DateTime
	input := `{"kind":"datetime","value":"2026-03-27T13:00:00+09:00","zone":"Asia/Tokyo"}`
	if err := json.Unmarshal([]byte(input), &dt); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if dt.Zone().ID() != "Asia/Tokyo" {
		t.Errorf("zone ID = %q, want %q", dt.Zone().ID(), "Asia/Tokyo")
	}
	if dt.Date().Year() != 2026 || dt.Date().Month() != time.March || dt.Date().Day() != 27 {
		t.Errorf("date = %v, want 2026-03-27", dt.Date())
	}
	if dt.Clock().Hour() != 13 {
		t.Errorf("hour = %d, want 13", dt.Clock().Hour())
	}
}

func TestDateTimeJSONRoundTrip(t *testing.T) {
	z := MustLoadZone("Asia/Tokyo")
	orig := mustDateTime(mustDate(2026, 3, 27), mustTime(13, 0, 0), z)
	b, _ := json.Marshal(orig)
	var got DateTime
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if !orig.Equal(got) {
		t.Errorf("instant mismatch: got %v, want %v", got, orig)
	}
	if got.Zone().ID() != orig.Zone().ID() {
		t.Errorf("zone mismatch: got %q, want %q", got.Zone().ID(), orig.Zone().ID())
	}
}

func TestDateTimeMarshalJSON_ZeroZoneProjectionsUseUTC(t *testing.T) {
	t.Parallel()

	base := UnixSeconds(0)
	tests := []struct {
		name string
		dt   DateTime
	}{
		{name: "instant in zero zone", dt: base.In(Zone{})},
		{name: "datetime in zero zone", dt: base.In(MustLoadZone("Asia/Tokyo")).In(Zone{})},
		{name: "stdlib time in zero zone", dt: DateTimeFromTime(base.Std(), Zone{})},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.dt.Zone().ID() != "UTC" {
				t.Fatalf("zone = %q, want UTC", tc.dt.Zone().ID())
			}
			b, err := json.Marshal(tc.dt)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}
			want := `{"kind":"datetime","value":"1970-01-01T00:00:00Z","zone":"UTC","calendar":"iso8601"}`
			if string(b) != want {
				t.Fatalf("Marshal() = %s, want %s", b, want)
			}
		})
	}
}

func TestDateTimeUnmarshalJSON_NoZone(t *testing.T) {
	// When zone field is absent, fall back to the offset embedded in the value.
	var dt DateTime
	input := `{"kind":"datetime","value":"2026-03-27T13:00:00+09:00"}`
	if err := json.Unmarshal([]byte(input), &dt); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if dt.Clock().Hour() != 13 {
		t.Errorf("hour = %d, want 13", dt.Clock().Hour())
	}
}
