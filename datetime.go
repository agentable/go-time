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

// clampAddDate adds years and months with end-of-month clamping, then adds days.
// Clamping prevents Go's default overflow (Jan 31 + 1 month overflows to Mar 3).
// When years and months are both zero, the fast path delegates to AddDate.
func clampAddDate(t time.Time, years, months, days int) time.Time {
	if years == 0 && months == 0 {
		return t.AddDate(0, 0, days)
	}
	year := t.Year() + years + months/12
	month := int(t.Month()) + months%12
	if month > 12 {
		year++
		month -= 12
	} else if month < 1 {
		year--
		month += 12
	}
	day := t.Day()
	if last := daysInMonth(year, time.Month(month)); day > last {
		day = last
	}
	result := time.Date(year, time.Month(month), day, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	if days != 0 {
		result = result.AddDate(0, 0, days)
	}
	return result
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
func DateTimeFromTime(t time.Time, z Zone) DateTime {
	z = normalizeZone(z)
	return DateTime{t: t.In(z.Location()), zone: z}
}

// Date returns the calendar date component of dt.
func (dt DateTime) Date() Date { return DateFromTime(dt.t) }

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
func (dt DateTime) In(z Zone) DateTime {
	z = normalizeZone(z)
	return DateTime{t: dt.t.In(z.Location()), zone: z}
}

// Add returns a new DateTime advanced by d using exact (nanosecond) arithmetic.
// To move back, pass a negative Duration: dt.Add(-30 * gotime.Minute).
func (dt DateTime) Add(d Duration) DateTime {
	return DateTime{t: dt.t.Add(d.Std()), zone: dt.zone}
}

// AddPeriod returns a new DateTime advanced by p (calendar arithmetic),
// preserving the local wall-clock time across DST transitions.
// Month/year arithmetic applies end-of-month clamping (Jan 31 + 1 month = Feb 28/29).
// To move back, pass a negated Period: dt.AddPeriod(gotime.Months(1).Negate()).
func (dt DateTime) AddPeriod(p Period) DateTime {
	return DateTime{t: clampAddDate(dt.t, int(p.Years), int(p.Months), int(p.Days)), zone: dt.zone}
}

// Sub returns the Duration from other to dt, mirroring time.Time.Sub.
// Use Add for arithmetic — there is no Sub(Duration) form.
func (dt DateTime) Sub(other DateTime) Duration {
	return Duration(dt.t.Sub(other.t))
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

// MarshalJSON encodes dt as {"kind":"datetime","value":"<RFC3339Nano>","zone":"<IANA id>","calendar":"iso8601"}.
func (dt DateTime) MarshalJSON() ([]byte, error) {
	zoneID := normalizeZone(dt.zone).ID()
	if isFixedOffsetID(zoneID) {
		return nil, newTimeError(
			ErrInvalidZone,
			"fixed UTC offsets are not IANA zones",
			zoneID,
			"represent numeric offsets as instant syntax, not as DateTime zone identity",
		)
	}
	return json.Marshal(struct {
		Kind     string `json:"kind"`
		Value    string `json:"value"`
		Zone     string `json:"zone"`
		Calendar string `json:"calendar"`
	}{
		Kind:     "datetime",
		Value:    dt.t.Format(time.RFC3339Nano),
		Zone:     zoneID,
		Calendar: "iso8601",
	})
}

// UnmarshalJSON decodes dt from {"kind":"datetime","value":"<RFC3339Nano>","zone":"<IANA id>"[,"calendar":"..."]}.
func (dt *DateTime) UnmarshalJSON(b []byte) error {
	var wire struct {
		Kind     string `json:"kind"`
		Value    string `json:"value"`
		Zone     string `json:"zone"`
		Calendar string `json:"calendar"`
	}
	if err := json.Unmarshal(b, &wire); err != nil {
		return err
	}
	t, err := time.Parse(time.RFC3339Nano, wire.Value)
	if err != nil {
		return fmt.Errorf("gotime: invalid datetime value %q: %w", wire.Value, err)
	}
	if wire.Zone == "" {
		return newTimeError(
			ErrInvalidZone,
			"datetime zone is required",
			wire.Value,
			"include an IANA zone id in the zone field, e.g. Asia/Tokyo",
		)
	}
	z, err := LoadZone(wire.Zone)
	if err != nil {
		return fmt.Errorf("gotime: invalid zone %q: %w", wire.Zone, err)
	}
	if !dateTimeOffsetMatchesZone(t, z) {
		return newTimeError(
			ErrInvalidZone,
			"datetime value offset does not match zone",
			wire.Value+"["+wire.Zone+"]",
			"encode DateTime values using the offset observed by the IANA zone at that instant",
		)
	}
	*dt = DateTimeFromTime(t, z)
	return nil
}

func dateTimeOffsetMatchesZone(t time.Time, z Zone) bool {
	_, valueOffset := t.Zone()
	_, zoneOffset := t.In(z.Location()).Zone()
	return valueOffset == zoneOffset
}
