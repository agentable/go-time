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
	archive, err := bundledArchive()
	if err != nil {
		return "", false
	}
	for _, entry := range archive.File {
		if strings.EqualFold(entry.Name, id) {
			return entry.Name, true
		}
	}
	return "", false
}
