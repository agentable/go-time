package gotime

import (
	"fmt"
	"math"
	"time"

	"github.com/go-json-experiment/json"
)

// Instant is an absolute UTC moment with nanosecond precision.
// It is the preferred type for storage, logging, and cross-system transfer.
type Instant struct {
	t time.Time // always UTC
}

var (
	unixNanoMinTime    = time.Unix(0, math.MinInt64).UTC()
	unixNanoMaxTime    = time.Unix(0, math.MaxInt64).UTC()
	unixMilliMinTime   = time.UnixMilli(math.MinInt64).UTC()
	unixMilliLimitTime = time.UnixMilli(math.MaxInt64).Add(time.Millisecond).UTC()
)

// InstantFromTime creates an Instant from a time.Time, forcing UTC.
func InstantFromTime(t time.Time) Instant {
	return Instant{t: t.UTC()}
}

// UnixSeconds creates an Instant from a Unix timestamp in seconds.
func UnixSeconds(s int64) Instant { return Instant{t: time.Unix(s, 0).UTC()} }

// UnixMillis creates an Instant from a Unix timestamp in milliseconds.
func UnixMillis(ms int64) Instant { return Instant{t: time.UnixMilli(ms).UTC()} }

// UnixNanos creates an Instant from a Unix timestamp in nanoseconds.
func UnixNanos(ns int64) Instant { return Instant{t: time.Unix(0, ns).UTC()} }

// Std returns the underlying time.Time in UTC.
func (i Instant) Std() time.Time { return i.t }

// UnixNano returns the instant as Unix time in nanoseconds.
// It returns ErrOverflow when the value cannot be represented by int64.
func (i Instant) UnixNano() (int64, error) {
	if !i.t.Before(unixNanoMinTime) && !i.t.After(unixNanoMaxTime) {
		return i.t.UnixNano(), nil
	}
	return 0, newTimeError(
		ErrOverflow,
		"instant is outside the Unix nanosecond range",
		i.String(),
		"use UnixMilli when millisecond precision is sufficient or keep the Instant value",
	)
}

// UnixMilli returns the instant as Unix time in milliseconds.
// It returns ErrOverflow when the value cannot be represented by int64.
func (i Instant) UnixMilli() (int64, error) {
	if !i.t.Before(unixMilliMinTime) && i.t.Before(unixMilliLimitTime) {
		return i.t.UnixMilli(), nil
	}
	return 0, newTimeError(
		ErrOverflow,
		"instant is outside the Unix millisecond range",
		i.String(),
		"keep the Instant value or use a wider scalar representation in the caller's domain",
	)
}

// IsZero reports whether i is the zero value.
func (i Instant) IsZero() bool { return i.t.IsZero() }

// In projects the Instant into a timezone, returning a DateTime.
// It returns ErrOverflow when the projected year is outside 0000..9999.
func (i Instant) In(z Zone) (DateTime, error) {
	return DateTimeFromTime(i.t, z)
}

// Add returns a new Instant advanced by d.
func (i Instant) Add(d Duration) Instant {
	return Instant{t: i.t.Add(d.Std())}
}

// Sub returns the Duration from other to i.
// It returns ErrOverflow when the exact difference does not fit Duration.
func (i Instant) Sub(other Instant) (Duration, error) {
	d := i.t.Sub(other.t)
	if other.t.Add(d).Equal(i.t) {
		return Duration(d), nil
	}
	return 0, newTimeError(
		ErrOverflow,
		"instant difference exceeds the Duration range",
		fmt.Sprintf("start=%s end=%s", other, i),
		"use closer instants or keep the wider difference in the caller's domain",
	)
}

// Before reports whether i is before other.
func (i Instant) Before(other Instant) bool { return i.t.Before(other.t) }

// After reports whether i is after other.
func (i Instant) After(other Instant) bool { return i.t.After(other.t) }

// Equal reports whether i and other represent the same moment.
func (i Instant) Equal(other Instant) bool { return i.t.Equal(other.t) }

// Compare returns -1 if i < other, 0 if i == other, 1 if i > other.
// Use with slices.MinFunc / slices.MaxFunc / cmp.Or for selection and clamping:
//
//	earliest := slices.MinFunc(times, gotime.Instant.Compare)
func (i Instant) Compare(other Instant) int {
	return i.t.Compare(other.t)
}

// String returns the RFC3339Nano representation in UTC.
func (i Instant) String() string {
	return i.t.Format(time.RFC3339Nano)
}

// MarshalJSON encodes i as {"kind":"instant","iso":"<RFC3339Nano UTC>"}.
func (i Instant) MarshalJSON() ([]byte, error) {
	iso, err := i.wireISO()
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Kind string `json:"kind"`
		ISO  string `json:"iso"`
	}{Kind: "instant", ISO: iso})
}

func (i Instant) wireISO() (string, error) {
	t := i.t.UTC()
	if year := t.Year(); year < 0 || year > 9999 {
		return "", newTimeError(
			ErrOverflow,
			"instant year is outside the supported wire domain",
			fmt.Sprintf("year=%d", year),
			"marshal an instant whose UTC year is between 0000 and 9999",
		)
	}
	iso := t.Format(time.RFC3339Nano)
	if _, err := time.Parse(time.RFC3339Nano, iso); err != nil {
		return "", newTimeError(
			ErrInvalidFormat,
			"instant cannot be represented by the RFC3339 wire format",
			iso,
			"marshal an instant that has a canonical RFC3339 UTC representation",
		)
	}
	return iso, nil
}

// UnmarshalJSON decodes i from {"kind":"instant","iso":"<RFC3339Nano>",...}.
func (i *Instant) UnmarshalJSON(b []byte) error {
	var wire struct {
		Kind string `json:"kind"`
		ISO  string `json:"iso"`
	}
	if err := unmarshalJSONWire(b, &wire); err != nil {
		return err
	}
	if err := requireJSONKind("instant", wire.Kind, "instant"); err != nil {
		return err
	}
	if err := requireJSONString("instant", "iso", wire.ISO); err != nil {
		return err
	}
	t, err := time.Parse(time.RFC3339Nano, wire.ISO)
	if err != nil {
		return newTimeErrorWithCause(
			ErrInvalidFormat,
			err,
			"instant iso is not valid RFC3339",
			wire.ISO,
			"use an RFC3339 instant such as 2026-03-27T04:00:00Z",
		)
	}
	*i = InstantFromTime(t)
	return nil
}
