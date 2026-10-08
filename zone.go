package gotime

import (
	"fmt"
	"slices"
	"time"

	"encoding/json/v2"

	ianazone "github.com/agentable/go-time/internal/zone"
)

var utcZone = Zone{id: "UTC"}

// UTC is the named UTC value. Treat it as read-only; assigning to it does not
// configure package defaults or change zero-Zone semantics.
var UTC = utcZone

// Zone represents an IANA timezone identity.
type Zone struct {
	id    string
	rules *ianazone.Rules
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

// ID returns the IANA timezone identifier.
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

// Equal reports whether two zones have the same identifier.
func (z Zone) Equal(other Zone) bool { return z.ID() == other.ID() }

// IsZero reports whether z is the Go zero value.
func (z Zone) IsZero() bool { return z.id == "" && z.rules == nil }

// MarshalJSON encodes z as {"kind":"zone","id":"<IANA id>"}.
// The output is deterministic and never depends on time.Now().
func (z Zone) MarshalJSON() ([]byte, error) {
	z = normalizeZone(z)
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
