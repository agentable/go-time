package zone

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var reFixedOffset = regexp.MustCompile(`^([+-])(\d{2}):?(\d{2})$`)
var reUTCOffset = regexp.MustCompile(`^(?i)UTC([+-])(\d{1,2})$`)

// ResolveLocation resolves an IANA name, case-insensitive IANA name, Windows
// timezone name, or fixed offset string into a canonical ID and location.
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

	if canonical, loc, ok := resolveFixedOffsetLocation(id); ok {
		return canonical, loc, true
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

func resolveFixedOffsetLocation(id string) (string, *time.Location, bool) {
	if m := reFixedOffset.FindStringSubmatch(id); m != nil {
		sign := 1
		if m[1] == "-" {
			sign = -1
		}
		h, _ := strconv.Atoi(m[2])
		min, _ := strconv.Atoi(m[3])
		if h > 23 || min > 59 {
			return "", nil, false
		}
		offsetSec := sign * (h*3600 + min*60)
		name := formatOffset(offsetSec)
		return name, time.FixedZone(name, offsetSec), true
	}

	if m := reUTCOffset.FindStringSubmatch(id); m != nil {
		sign := 1
		if m[1] == "-" {
			sign = -1
		}
		h, _ := strconv.Atoi(m[2])
		if h > 23 {
			return "", nil, false
		}
		offsetSec := sign * h * 3600
		name := formatOffset(offsetSec)
		return name, time.FixedZone(name, offsetSec), true
	}

	return "", nil, false
}

func formatOffset(offsetSec int) string {
	sign := "+"
	if offsetSec < 0 {
		sign = "-"
		offsetSec = -offsetSec
	}
	h := offsetSec / 3600
	m := (offsetSec % 3600) / 60
	return sign + twoDigits(h) + ":" + twoDigits(m)
}

func twoDigits(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}
