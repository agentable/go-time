package zone

import (
	"strings"
	"time"
)

// ResolveLocation resolves an IANA name, case-insensitive IANA name, Windows
// timezone name into a canonical ID and location.
func ResolveLocation(id string) (string, *time.Location, bool) {
	if id == "" {
		return "", nil, false
	}

	if loc, err := time.LoadLocation(id); err == nil {
		if canonical, ok := findCanonicalCase(id); ok {
			if canonicalLoc, loadErr := time.LoadLocation(canonical); loadErr == nil {
				return canonical, canonicalLoc, true
			}
		}
		return id, loc, true
	}

	if canonical, ok := findCanonicalCase(id); ok {
		if loc, err := time.LoadLocation(canonical); err == nil {
			return canonical, loc, true
		}
	}

	if iana, ok := WindowsToIANA[id]; ok {
		if loc, err := time.LoadLocation(iana); err == nil {
			return iana, loc, true
		}
	}

	return "", nil, false
}

func findCanonicalCase(id string) (string, bool) {
	for _, candidate := range Zones {
		if strings.EqualFold(candidate, id) {
			return candidate, true
		}
	}
	return "", false
}
