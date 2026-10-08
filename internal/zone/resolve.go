package zone

import (
	"strings"
)

// ResolveLocation resolves an IANA name, case-insensitive IANA name, Windows
// timezone name into a canonical ID and rule snapshot.
func ResolveLocation(id string) (string, *Rules, bool) {
	if id == "" || id == "Local" {
		return "", nil, false
	}

	if loc, err := Load(id); err == nil {
		if canonical, ok := findCanonicalCase(id); ok && canonical != id {
			if canonicalLoc, loadErr := Load(canonical); loadErr == nil {
				return canonical, canonicalLoc, true
			}
		}
		return id, loc, true
	}

	if canonical, ok := findCanonicalCase(id); ok {
		if loc, err := Load(canonical); err == nil {
			return canonical, loc, true
		}
	}

	if iana, ok := WindowsToIANA[id]; ok {
		if loc, err := Load(iana); err == nil {
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
