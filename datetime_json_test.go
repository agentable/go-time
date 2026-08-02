package gotime

import (
	"bytes"
	"errors"
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
	want := `{"kind":"datetime","instant":"2026-03-27T04:00:00Z","zone":"Asia/Tokyo"}`
	if string(b) != want {
		t.Errorf("got %s, want %s", b, want)
	}
}

func TestDateTimeUnmarshalJSON(t *testing.T) {
	var dt DateTime
	input := `{"kind":"datetime","instant":"2026-03-27T04:00:00Z","zone":"Asia/Tokyo"}`
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
	orig := mustDateTime(mustDate(2026, 3, 27), mustTimeNanos(13, 0, 0, 123_456_789), z)
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
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
	again, err := json.Marshal(got)
	if err != nil || !bytes.Equal(again, b) {
		t.Fatalf("second Marshal() = %s, %v; want %s", again, err, b)
	}
}

func TestDateTimeMarshalJSON_FixedOffsetRejectsZoneWire(t *testing.T) {
	loc := time.FixedZone("+09:00", 9*3600)
	dt := DateTime{
		t:    time.Date(2026, time.March, 27, 13, 0, 0, 0, loc),
		zone: Zone{id: "+09:00", loc: loc},
	}
	_, err := json.Marshal(dt)
	if !errors.Is(err, ErrInvalidZone) {
		t.Fatalf("Marshal error = %v, want ErrInvalidZone", err)
	}
}

func TestDateTimeMarshalJSON_UnknownPrivateZoneRejectsWire(t *testing.T) {
	dt := DateTime{
		t:    time.Date(2026, time.March, 27, 13, 0, 0, 0, time.UTC),
		zone: Zone{id: "Mars/Olympus", loc: time.UTC},
	}
	_, err := json.Marshal(dt)
	if !errors.Is(err, ErrInvalidZone) {
		t.Fatalf("Marshal error = %v, want ErrInvalidZone", err)
	}
}

func TestDateTimeMarshalJSON_ZeroZoneProjectionsUseUTC(t *testing.T) {
	t.Parallel()

	base := UnixSeconds(0)
	tests := []struct {
		name string
		dt   DateTime
	}{
		{name: "instant in zero zone", dt: mustInstantIn(t, base, Zone{})},
		{name: "datetime in zero zone", dt: mustDateTimeIn(t, mustInstantIn(t, base, MustLoadZone("Asia/Tokyo")), Zone{})},
		{name: "stdlib time in zero zone", dt: mustDateTimeFromTime(t, base.Std(), Zone{})},
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
			want := `{"kind":"datetime","instant":"1970-01-01T00:00:00Z","zone":"UTC"}`
			if string(b) != want {
				t.Fatalf("Marshal() = %s, want %s", b, want)
			}
		})
	}
}

func TestDateTimeUnmarshalJSON_NoZone(t *testing.T) {
	var dt DateTime
	input := `{"kind":"datetime","instant":"2026-03-27T04:00:00Z"}`
	err := json.Unmarshal([]byte(input), &dt)
	if !errors.Is(err, ErrInvalidFormat) {
		t.Fatalf("Unmarshal error = %v, want ErrInvalidFormat", err)
	}
}

func TestDateTimeUnmarshalJSON_ProjectsStoredInstantInZone(t *testing.T) {
	var dt DateTime
	input := `{"kind":"datetime","instant":"2026-03-27T04:00:00Z","zone":"Asia/Tokyo"}`
	if err := json.Unmarshal([]byte(input), &dt); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}
	if got := dt.String(); got != "2026-03-27T13:00:00+09:00" {
		t.Fatalf("Unmarshal projection = %s, want 2026-03-27T13:00:00+09:00", got)
	}
}

func TestDateTimeUnmarshalJSON_RejectsInvalidInstantOrZone(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  error
	}{
		{name: "invalid instant", input: `{"kind":"datetime","instant":"not-a-time","zone":"UTC"}`, want: ErrInvalidFormat},
		{name: "invalid zone", input: `{"kind":"datetime","instant":"2026-03-27T04:00:00Z","zone":"Mars/Olympus"}`, want: ErrInvalidZone},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var dt DateTime
			err := json.Unmarshal([]byte(tc.input), &dt)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Unmarshal(%s) error = %v, want %v", tc.input, err, tc.want)
			}
			var te *TimeError
			if !errors.As(err, &te) || te.Hint == "" {
				t.Fatalf("Unmarshal(%s) error = %#v, want TimeError with hint", tc.input, err)
			}
		})
	}
}

func TestDateTimeMarshalJSON_RejectsInstantOutsideWireDomain(t *testing.T) {
	dt := DateTime{t: time.Date(10_000, time.January, 1, 0, 0, 0, 0, time.UTC), zone: UTC}
	_, err := json.Marshal(dt)
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("Marshal(year 10000) error = %v, want ErrOverflow", err)
	}
	var te *TimeError
	if !errors.As(err, &te) || te.Hint == "" {
		t.Fatalf("Marshal(year 10000) error = %#v, want TimeError with hint", err)
	}
}

func TestDateTimeMarshalJSON_RejectsProjectedCivilYearOutsideWireDomain(t *testing.T) {
	z := MustLoadZone("Asia/Tokyo")
	dt := DateTime{
		t:    time.Date(10_000, time.January, 1, 0, 0, 0, 0, z.Location()),
		zone: z,
	}

	_, err := json.Marshal(dt)
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("Marshal(projected year 10000) error = %v, want ErrOverflow", err)
	}
	var te *TimeError
	if !errors.As(err, &te) || te.Hint == "" {
		t.Fatalf("Marshal(projected year 10000) error = %#v, want TimeError with hint", err)
	}
}
