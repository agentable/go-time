package gotime

import (
	"fmt"
	"slices"
	"time"

	"github.com/go-json-experiment/json"

	ianazone "github.com/agentable/go-time/internal/zone"
)

// UTC is the UTC timezone.
var UTC Zone

// Local is the process-local timezone reported by the host system.
var Local Zone

func init() {
	UTC = Zone{id: "UTC", loc: time.UTC}
	// Use whatever name time.Local reports — typically the IANA id resolved
	// from TZ or the system. Do not relabel "Local" to "UTC": that would
	// make Local.ID() lie about the underlying offset.
	Local = Zone{id: time.Local.String(), loc: time.Local}
}

// Zone represents an IANA timezone identity.
type Zone struct {
	id  string
	loc *time.Location
}

// LoadZone loads a Zone by IANA timezone id.
func LoadZone(id string) (Zone, error) {
	if id == "" {
		return Zone{}, newTimeError(
			ErrInvalidZone,
			"zone id must not be empty",
			"",
			"provide a non-empty IANA zone id like Asia/Tokyo",
		)
	}
	loc, err := time.LoadLocation(id)
	if err != nil {
		return Zone{}, newTimeError(
			ErrInvalidZone,
			fmt.Sprintf("unknown time zone: %s", id),
			id,
			"use IANA zone ids like Asia/Tokyo; call gotime.Zones() for all valid ids",
		)
	}
	return Zone{id: id, loc: loc}, nil
}

// MustLoadZone is like LoadZone but panics if id is invalid.
// It is intended for use with fixed IANA identifiers in variable initializations.
func MustLoadZone(id string) Zone {
	z, err := LoadZone(id)
	if err != nil {
		panic(fmt.Errorf("gotime.MustLoadZone: %w", err))
	}
	return z
}

// ID returns the IANA timezone identifier.
func (z Zone) ID() string { return z.id }

// Location returns the underlying *time.Location for stdlib interop.
// The zero Zone falls back to time.UTC.
func (z Zone) Location() *time.Location {
	if z.IsZero() {
		return time.UTC
	}
	return z.loc
}

func normalizeZone(z Zone) Zone {
	if z.IsZero() {
		return UTC
	}
	return z
}

// String returns the zone identifier.
func (z Zone) String() string { return z.id }

// Equal reports whether two zones have the same identifier.
func (z Zone) Equal(other Zone) bool { return z.id == other.id }

// IsZero reports whether z is the zero value (no zone explicitly set).
// Note that the zero Zone is still safe to operate on: Location falls back
// to time.UTC. Use IsZero only when you need to detect "was this set?".
func (z Zone) IsZero() bool { return z.id == "" && z.loc == nil }

// MarshalJSON encodes z as {"kind":"zone","id":"<IANA id>"}.
// The output is deterministic and never depends on time.Now() —
// time-dependent offset and abbreviation data lives in Zone.Snapshot(at).
func (z Zone) MarshalJSON() ([]byte, error) {
	if isFixedOffsetID(z.ID()) {
		return nil, newTimeError(
			ErrInvalidZone,
			"fixed UTC offsets are not IANA zones",
			z.ID(),
			"represent numeric offsets as instant syntax, not as zone identity",
		)
	}
	return json.Marshal(struct {
		Kind string `json:"kind"`
		ID   string `json:"id"`
	}{
		Kind: "zone",
		ID:   z.ID(),
	})
}

func isFixedOffsetID(id string) bool {
	_, err := parseOffsetLocation(id)
	return err == nil
}

// UnmarshalJSON decodes z from {"kind":"zone","id":"<IANA id>",...}.
func (z *Zone) UnmarshalJSON(b []byte) error {
	var wire struct {
		Kind string `json:"kind"`
		ID   string `json:"id"`
	}
	if err := unmarshalJSONWire(b, &wire); err != nil {
		return err
	}
	if err := requireJSONKind("zone", wire.Kind, "zone"); err != nil {
		return err
	}
	loaded, err := LoadZone(wire.ID)
	if err != nil {
		return fmt.Errorf("gotime: invalid zone id %q: %w", wire.ID, err)
	}
	*z = loaded
	return nil
}

// formatOffset formats an offset in seconds as "+HH:MM" or "-HH:MM".
func formatOffset(offsetSec int) string {
	sign := "+"
	if offsetSec < 0 {
		sign = "-"
		offsetSec = -offsetSec
	}
	h := offsetSec / 3600
	m := (offsetSec % 3600) / 60
	return fmt.Sprintf("%s%02d:%02d", sign, h, m)
}

// OffsetAt returns the UTC offset of the zone at the given Instant as a "+HH:MM" or "-HH:MM" string.
func (z Zone) OffsetAt(i Instant) string {
	_, offsetSec := i.Std().In(z.Location()).Zone()
	return formatOffset(offsetSec)
}

// Abbreviation returns the timezone abbreviation (e.g., "JST", "EDT") at the given Instant.
func (z Zone) Abbreviation(i Instant) string {
	abbr, _ := i.Std().In(z.Location()).Zone()
	return abbr
}

// ZoneSnapshot is a point-in-time snapshot of a Zone's display data.
// It is intentionally cheap and copy-safe — callers compute it on demand.
type ZoneSnapshot struct {
	// ID is the IANA zone identifier.
	ID string `json:"id"`
	// Offset is the UTC offset at the snapshot time, formatted as "+HH:MM".
	Offset string `json:"offset"`
	// Abbreviation is the zone abbreviation (e.g. "JST", "PDT").
	Abbreviation string `json:"abbreviation"`
}

// Snapshot returns a point-in-time view of z (offset and abbreviation) at i.
// The snapshot is decoupled from JSON serialization — callers requiring
// time-dependent fields compute and embed them explicitly.
func (z Zone) Snapshot(i Instant) ZoneSnapshot {
	return ZoneSnapshot{
		ID:           z.ID(),
		Offset:       z.OffsetAt(i),
		Abbreviation: z.Abbreviation(i),
	}
}

// ResolveZone resolves a timezone identifier by trying exact IANA names,
// case-insensitive IANA matches, and Windows timezone names.
// Legacy IANA aliases such as "US/Eastern" are handled by Go's time.LoadLocation.
func ResolveZone(id string) (Zone, error) {
	if id == "" {
		return Zone{}, newTimeError(
			ErrInvalidZone,
			"zone id must not be empty",
			"",
			"provide a non-empty IANA id or Windows zone name",
		)
	}
	if canonical, loc, ok := ianazone.ResolveLocation(id); ok {
		return Zone{id: canonical, loc: loc}, nil
	}
	return Zone{}, newTimeError(
		ErrInvalidZone,
		fmt.Sprintf("cannot resolve timezone: %s", id),
		id,
		"use a canonical IANA id (e.g. Asia/Tokyo); call gotime.Zones() for the full list",
	)
}

// Zones returns a sorted copy of all known IANA timezone identifiers.
func Zones() []string {
	return slices.Clone(ianazone.Zones)
}

// ZoneCatalogVersion returns the IANA tzdb version used to generate Zones.
// It does not describe the transition-rule data used by time.LoadLocation.
func ZoneCatalogVersion() string {
	return ianazone.CatalogVersion
}
