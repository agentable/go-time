package gotime

import (
	"fmt"
	"time"

	"github.com/go-json-experiment/json"

	ianazone "github.com/agentable/go-time/internal/zone"
)

// LocalDateTime is a calendar date plus a clock time, without a zone.
// Resolving it in a Zone may produce zero, one, or multiple DateTime candidates.
type LocalDateTime struct {
	// Date is the calendar date component.
	Date Date
	// Time is the clock time component.
	Time Time
}

// NewLocalDateTime combines a Date and Time without resolving them into a zone.
func NewLocalDateTime(d Date, t Time) LocalDateTime {
	return LocalDateTime{Date: d, Time: t}
}

// String returns the ISO 8601 local date-time string "YYYY-MM-DDTHH:MM:SS[.fraction]".
func (ldt LocalDateTime) String() string {
	return ldt.Date.String() + "T" + ldt.Time.String()
}

// MarshalJSON encodes ldt as {"kind":"local_datetime","value":"YYYY-MM-DDTHH:MM:SS","calendar":"iso8601"}.
func (ldt LocalDateTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Kind     string `json:"kind"`
		Value    string `json:"value"`
		Calendar string `json:"calendar"`
	}{
		Kind:     "local_datetime",
		Value:    ldt.String(),
		Calendar: "iso8601",
	})
}

// UnmarshalJSON decodes ldt from {"kind":"local_datetime","value":"YYYY-MM-DDTHH:MM:SS[.fraction]"}.
func (ldt *LocalDateTime) UnmarshalJSON(b []byte) error {
	var wire struct {
		Kind     string `json:"kind"`
		Value    string `json:"value"`
		Calendar string `json:"calendar"`
	}
	if err := json.Unmarshal(b, &wire); err != nil {
		return err
	}
	parsed, err := time.Parse("2006-01-02T15:04:05", wire.Value)
	if err != nil {
		return fmt.Errorf("gotime: invalid local datetime value %q: %w", wire.Value, err)
	}
	*ldt = NewLocalDateTime(DateFromTime(parsed), TimeFromTime(parsed))
	return nil
}

// LocalResolutionStatus describes how a LocalDateTime maps into a Zone.
type LocalResolutionStatus string

const (
	// LocalInvalid means the local date or clock time is invalid.
	LocalInvalid LocalResolutionStatus = "invalid"
	// LocalResolved means the local time maps to exactly one DateTime.
	LocalResolved LocalResolutionStatus = "resolved"
	// LocalNonexistent means the local time falls in a DST gap.
	LocalNonexistent LocalResolutionStatus = "nonexistent"
	// LocalAmbiguous means the local time maps to multiple DateTime candidates.
	LocalAmbiguous LocalResolutionStatus = "ambiguous"
)

// LocalResolution holds the result of resolving a LocalDateTime in a Zone.
type LocalResolution struct {
	// Status classifies the local time projection.
	Status LocalResolutionStatus
	// Zone is the zone used for resolution. A zero Zone resolves as UTC.
	Zone Zone
	// Local is the unresolved local date-time.
	Local LocalDateTime
	// Candidates holds the matching DateTime values in chronological order.
	Candidates []DateTime
}

// Resolve projects ldt into z and classifies DST gaps and overlaps explicitly.
func (ldt LocalDateTime) Resolve(z Zone) LocalResolution {
	z = normalizeZone(z)
	result := LocalResolution{
		Status: LocalInvalid,
		Zone:   z,
		Local:  ldt,
	}

	if msg := validateDateComponents(ldt.Date.year, int(ldt.Date.month), ldt.Date.day); msg != "" {
		return result
	}
	if msg := validateTimeComponents(ldt.Time.hour, ldt.Time.minute, ldt.Time.second, ldt.Time.nanosecond); msg != "" {
		return result
	}

	res := ianazone.ProjectLocalTime(
		z.Location(),
		ldt.Date.year,
		ldt.Date.month,
		ldt.Date.day,
		ldt.Time.hour,
		ldt.Time.minute,
		ldt.Time.second,
	)
	switch res.Status {
	case ianazone.DSTNormal:
		result.Status = LocalResolved
		result.Candidates = []DateTime{ldt.dateTimeAt(z, res.Times[0])}
	case ianazone.DSTNonexistent:
		result.Status = LocalNonexistent
	case ianazone.DSTAmbiguous:
		result.Status = LocalAmbiguous
		result.Candidates = make([]DateTime, 0, len(res.Times))
		for _, candidate := range res.Times {
			result.Candidates = append(result.Candidates, ldt.dateTimeAt(z, candidate))
		}
	}
	return result
}

func (ldt LocalDateTime) dateTimeAt(z Zone, t time.Time) DateTime {
	return DateTime{t: t.Add(time.Duration(ldt.Time.nanosecond)), zone: z}
}

// Only returns the single resolved DateTime, or an error if resolution did not
// produce exactly one candidate.
func (r LocalResolution) Only() (DateTime, error) {
	switch r.Status {
	case LocalResolved:
		if len(r.Candidates) == 1 {
			return r.Candidates[0], nil
		}
		return DateTime{}, newTimeError(
			ErrInvalidTime,
			"resolved local time must have exactly one candidate",
			r.input(),
			"resolve the LocalDateTime again and use the returned candidate",
		)
	case LocalNonexistent:
		return DateTime{}, newTimeError(
			ErrNonexistentTime,
			fmt.Sprintf("local time %s does not exist in %s", r.Local, r.zoneID()),
			r.input(),
			"choose a real wall-clock time after the DST gap or construct from an Instant",
		)
	case LocalAmbiguous:
		return DateTime{}, newTimeError(
			ErrDuplicateTime,
			fmt.Sprintf("local time %s occurs more than once in %s", r.Local, r.zoneID()),
			r.input(),
			"choose one of LocalResolution.Candidates",
		)
	case LocalInvalid:
		if msg := validateDateComponents(r.Local.Date.year, int(r.Local.Date.month), r.Local.Date.day); msg != "" {
			return DateTime{}, newTimeError(
				ErrInvalidDate,
				msg,
				r.input(),
				"construct Date with NewDate before resolving a LocalDateTime",
			)
		}
		if msg := validateTimeComponents(r.Local.Time.hour, r.Local.Time.minute, r.Local.Time.second, r.Local.Time.nanosecond); msg != "" {
			return DateTime{}, newTimeError(
				ErrInvalidTime,
				msg,
				r.input(),
				"construct Time with NewTime or NewTimeNanos before resolving a LocalDateTime",
			)
		}
		return DateTime{}, newTimeError(
			ErrInvalidTime,
			"local time resolution is invalid",
			r.input(),
			"resolve the LocalDateTime in a Zone before asking for one DateTime",
		)
	default:
		return DateTime{}, newTimeError(
			ErrInvalidTime,
			"local time has not been resolved",
			r.input(),
			"call LocalDateTime.Resolve with a Zone before asking for one DateTime",
		)
	}
}

func (r LocalResolution) input() string {
	return fmt.Sprintf("%s[%s]", r.Local, r.zoneID())
}

func (r LocalResolution) zoneID() string {
	return normalizeZone(r.Zone).ID()
}
