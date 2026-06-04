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
func (i Instant) In(z Zone) DateTime {
	z = normalizeZone(z)
	return DateTime{t: i.t.In(z.Location()), zone: z}
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

// MarshalJSON encodes i as {"kind":"instant","iso":"<RFC3339Nano UTC>","epoch_ms":N}.
func (i Instant) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Kind    string `json:"kind"`
		ISO     string `json:"iso"`
		EpochMs int64  `json:"epoch_ms"`
	}{Kind: "instant", ISO: i.t.Format(time.RFC3339Nano), EpochMs: i.t.UnixMilli()})
}

// UnmarshalJSON decodes i from {"kind":"instant","iso":"<RFC3339Nano>",...}.
func (i *Instant) UnmarshalJSON(b []byte) error {
	var wire struct {
		Kind string `json:"kind"`
		ISO  string `json:"iso"`
	}
	if err := json.Unmarshal(b, &wire); err != nil {
		return err
	}
	t, err := time.Parse(time.RFC3339Nano, wire.ISO)
	if err != nil {
		return fmt.Errorf("gotime: invalid instant iso %q: %w", wire.ISO, err)
	}
	*i = InstantFromTime(t)
	return nil
}
