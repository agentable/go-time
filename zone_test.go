package gotime

import (
	"errors"
	"testing"
	"time"
)

func TestUTC(t *testing.T) {
	if UTC.ID() != "UTC" {
		t.Errorf("UTC.ID() = %q, want %q", UTC.ID(), "UTC")
	}
	if UTC.Location() == nil {
		t.Error("UTC.Location() must not be nil")
	}
}

func TestLocal(t *testing.T) {
	if UTC.ID() == "" {
		t.Error("Local.ID() must not be empty")
	}
	if Local.Location() == nil {
		t.Error("Local.Location() must not be nil")
	}
}

func TestZone_Equal(t *testing.T) {
	z1 := MustLoadZone("Asia/Tokyo")
	z2 := MustLoadZone("Asia/Tokyo")
	if !z1.Equal(z2) {
		t.Error("same zones should be equal")
	}

	z3 := MustLoadZone("America/New_York")
	if z1.Equal(z3) {
		t.Error("different zones should not be equal")
	}

	utcCopy := UTC
	if !UTC.Equal(utcCopy) {
		t.Error("UTC should equal itself")
	}
}

func TestZone_String(t *testing.T) {
	z := MustLoadZone("Asia/Tokyo")
	if z.String() != "Asia/Tokyo" {
		t.Errorf("String() = %q, want %q", z.String(), "Asia/Tokyo")
	}
}

// Phase 1: Zone query methods

var (
	summerInstant = InstantFromTime(time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC))
	winterInstant = InstantFromTime(time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC))
)

func TestZone_OffsetAt_Tokyo(t *testing.T) {
	z := MustLoadZone("Asia/Tokyo")
	if got := z.OffsetAt(summerInstant); got != "+09:00" {
		t.Errorf("OffsetAt = %q, want +09:00", got)
	}
}

func TestZone_OffsetAt_NewYork_Summer(t *testing.T) {
	z := MustLoadZone("America/New_York")
	if got := z.OffsetAt(summerInstant); got != "-04:00" {
		t.Errorf("OffsetAt = %q, want -04:00", got)
	}
}

func TestZone_OffsetAt_NewYork_Winter(t *testing.T) {
	z := MustLoadZone("America/New_York")
	if got := z.OffsetAt(winterInstant); got != "-05:00" {
		t.Errorf("OffsetAt = %q, want -05:00", got)
	}
}

func TestZone_OffsetAt_UTC(t *testing.T) {
	if got := UTC.OffsetAt(summerInstant); got != "+00:00" {
		t.Errorf("UTC.OffsetAt = %q, want +00:00", got)
	}
}

func TestZone_Abbreviation_Tokyo(t *testing.T) {
	z := MustLoadZone("Asia/Tokyo")
	if got := z.Abbreviation(summerInstant); got != "JST" {
		t.Errorf("Abbreviation = %q, want JST", got)
	}
}

func TestZone_Abbreviation_NewYork_Summer(t *testing.T) {
	z := MustLoadZone("America/New_York")
	if got := z.Abbreviation(summerInstant); got != "EDT" {
		t.Errorf("Abbreviation = %q, want EDT", got)
	}
}

func TestZone_Abbreviation_NewYork_Winter(t *testing.T) {
	z := MustLoadZone("America/New_York")
	if got := z.Abbreviation(winterInstant); got != "EST" {
		t.Errorf("Abbreviation = %q, want EST", got)
	}
}

func TestZone_Abbreviation_UTC(t *testing.T) {
	if got := UTC.Abbreviation(summerInstant); got != "UTC" {
		t.Errorf("UTC.Abbreviation = %q, want UTC", got)
	}
}

func TestZone_ZeroValue_NoPanic(t *testing.T) {
	var z Zone
	if z.Location() != time.UTC {
		t.Errorf("zero Zone Location() = %v, want UTC", z.Location())
	}
	offset := z.OffsetAt(summerInstant)
	abbr := z.Abbreviation(summerInstant)
	if offset != "+00:00" {
		t.Errorf("zero Zone OffsetAt = %q, want +00:00", offset)
	}
	if abbr != "UTC" {
		t.Errorf("zero Zone Abbreviation = %q, want UTC", abbr)
	}
}

// Phase 2: LoadZone, MustLoadZone, ResolveZone, Zones

func TestLoadZone_Valid(t *testing.T) {
	z, err := LoadZone("Asia/Tokyo")
	if err != nil {
		t.Fatalf("LoadZone: unexpected error: %v", err)
	}
	if z.ID() != "Asia/Tokyo" {
		t.Errorf("ID() = %q, want Asia/Tokyo", z.ID())
	}
}

func TestLoadZone_EmptyID(t *testing.T) {
	_, err := LoadZone("")
	if err == nil {
		t.Error("LoadZone with empty id should return error")
	}
}

func TestLoadZone_InvalidID(t *testing.T) {
	_, err := LoadZone("XYZ/Nowhere")
	if err == nil {
		t.Fatal("LoadZone with invalid id should return error")
	}

	var timeErr *TimeError
	if !errors.As(err, &timeErr) {
		t.Fatalf("LoadZone invalid error type = %T, want *TimeError", err)
	}
	if timeErr.Code != CodeInvalidZone {
		t.Errorf("TimeError.Code = %q, want %q", timeErr.Code, CodeInvalidZone)
	}
	if timeErr.Input != "XYZ/Nowhere" {
		t.Errorf("TimeError.Input = %q, want %q", timeErr.Input, "XYZ/Nowhere")
	}
	if timeErr.Hint == "" {
		t.Error("TimeError.Hint must not be empty")
	}
}

func TestMustLoadZone_Valid(t *testing.T) {
	z := MustLoadZone("Asia/Tokyo")
	if z.ID() != "Asia/Tokyo" {
		t.Errorf("ID() = %q, want Asia/Tokyo", z.ID())
	}
}

func TestMustLoadZone_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustLoadZone with invalid id should panic")
		}
	}()
	MustLoadZone("XYZ/Nowhere")
}

func TestLoadZone_UTC_EqualsSentinel(t *testing.T) {
	z, err := LoadZone("UTC")
	if err != nil {
		t.Fatalf("LoadZone(UTC): %v", err)
	}
	if z.ID() != "UTC" {
		t.Errorf("ID() = %q, want UTC", z.ID())
	}
}

// Phase 3: Zones (catalog)

func TestZones_NonEmpty(t *testing.T) {
	zones := Zones()
	if len(zones) == 0 {
		t.Error("Zones() should return non-empty slice")
	}
}

func TestZones_ContainsKnownZones(t *testing.T) {
	known := []string{"America/New_York", "Asia/Tokyo", "Europe/London", "UTC"}
	zones := Zones()
	set := make(map[string]bool, len(zones))
	for _, z := range zones {
		set[z] = true
	}
	for _, k := range known {
		if !set[k] {
			t.Errorf("Zones() missing %q", k)
		}
	}
}

func TestZones_Sorted(t *testing.T) {
	zones := Zones()
	for i := range len(zones) - 1 {
		i++
		if zones[i] < zones[i-1] {
			t.Errorf("Zones() not sorted: %q before %q", zones[i-1], zones[i])
			break
		}
	}
}

func TestZones_NewSlice(t *testing.T) {
	z1 := Zones()
	z2 := Zones()
	if len(z1) > 0 {
		z1[0] = "MODIFIED"
	}
	if len(z2) > 0 && z2[0] == "MODIFIED" {
		t.Error("Zones() should return a new slice each call")
	}
}

func TestZoneCatalogVersion(t *testing.T) {
	if got := ZoneCatalogVersion(); got != "2025b" {
		t.Errorf("ZoneCatalogVersion() = %q, want 2025b", got)
	}
}

func TestZones_TracksCatalogVersion(t *testing.T) {
	zones := Zones()
	set := make(map[string]bool, len(zones))
	for _, z := range zones {
		set[z] = true
	}
	for _, want := range []string{"America/Coyhaique", "America/Ciudad_Juarez", "Pacific/Midway"} {
		if !set[want] {
			t.Errorf("Zones() missing %q from %s catalog", want, ZoneCatalogVersion())
		}
	}
}

// TestLoadZone_SpecExample verifies the spec usage pattern: MustLoadZone used with In.
func TestLoadZone_SpecExample(t *testing.T) {
	tokyo := MustLoadZone("Asia/Tokyo")
	ny := MustLoadZone("America/New_York")
	instant := Now()
	tokyoTime := instant.In(tokyo)
	nyTime := tokyoTime.In(ny)
	// Both should represent the same underlying UTC moment
	if !tokyoTime.Instant().Equal(nyTime.Instant()) {
		t.Error("zone conversion must preserve the underlying Instant")
	}
}

// Phase 5: ResolveZone — fuzzy timezone resolution

func TestResolveZone_IANA(t *testing.T) {
	z, err := ResolveZone("Asia/Tokyo")
	if err != nil {
		t.Fatalf("ResolveZone(IANA): %v", err)
	}
	if z.ID() != "Asia/Tokyo" {
		t.Errorf("ID() = %q, want Asia/Tokyo", z.ID())
	}
}

func TestResolveZone_CaseInsensitive(t *testing.T) {
	z, err := ResolveZone("asia/tokyo")
	if err != nil {
		t.Fatalf("ResolveZone(case-insensitive): %v", err)
	}
	if z.ID() != "Asia/Tokyo" {
		t.Errorf("ID() = %q, want Asia/Tokyo", z.ID())
	}
}

func TestResolveZone_Windows(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Eastern Standard Time", "America/New_York"},
		{"China Standard Time", "Asia/Shanghai"},
		{"Tokyo Standard Time", "Asia/Tokyo"},
		{"W. Europe Standard Time", "Europe/Berlin"},
		{"Pacific Standard Time", "America/Los_Angeles"},
	}
	for _, tc := range tests {
		z, err := ResolveZone(tc.input)
		if err != nil {
			t.Errorf("ResolveZone(%q): %v", tc.input, err)
			continue
		}
		if z.ID() != tc.want {
			t.Errorf("ResolveZone(%q) = %q, want %q", tc.input, z.ID(), tc.want)
		}
	}
}

func TestResolveZone_RejectsFixedOffset(t *testing.T) {
	tests := []string{"+08:00", "-05:00", "+0800", "-0530", "UTC+8", "UTC-5"}
	for _, tc := range tests {
		_, err := ResolveZone(tc)
		if !errors.Is(err, ErrInvalidZone) {
			t.Errorf("ResolveZone(%q) error = %v, want ErrInvalidZone", tc, err)
		}
	}
}

func TestResolveZone_Empty(t *testing.T) {
	_, err := ResolveZone("")
	if err == nil {
		t.Error("ResolveZone('') should return error")
	}
}

func TestResolveZone_Invalid(t *testing.T) {
	_, err := ResolveZone("Not A Real Timezone")
	if err == nil {
		t.Error("ResolveZone with invalid input should return error")
	}
}
