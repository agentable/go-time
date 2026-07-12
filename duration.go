package gotime

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-json-experiment/json"
)

// errInvalidISO8601Duration is returned when an ISO 8601 duration string cannot be parsed.
var errInvalidISO8601Duration = errors.New("invalid ISO 8601 duration")

// Duration represents an exact elapsed time with nanosecond precision.
// It is a typed alias for time.Duration so that const arithmetic works:
// callers write 5 * gotime.Minute exactly like stdlib.
//
// Use Duration for exact-time math (timers, spans, sampling intervals).
// Use Period for calendar-aware math (months, years, "next Monday").
// They are deliberately distinct types — the type system prevents mixing.
//
// Day is intentionally NOT a Duration constant. A calendar day is a Period
// concept (Days(n)) and crosses DST boundaries safely; an exact 24-hour span
// is 24 * Hour. Conflating them is a frequent source of bugs.
type Duration time.Duration

// Duration unit constants mirror time.Duration's constants and support
// const arithmetic such as 5 * gotime.Minute.
const (
	Nanosecond  Duration = 1
	Microsecond          = 1000 * Nanosecond
	Millisecond          = 1000 * Microsecond
	Second               = 1000 * Millisecond
	Minute               = 60 * Second
	Hour                 = 60 * Minute
)

// Std returns d as a stdlib time.Duration.
func (d Duration) Std() time.Duration { return time.Duration(d) }

// Nanoseconds returns d in nanoseconds.
func (d Duration) Nanoseconds() int64 { return int64(d) }

// Milliseconds returns d in whole milliseconds.
func (d Duration) Milliseconds() int64 { return int64(d) / int64(time.Millisecond) }

// InSeconds returns d in seconds as a float64.
func (d Duration) InSeconds() float64 { return float64(d) / float64(time.Second) }

// InMinutes returns d in minutes as a float64.
func (d Duration) InMinutes() float64 { return float64(d) / float64(time.Minute) }

// InHours returns d in hours as a float64.
func (d Duration) InHours() float64 { return float64(d) / float64(time.Hour) }

// IsZero reports whether d is zero.
func (d Duration) IsZero() bool { return d == 0 }

// IsNegative reports whether d is negative.
func (d Duration) IsNegative() bool { return d < 0 }

// Abs returns the absolute value of d.
func (d Duration) Abs() Duration {
	return Duration(time.Duration(d).Abs())
}

// String returns d in the same format as time.Duration.String() ("1h30m0s",
// "500ms"). Identical to stdlib so round-trip with time.Duration is exact and
// callers can rely on a single well-known Stringer contract.
func (d Duration) String() string { return time.Duration(d).String() }

// decomposeDuration breaks nanoseconds into sign, hours, minutes, seconds, and sub-second ns.
func decomposeDuration(ns int64) (neg bool, h, m, s, subsecNs int64) {
	h = ns / int64(time.Hour)
	ns %= int64(time.Hour)
	m = ns / int64(time.Minute)
	ns %= int64(time.Minute)
	s = ns / int64(time.Second)
	subsecNs = ns % int64(time.Second)
	if h < 0 || m < 0 || s < 0 || subsecNs < 0 {
		neg = true
		h, m, s, subsecNs = -h, -m, -s, -subsecNs
	}
	return
}

// ISO8601 returns the ISO 8601 duration string, e.g. "PT1H30M", "-PT30M", "PT0S".
// Sub-second precision uses fractional seconds.
func (d Duration) ISO8601() string {
	neg, h, m, s, ns := decomposeDuration(int64(d))
	if h == 0 && m == 0 && s == 0 && ns == 0 {
		return "PT0S"
	}
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	b.WriteString("PT")
	if h > 0 {
		fmt.Fprintf(&b, "%dH", h)
	}
	if m > 0 {
		fmt.Fprintf(&b, "%dM", m)
	}
	if s > 0 || ns > 0 {
		if ns == 0 {
			fmt.Fprintf(&b, "%dS", s)
		} else {
			fmt.Fprintf(&b, "%d.%sS", s, trimmedNanoseconds(ns))
		}
	}
	return b.String()
}

func trimmedNanoseconds(ns int64) string {
	frac := fmt.Sprintf("%09d", ns)
	return strings.TrimRight(frac, "0")
}

// isoDurationRe matches subset of ISO 8601 duration strings produced by ISO8601().
// Groups: (1)sign (2)hours (3)minutes (4)seconds.
var isoDurationRe = regexp.MustCompile(
	`^(-?)PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?$`,
)

// parseISO8601Duration parses the subset of ISO 8601 duration strings produced by ISO8601().
func parseISO8601Duration(s string) (Duration, error) {
	m := isoDurationRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("duration %q: %w", s, errInvalidISO8601Duration)
	}
	if m[2] == "" && m[3] == "" && m[4] == "" {
		return 0, fmt.Errorf("duration %q: %w", s, errInvalidISO8601Duration)
	}

	negative := m[1] == "-"
	limit := uint64(1<<63 - 1)
	if negative {
		limit++
	}
	var totalNs uint64
	for _, component := range []struct {
		raw  string
		name string
		unit time.Duration
	}{
		{m[2], "hour", time.Hour},
		{m[3], "minute", time.Minute},
		{m[4], "second", time.Second},
	} {
		if component.raw == "" {
			continue
		}
		ns, ok := parseDurationComponent(component.raw, component.unit)
		if !ok {
			return 0, fmt.Errorf("duration %s component %q: %w", component.name, component.raw, errInvalidISO8601Duration)
		}
		magnitude := uint64(ns) //nolint:gosec // parseDurationComponent returns only non-negative values.
		if magnitude > limit-totalNs {
			return 0, fmt.Errorf("duration %q overflows nanoseconds: %w", s, errInvalidISO8601Duration)
		}
		totalNs += magnitude
	}

	if !negative {
		return Duration(totalNs), nil
	}
	if totalNs == uint64(1)<<63 {
		return Duration(-1 << 63), nil
	}
	return Duration(-int64(totalNs)), nil
}

// Decompose breaks d into renderable slots: Hours, Minutes, Seconds,
// Milliseconds, Microseconds, and Nanoseconds. Days/Months/Years are not
// extracted — Duration is exact wall-clock-free time, and elevating 24h to
// "1 day" presumes a calendar relationship the type does not carry. Callers
// can roll Hours into Days themselves if desired.
//
// The sign of d is preserved across every non-zero slot; for example
// (-2*Hour - 30*Minute).Decompose() yields Hours=-2, Minutes=-30.
func (d Duration) Decompose() DurationComponents {
	// Go's integer / and % truncate toward zero, so each slot inherits the
	// sign of d without an explicit negation pass.
	ns := int64(d)
	h := ns / int64(time.Hour)
	ns %= int64(time.Hour)
	m := ns / int64(time.Minute)
	ns %= int64(time.Minute)
	s := ns / int64(time.Second)
	ns %= int64(time.Second)
	ms := ns / 1_000_000
	ns %= 1_000_000
	us := ns / 1_000
	ns %= 1_000
	return DurationComponents{
		Hours:        h,
		Minutes:      m,
		Seconds:      s,
		Milliseconds: ms,
		Microseconds: us,
		Nanoseconds:  ns,
	}
}

// MarshalJSON encodes d as {"kind":"duration","iso":"<ISO8601>"}.
// The ISO 8601 string is the single source of truth; callers that need
// structured slots can run d.Decompose() at the call site.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Kind string `json:"kind"`
		ISO  string `json:"iso"`
	}{Kind: "duration", ISO: d.ISO8601()})
}

// UnmarshalJSON decodes d from {"kind":"duration","iso":"<ISO8601>",...}.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var wire struct {
		Kind string `json:"kind"`
		ISO  string `json:"iso"`
	}
	if err := unmarshalJSONWire(b, &wire); err != nil {
		return err
	}
	if err := requireJSONKind("duration", wire.Kind, "duration"); err != nil {
		return err
	}
	if err := requireJSONString("duration", "iso", wire.ISO); err != nil {
		return err
	}
	parsed, err := parseISO8601Duration(wire.ISO)
	if err != nil {
		return fmt.Errorf("gotime: invalid duration iso %q: %w", wire.ISO, err)
	}
	*d = parsed
	return nil
}
