package gotime

import (
	"fmt"
	"time"

	"github.com/go-json-experiment/json"
)

// Instant is an absolute UTC moment with nanosecond precision.
// It is the preferred type for storage, logging, and cross-system transfer.
type Instant struct {
	t time.Time // always UTC
}

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
func (i Instant) UnixNano() int64 { return i.t.UnixNano() }

// UnixMilli returns the instant as Unix time in milliseconds.
func (i Instant) UnixMilli() int64 { return i.t.UnixMilli() }

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
func (i Instant) Sub(other Instant) Duration {
	return Duration(i.t.Sub(other.t))
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
		return fmt.Errorf("gotime: invalid instant iso %q: %w", wire.ISO, err)
	}
	*i = InstantFromTime(t)
	return nil
}
