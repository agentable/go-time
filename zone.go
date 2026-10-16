package gotime

import (
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"encoding/json/v2"

	ianazone "github.com/agentable/go-time/internal/zone"
)

var utcZone = Zone{id: "UTC"}

// UTC is the named UTC value. Treat it as read-only; assigning to it does not
// configure package defaults or change zero-Zone semantics.
var UTC = utcZone

// Zone holds a named, immutable timezone rule snapshot.
type Zone struct {
	id     string
	rules  *ianazone.Rules
	custom bool
}

// LoadZone loads a Zone by IANA timezone id.
func LoadZone(id string) (Zone, error) {
	if id == "" || id == "Local" {
		return Zone{}, newTimeError(
			ErrInvalidZone,
			"zone id must be an explicit IANA identity",
			id,
			"provide a non-empty IANA zone id like Asia/Tokyo",
		)
	}
	rules, err := ianazone.Load(id)
	if err != nil {
		return Zone{}, newTimeErrorWithCause(
			ErrInvalidZone,
			err,
			fmt.Sprintf("unknown time zone: %s", id),
			id,
			"use IANA zone ids like Asia/Tokyo; call gotime.Zones() for the generated IANA catalog",
		)
	}
	return Zone{id: id, rules: rules}, nil
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

// ID returns the timezone identifier, which alone does not identify its rules.
func (z Zone) ID() string {
	if z.IsZero() {
		return "UTC"
	}
	return z.id
}

// Location returns the underlying *time.Location for stdlib interop.
// The zero Zone falls back to time.UTC.
func (z Zone) Location() *time.Location {
	return z.rules.Location()
}

func normalizeZone(z Zone) Zone {
	if z.IsZero() {
		return utcZone
	}
	return z
}

// String returns the zone identifier.
func (z Zone) String() string { return z.ID() }

// Equal compares the identifier, rule source kind, and exact TZif snapshot.
// It does not prove semantic equivalence between differently encoded rules.
func (z Zone) Equal(other Zone) bool {
	return z.ID() == other.ID() && z.custom == other.custom && z.rules.Equal(other.rules)
}

// IsZero reports whether z is the Go zero value.
func (z Zone) IsZero() bool { return z.id == "" && z.rules == nil }

// MarshalJSON encodes z as {"kind":"zone","id":"<IANA id>"}.
// The output is deterministic and never depends on time.Now().
func (z Zone) MarshalJSON() ([]byte, error) {
	z = normalizeZone(z)
	if err := z.checkJSON(); err != nil {
		return nil, err
	}
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

func (z Zone) checkJSON() error {
	if z.custom {
		return newTimeError(ErrInvalidZone, "JSON cannot preserve custom timezone rules", z.ID(), "retain the original rule data, or explicitly convert a DateTime to UTC")
	}
	return nil
}

func isFixedOffsetID(id string) bool {
	_, err := parseOffsetLocation(id)
	return err == nil
}

// UnmarshalJSON decodes z from {"kind":"zone","id":"<IANA id>",...} and
// replaces z. Callers must not read or write the same receiver concurrently.
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
	if err := requireJSONString("zone", "id", wire.ID); err != nil {
		return err
	}
	loaded, err := LoadZone(wire.ID)
	if err != nil {
		return err
	}
	*z = loaded
	return nil
}

// ResolveZone resolves a timezone identifier by trying exact IANA names,
// case-insensitive IANA matches, and Windows timezone names.
// Legacy IANA aliases such as "US/Eastern" are handled by the IANA rule sources.
func ResolveZone(id string) (Zone, error) {
	if id == "" || id == "Local" {
		return Zone{}, newTimeError(
			ErrInvalidZone,
			"zone id must be an explicit IANA identity",
			id,
			"provide a non-empty IANA id or Windows zone name",
		)
	}
	if canonical, rules, ok := ianazone.ResolveLocation(id); ok {
		return Zone{id: canonical, rules: rules}, nil
	}
	return Zone{}, newTimeError(
		ErrInvalidZone,
		fmt.Sprintf("cannot resolve timezone: %s", id),
		id,
		"use a canonical IANA id (e.g. Asia/Tokyo); call gotime.Zones() for the generated IANA catalog",
	)
}

// Zones returns a sorted caller-owned copy of the generated IANA catalog.
// LoadZone also accepts IANA links that are absent from this catalog.
func Zones() []string {
	return slices.Clone(ianazone.Zones)
}

// ZoneCatalogVersion returns the IANA tzdb version used to generate Zones.
// It does not describe the transition-rule data held by a Zone.
func ZoneCatalogVersion() string {
	return ianazone.CatalogVersion
}

// LoadZoneFromTZData loads an isolated rule snapshot from caller-supplied TZif
// data, limited to 10 MiB. It does not register the identifier for named lookup.
// The identifier must be nonblank UTF-8 without control characters and cannot
// be UTC, Local, or a numeric offset. The data must not change during the call;
// subsequent changes do not affect the returned Zone. Custom zones and their
// DateTimes cannot use the IANA-only JSON format and return ErrInvalidZone.
func LoadZoneFromTZData(id string, data []byte) (Zone, error) {
	if strings.TrimSpace(id) == "" || !utf8.ValidString(id) || strings.ContainsFunc(id, unicode.IsControl) ||
		id == "UTC" || id == "Local" || isFixedOffsetID(id) {
		return Zone{}, newTimeError(ErrInvalidZone, "invalid custom timezone identifier", id, "use a nonblank name without controls, distinct from UTC, Local, and numeric offsets")
	}
	rules, err := ianazone.LoadData(id, data)
	if err != nil {
		return Zone{}, newTimeErrorWithCause(ErrInvalidZone, err, "cannot load TZif rules", id, "supply valid TZif data")
	}
	return Zone{id: id, rules: rules, custom: true}, nil
}
