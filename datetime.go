package gotime

import (
	"fmt"
	"time"

	"github.com/go-json-experiment/json"
)

// daysInMonth returns the number of days in the given month of the given year.
func daysInMonth(year int, month time.Month) int {
	// time.Date normalizes day=0 to the last day of the previous month.
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// DateTime is a date and time in a specific timezone.
// It is the "human-readable" view of a moment.
type DateTime struct {
	t    time.Time // in zone.loc
	zone Zone
}

// NewDateTime creates a DateTime from a Date, clock Time, and Zone.
func NewDateTime(d Date, t Time, z Zone) (DateTime, error) {
	if msg := validateDateComponents(d.year, int(d.month), d.day); msg != "" {
		return DateTime{}, newTimeError(
			ErrInvalidDate,
			msg,
			d.String(),
			"construct Date with NewDate before combining it into a DateTime",
		)
	}
	if msg := validateTimeComponents(t.hour, t.minute, t.second, t.nanosecond); msg != "" {
		return DateTime{}, newTimeError(
			ErrInvalidTime,
			msg,
			t.String(),
			"construct Time with NewTime or NewTimeNanos before combining it into a DateTime",
		)
	}
	return NewLocalDateTime(d, t).Resolve(z).Only()
}

// DateTimeFromTime creates a DateTime from a stdlib time.Time and a Zone.
// It returns ErrOverflow when projection into z produces a year outside 0000..9999.
func DateTimeFromTime(t time.Time, z Zone) (DateTime, error) {
	z = normalizeZone(z)
	projected := t.In(z.Location())
	if _, err := DateFromTime(projected); err != nil {
		return DateTime{}, err
	}
	return dateTimeFromTimeTrusted(projected, z), nil
}

func dateTimeFromTimeTrusted(t time.Time, z Zone) DateTime {
	z = normalizeZone(z)
	return DateTime{t: t.In(z.Location()), zone: z}
}

// Date returns the calendar date component of dt.
func (dt DateTime) Date() Date { return dateFromTimeTrusted(dt.t) }

// Clock returns the clock time component of dt.
func (dt DateTime) Clock() Time { return TimeFromTime(dt.t) }

// Std returns the underlying time.Time in dt's zone. Use this to bridge
// dt to any API expecting a stdlib time.Time (e.g. a formatter).
func (dt DateTime) Std() time.Time { return dt.t }

// Zone returns the timezone of dt.
func (dt DateTime) Zone() Zone { return dt.zone }

// Instant converts dt to an absolute UTC Instant.
func (dt DateTime) Instant() Instant { return InstantFromTime(dt.t) }

// In converts dt to the same absolute moment expressed in zone z.
// It returns ErrOverflow when the projected year is outside 0000..9999.
func (dt DateTime) In(z Zone) (DateTime, error) {
	return DateTimeFromTime(dt.t, z)
}

// Add returns a new DateTime advanced by d using exact (nanosecond) arithmetic.
// To move back, pass a negative Duration: dt.Add(-30 * gotime.Minute).
// It returns ErrOverflow when the result leaves the supported civil year domain.
func (dt DateTime) Add(d Duration) (DateTime, error) {
	return DateTimeFromTime(dt.t.Add(d.Std()), dt.zone)
}

// AddPeriod advances dt by p using calendar arithmetic and resolves the target
// local time in the same zone. Gaps and overlaps remain explicit in the result.
func (dt DateTime) AddPeriod(p Period) (LocalResolution, error) {
	targetDate, err := dt.Date().Add(p)
	if err != nil {
		return LocalResolution{}, err
	}
	return NewLocalDateTime(targetDate, dt.Clock()).Resolve(dt.zone), nil
}

// Sub returns the Duration from other to dt.
// Use Add for arithmetic — there is no Sub(Duration) form.
// It returns ErrOverflow when the exact difference does not fit Duration.
func (dt DateTime) Sub(other DateTime) (Duration, error) {
	return dt.Instant().Sub(other.Instant())
}

// Before reports whether dt is before other.
func (dt DateTime) Before(other DateTime) bool { return dt.t.Before(other.t) }

// After reports whether dt is after other.
func (dt DateTime) After(other DateTime) bool { return dt.t.After(other.t) }

// Equal reports whether dt and other represent the same absolute moment.
func (dt DateTime) Equal(other DateTime) bool { return dt.t.Equal(other.t) }

// Compare returns -1 if dt < other, 0 if equal, 1 if dt > other.
func (dt DateTime) Compare(other DateTime) int { return dt.t.Compare(other.t) }

// IsZero reports whether dt is the zero value.
func (dt DateTime) IsZero() bool { return dt.t.IsZero() }

// String returns the RFC3339Nano string with zone offset.
func (dt DateTime) String() string {
	return dt.t.Format(time.RFC3339Nano)
}

// MarshalJSON encodes dt as {"kind":"datetime","instant":"<RFC3339Nano UTC>","zone":"<IANA id>"}.
func (dt DateTime) MarshalJSON() ([]byte, error) {
	z := normalizeZone(dt.zone)
	zoneID := z.ID()
	if isFixedOffsetID(zoneID) {
		return nil, newTimeError(
			ErrInvalidZone,
			"fixed UTC offsets are not IANA zones",
			zoneID,
			"represent numeric offsets as instant syntax, not as DateTime zone identity",
		)
	}
	loadedZone, err := LoadZone(zoneID)
	if err != nil {
		return nil, err
	}
	projected, err := DateTimeFromTime(dt.t, loadedZone)
	if err != nil {
		return nil, err
	}
	instant, err := projected.Instant().wireISO()
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Kind    string `json:"kind"`
		Instant string `json:"instant"`
		Zone    string `json:"zone"`
	}{
		Kind:    "datetime",
		Instant: instant,
		Zone:    zoneID,
	})
}

// UnmarshalJSON decodes dt from {"kind":"datetime","instant":"<RFC3339Nano UTC>","zone":"<IANA id>"}.
func (dt *DateTime) UnmarshalJSON(b []byte) error {
	var wire struct {
		Kind    string `json:"kind"`
		Instant string `json:"instant"`
		Zone    string `json:"zone"`
	}
	if err := unmarshalJSONWire(b, &wire); err != nil {
		return err
	}
	if err := requireJSONKind("datetime", wire.Kind, "datetime"); err != nil {
		return err
	}
	if err := requireJSONString("datetime", "instant", wire.Instant); err != nil {
		return err
	}
	if err := requireJSONString("datetime", "zone", wire.Zone); err != nil {
		return err
	}
	t, err := time.Parse(time.RFC3339Nano, wire.Instant)
	if err != nil {
		return newTimeErrorWithCause(
			ErrInvalidFormat,
			err,
			"datetime instant is not valid RFC3339",
			wire.Instant,
			"use a canonical UTC instant such as 2026-03-27T04:00:00Z",
		)
	}
	instant := InstantFromTime(t)
	if canonical := instant.String(); wire.Instant != canonical {
		return newTimeError(
			ErrInvalidFormat,
			"datetime instant is not canonical UTC",
			wire.Instant,
			fmt.Sprintf("use %q for this instant", canonical),
		)
	}
	z, err := LoadZone(wire.Zone)
	if err != nil {
		return err
	}
	parsed, err := instant.In(z)
	if err != nil {
		return err
	}
	*dt = parsed
	return nil
}
