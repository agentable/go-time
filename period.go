package gotime

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-json-experiment/json"
)

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
	Years  int32 `json:"years,omitzero"`
	Months int32 `json:"months,omitzero"`
	Days   int32 `json:"days,omitzero"`
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

// Negate returns -p (all fields flipped).
func (p Period) Negate() Period {
	return Period{Years: -p.Years, Months: -p.Months, Days: -p.Days}
}

// Abs returns p with all fields made non-negative.
func (p Period) Abs() Period {
	return Period{Years: absInt32(p.Years), Months: absInt32(p.Months), Days: absInt32(p.Days)}
}

// Add returns p + other (componentwise).
func (p Period) Add(other Period) Period {
	return Period{Years: p.Years + other.Years, Months: p.Months + other.Months, Days: p.Days + other.Days}
}

// Sub returns p - other (componentwise).
func (p Period) Sub(other Period) Period {
	return Period{Years: p.Years - other.Years, Months: p.Months - other.Months, Days: p.Days - other.Days}
}

// ISO8601 returns the ISO 8601 representation of p, e.g. "P1Y3M7D".
// A zero Period returns "P0D". Negative components produce a leading "-".
func (p Period) ISO8601() string {
	if p.IsZero() {
		return "P0D"
	}
	abs := p.Abs()
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
	if abs.Years != 0 {
		fmt.Fprintf(&b, "%dY", abs.Years)
	}
	if abs.Months != 0 {
		fmt.Fprintf(&b, "%dM", abs.Months)
	}
	if abs.Days != 0 {
		fmt.Fprintf(&b, "%dD", abs.Days)
	}
	return b.String()
}

// RFC5545 formats p per RFC 5545 §3.3.6. RFC 5545 disallows months/years
// in DURATION values, but Period intentionally carries calendar offsets;
// we still emit P{n}Y, P{n}M, P{n}D — callers responsible for validating
// the receiving system supports this. For weeks-aligned days, emit "P{n}W".
func (p Period) RFC5545() string {
	if p.Years == 0 && p.Months == 0 && p.Days != 0 && p.Days%7 == 0 {
		neg := p.Days < 0
		days := p.Days
		if neg {
			days = -days
		}
		var b strings.Builder
		if neg {
			b.WriteByte('-')
		}
		fmt.Fprintf(&b, "P%dW", days/7)
		return b.String()
	}
	return p.ISO8601()
}

// String returns a compact human form, e.g. "1y3mo7d", "-2mo".
func (p Period) String() string {
	if p.IsZero() {
		return "0d"
	}
	var b strings.Builder
	abs := p.Abs()
	allNeg := !p.IsZero() && p.Years <= 0 && p.Months <= 0 && p.Days <= 0
	if allNeg {
		b.WriteByte('-')
	}
	if abs.Years != 0 {
		fmt.Fprintf(&b, "%dy", abs.Years)
	}
	if abs.Months != 0 {
		fmt.Fprintf(&b, "%dmo", abs.Months)
	}
	if abs.Days != 0 {
		fmt.Fprintf(&b, "%dd", abs.Days)
	}
	return b.String()
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
	if err := json.Unmarshal(b, &wire); err != nil {
		return err
	}
	parsed, err := parseISO8601Period(wire.ISO)
	if err != nil {
		return fmt.Errorf("gotime: invalid period iso %q: %w", wire.ISO, err)
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
		return Period{}, fmt.Errorf("invalid ISO 8601 period: %q", s)
	}
	if m[2] == "" && m[3] == "" && m[4] == "" && m[5] == "" {
		return Period{}, fmt.Errorf("invalid ISO 8601 period: %q", s)
	}
	neg := m[1] == "-"
	y, ok := parsePeriodJSONComponent(m[2])
	if !ok {
		return Period{}, fmt.Errorf("invalid years component %q", m[2])
	}
	mo, ok := parsePeriodJSONComponent(m[3])
	if !ok {
		return Period{}, fmt.Errorf("invalid months component %q", m[3])
	}
	w, ok := parsePeriodJSONComponent(m[4])
	if !ok {
		return Period{}, fmt.Errorf("invalid weeks component %q", m[4])
	}
	d, ok := parsePeriodJSONComponent(m[5])
	if !ok {
		return Period{}, fmt.Errorf("invalid days component %q", m[5])
	}
	totalDays := int64(w)*7 + int64(d)
	if totalDays > maxInt32 || totalDays < -maxInt32-1 {
		return Period{}, fmt.Errorf("days component overflows int32")
	}
	if neg {
		y, mo, totalDays = -y, -mo, -totalDays
	}
	return Period{Years: y, Months: mo, Days: int32(totalDays)}, nil //nolint:gosec // checked above against int32 bounds
}

func parsePeriodJSONComponent(s string) (int32, bool) {
	if s == "" {
		return 0, true
	}
	return parsePeriodComponent(s)
}

func absInt32(n int32) int32 {
	if n < 0 {
		return -n
	}
	return n
}

func writeSigned(b *strings.Builder, n int32, suffix byte) {
	if n == 0 {
		return
	}
	fmt.Fprintf(b, "%+d%c", n, suffix)
}
