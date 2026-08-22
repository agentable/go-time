package gotime

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"encoding/json/v2"
)

// errInvalidISO8601Period is returned when an ISO 8601 period string cannot be parsed.
var errInvalidISO8601Period = errors.New("invalid ISO 8601 period")

// Period represents a calendar offset in years, months, and days.
// It is the calendar-aware counterpart to Duration: Period operations
// preserve wall-clock time across DST transitions and apply
// end-of-month clamping for month/year arithmetic.
//
// Use Period for "next month", "in 7 days", recurring schedules.
// Use Duration for exact-time math.
//
// Fields are exported so callers can construct via struct literal:
//
//	p := gotime.Period{Years: 1, Months: 3, Days: 7}
type Period struct {
	// Years is the calendar-year offset.
	Years int32 `json:"years,omitzero"`
	// Months is the calendar-month offset.
	Months int32 `json:"months,omitzero"`
	// Days is the calendar-day offset.
	Days int32 `json:"days,omitzero"`
}

// NewPeriod creates a Period from calendar year, month, and day components.
func NewPeriod(years, months, days int32) Period {
	return Period{Years: years, Months: months, Days: days}
}

// Years returns Period{Years: n}. Sugar for the struct literal.
func Years(n int32) Period { return Period{Years: n} }

// Months returns Period{Months: n}. Sugar for the struct literal.
func Months(n int32) Period { return Period{Months: n} }

// Days returns Period{Days: n}. These are calendar days (not 24-hour spans);
// they preserve wall-clock time across DST boundaries. For exact 24-hour math
// write 24 * gotime.Hour.
func Days(n int32) Period { return Period{Days: n} }

// IsZero reports whether p has no calendar offset.
func (p Period) IsZero() bool { return p.Years == 0 && p.Months == 0 && p.Days == 0 }

// IsNegative reports whether any field of p is negative.
// A Period with mixed signs is not normalized — use Negate to flip all fields.
func (p Period) IsNegative() bool {
	return p.Years < 0 || p.Months < 0 || p.Days < 0
}

// Negate returns -p (all fields flipped), or ErrOverflow when a component is math.MinInt32.
func (p Period) Negate() (Period, error) {
	if p.Years == math.MinInt32 || p.Months == math.MinInt32 || p.Days == math.MinInt32 {
		return Period{}, newTimeError(
			ErrOverflow,
			"period negation overflows int32",
			fmt.Sprintf("years=%d months=%d days=%d", p.Years, p.Months, p.Days),
			"use smaller period components before negating",
		)
	}
	return Period{Years: -p.Years, Months: -p.Months, Days: -p.Days}, nil
}

// Abs returns p with all fields made non-negative, or ErrOverflow when a component is math.MinInt32.
func (p Period) Abs() (Period, error) {
	if p.Years == math.MinInt32 || p.Months == math.MinInt32 || p.Days == math.MinInt32 {
		return Period{}, newTimeError(
			ErrOverflow,
			"period absolute value overflows int32",
			fmt.Sprintf("years=%d months=%d days=%d", p.Years, p.Months, p.Days),
			"use smaller period components before taking the absolute value",
		)
	}
	return Period{Years: int32Magnitude(p.Years), Months: int32Magnitude(p.Months), Days: int32Magnitude(p.Days)}, nil
}

// Add returns p + other componentwise, or ErrOverflow when a component exceeds int32.
func (p Period) Add(other Period) (Period, error) {
	result, ok := periodFromInt64(
		int64(p.Years)+int64(other.Years),
		int64(p.Months)+int64(other.Months),
		int64(p.Days)+int64(other.Days),
	)
	if !ok {
		return Period{}, newTimeError(
			ErrOverflow,
			"period addition overflows int32",
			fmt.Sprintf("left=%s right=%s", p.ISO8601(), other.ISO8601()),
			"use smaller period components before adding",
		)
	}
	return result, nil
}

// Sub returns p - other componentwise, or ErrOverflow when a component exceeds int32.
func (p Period) Sub(other Period) (Period, error) {
	result, ok := periodFromInt64(
		int64(p.Years)-int64(other.Years),
		int64(p.Months)-int64(other.Months),
		int64(p.Days)-int64(other.Days),
	)
	if !ok {
		return Period{}, newTimeError(
			ErrOverflow,
			"period subtraction overflows int32",
			fmt.Sprintf("left=%s right=%s", p.ISO8601(), other.ISO8601()),
			"use smaller period components before subtracting",
		)
	}
	return result, nil
}

// ISO8601 returns the ISO 8601 representation of p, e.g. "P1Y3M7D".
// A zero Period returns "P0D". Negative components produce a leading "-".
func (p Period) ISO8601() string {
	if p.IsZero() {
		return "P0D"
	}
	neg := p.Years < 0 || p.Months < 0 || p.Days < 0
	mixed := neg && (p.Years > 0 || p.Months > 0 || p.Days > 0)

	var b strings.Builder
	if neg && !mixed {
		b.WriteByte('-')
	}
	b.WriteByte('P')
	if mixed {
		// Mixed signs: render each component with its own sign.
		writeSigned(&b, p.Years, 'Y')
		writeSigned(&b, p.Months, 'M')
		writeSigned(&b, p.Days, 'D')
		return b.String()
	}
	if p.Years != 0 {
		fmt.Fprintf(&b, "%dY", int64Magnitude(p.Years))
	}
	if p.Months != 0 {
		fmt.Fprintf(&b, "%dM", int64Magnitude(p.Months))
	}
	if p.Days != 0 {
		fmt.Fprintf(&b, "%dD", int64Magnitude(p.Days))
	}
	return b.String()
}

// String returns the canonical ISO 8601 representation.
func (p Period) String() string {
	return p.ISO8601()
}

// MarshalJSON encodes p as {"kind":"period","iso":"<ISO8601>"}.
// The ISO 8601 string is the single source of truth; callers that need
// structured slots can read p.Years / p.Months / p.Days directly.
func (p Period) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Kind string `json:"kind"`
		ISO  string `json:"iso"`
	}{
		Kind: "period",
		ISO:  p.ISO8601(),
	})
}

// UnmarshalJSON decodes p from {"kind":"period","iso":"<ISO8601>",...}.
func (p *Period) UnmarshalJSON(b []byte) error {
	var wire struct {
		Kind string `json:"kind"`
		ISO  string `json:"iso"`
	}
	if err := unmarshalJSONWire(b, &wire); err != nil {
		return err
	}
	if err := requireJSONKind("period", wire.Kind, "period"); err != nil {
		return err
	}
	if err := requireJSONString("period", "iso", wire.ISO); err != nil {
		return err
	}
	parsed, err := parseISO8601Period(wire.ISO)
	if err != nil {
		return newTimeErrorWithCause(
			ErrInvalidPeriod,
			err,
			"period iso is not a valid calendar period",
			wire.ISO,
			"use an ISO 8601 date period such as P1Y2M3D",
		)
	}
	*p = parsed
	return nil
}

// isoPeriodRe matches ISO 8601 period strings.
// Groups: (1)sign (2)years (3)months (4)weeks (5)days
var isoPeriodRe = regexp.MustCompile(
	`^(-?)P(?:([+-]?\d+)Y)?(?:([+-]?\d+)M)?(?:([+-]?\d+)W)?(?:([+-]?\d+)D)?$`,
)

// parseISO8601Period parses ISO 8601 period strings (date-only portion).
func parseISO8601Period(s string) (Period, error) {
	m := isoPeriodRe.FindStringSubmatch(s)
	if m == nil {
		return Period{}, fmt.Errorf("period %q: %w", s, errInvalidISO8601Period)
	}
	if m[2] == "" && m[3] == "" && m[4] == "" && m[5] == "" {
		return Period{}, fmt.Errorf("period %q: %w", s, errInvalidISO8601Period)
	}
	neg := m[1] == "-"
	if neg && hasSignedPeriodComponent(m[2:6]) {
		return Period{}, fmt.Errorf("period %q combines a leading sign with component signs: %w", s, errInvalidISO8601Period)
	}
	y, ok := parsePeriodWireComponent(m[2], neg)
	if !ok {
		return Period{}, fmt.Errorf("period years component %q: %w", m[2], errInvalidISO8601Period)
	}
	mo, ok := parsePeriodWireComponent(m[3], neg)
	if !ok {
		return Period{}, fmt.Errorf("period months component %q: %w", m[3], errInvalidISO8601Period)
	}
	w, ok := parsePeriodWireMagnitude(m[4])
	if !ok {
		return Period{}, fmt.Errorf("period weeks component %q: %w", m[4], errInvalidISO8601Period)
	}
	d, ok := parsePeriodWireMagnitude(m[5])
	if !ok {
		return Period{}, fmt.Errorf("period days component %q: %w", m[5], errInvalidISO8601Period)
	}
	totalDays := w*7 + d
	if neg {
		totalDays = -totalDays
	}
	if totalDays > math.MaxInt32 || totalDays < math.MinInt32 {
		return Period{}, fmt.Errorf("period days component overflows int32: %w", errInvalidISO8601Period)
	}
	return Period{Years: y, Months: mo, Days: int32(totalDays)}, nil
}

func hasSignedPeriodComponent(components []string) bool {
	for _, component := range components {
		if strings.HasPrefix(component, "+") || strings.HasPrefix(component, "-") {
			return true
		}
	}
	return false
}

func parsePeriodWireComponent(s string, negative bool) (int32, bool) {
	n, ok := parsePeriodWireMagnitude(s)
	if !ok {
		return 0, false
	}
	if negative {
		n = -n
	}
	if n < math.MinInt32 || n > math.MaxInt32 {
		return 0, false
	}
	return int32(n), true
}

func parsePeriodWireMagnitude(s string) (int64, bool) {
	if s == "" {
		return 0, true
	}
	if strings.ContainsAny(s, ".,") {
		return 0, false
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < math.MinInt32 || n > int64(math.MaxInt32)+1 {
		return 0, false
	}
	return n, true
}

func int32Magnitude(n int32) int32 {
	if n < 0 {
		return -n
	}
	return n
}

func int64Magnitude(n int32) int64 {
	v := int64(n)
	if v < 0 {
		return -v
	}
	return v
}

func writeSigned(b *strings.Builder, n int32, suffix byte) {
	if n == 0 {
		return
	}
	fmt.Fprintf(b, "%+d%c", n, suffix)
}

func periodFromInt64(years, months, days int64) (Period, bool) {
	if years < math.MinInt32 || years > math.MaxInt32 ||
		months < math.MinInt32 || months > math.MaxInt32 ||
		days < math.MinInt32 || days > math.MaxInt32 {
		return Period{}, false
	}
	return Period{Years: int32(years), Months: int32(months), Days: int32(days)}, true
}
