package gotime

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-json-experiment/json"
)

// errInvalidIntervalValue is returned when an interval value string lacks the expected "<start>/<end>" format.
var errInvalidIntervalValue = errors.New("invalid interval value: expected <start>/<end>")

// Interval is a half-open time range [start, end) bounded by two Instants.
// Interval is zone-free — projection zone for display is the caller's concern;
// bridge via iv.StdRange() and format outside this package.
// Arithmetic is UTC-based on the underlying Instants.
type Interval struct {
	start Instant
	end   Instant
}

// NewInterval creates an Interval. Returns ErrIntervalReversed if end < start.
func NewInterval(start, end Instant) (Interval, error) {
	if end.Before(start) {
		return Interval{}, newTimeError(
			ErrIntervalReversed,
			"interval end must not be before start",
			fmt.Sprintf("start=%s end=%s", start, end),
			"swap start and end, or check the upstream parsing",
		)
	}
	return Interval{start: start, end: end}, nil
}

// NewIntervalStartingAt creates an Interval from a start Instant and non-negative length.
func NewIntervalStartingAt(start Instant, length Duration) (Interval, error) {
	if length.IsNegative() {
		return Interval{}, newTimeError(
			ErrInvalidDuration,
			"interval length must not be negative",
			length.String(),
			"provide a non-negative length or use NewIntervalEndingAt for an interval ending at a known instant",
		)
	}
	return NewInterval(start, start.Add(length))
}

// NewIntervalEndingAt creates an Interval from an end Instant and non-negative length.
func NewIntervalEndingAt(end Instant, length Duration) (Interval, error) {
	if length.IsNegative() {
		return Interval{}, newTimeError(
			ErrInvalidDuration,
			"interval length must not be negative",
			length.String(),
			"provide a non-negative length or use NewIntervalStartingAt for an interval starting at a known instant",
		)
	}
	return NewInterval(end.Add(-length), end)
}

// Start returns the start instant (inclusive).
func (iv Interval) Start() Instant { return iv.start }

// End returns the end instant (exclusive).
func (iv Interval) End() Instant { return iv.end }

// Length returns the duration of the interval.
func (iv Interval) Length() Duration { return iv.end.Sub(iv.start) }

// StdRange returns the underlying start and end as stdlib time.Time values in UTC.
// Use this to bridge an Interval to any API expecting (time.Time, time.Time).
func (iv Interval) StdRange() (start, end time.Time) {
	return iv.start.Std(), iv.end.Std()
}

// IsZero reports whether iv is the zero value.
func (iv Interval) IsZero() bool { return iv.start.IsZero() && iv.end.IsZero() }

// Contains reports whether i is within the half-open interval [start, end).
func (iv Interval) Contains(i Instant) bool {
	return !i.Before(iv.start) && i.Before(iv.end)
}

// Overlaps reports whether iv and other share any moment.
// Half-open intervals [a, b) and [b, c) do NOT overlap — they are adjacent.
func (iv Interval) Overlaps(other Interval) bool {
	return iv.start.Before(other.end) && other.start.Before(iv.end)
}

// Adjacent reports whether iv and other share exactly one boundary with no overlap and no gap.
// For half-open intervals, [a, b) and [b, c) are adjacent.
func (iv Interval) Adjacent(other Interval) bool {
	return iv.end.Equal(other.start) || other.end.Equal(iv.start)
}

// Intersect returns the overlapping portion of iv and other.
// Returns (zero, false) if they are disjoint.
func (iv Interval) Intersect(other Interval) (Interval, bool) {
	start := laterInstant(iv.start, other.start)
	end := earlierInstant(iv.end, other.end)
	if end.Compare(start) <= 0 {
		return Interval{}, false
	}
	return Interval{start: start, end: end}, true
}

// Union returns the smallest interval containing both iv and other.
// Adjacent intervals ([a, b) and [b, c)) can be unioned.
// Returns an error if the intervals are disjoint with a gap between them.
func (iv Interval) Union(other Interval) (Interval, error) {
	if !iv.Overlaps(other) && !iv.Adjacent(other) {
		return Interval{}, newTimeError(
			ErrIntervalsDisjoint,
			"intervals are disjoint",
			fmt.Sprintf("%s and %s", iv, other),
			"only overlapping or adjacent intervals can be unioned",
		)
	}
	return Interval{
		start: earlierInstant(iv.start, other.start),
		end:   laterInstant(iv.end, other.end),
	}, nil
}

func earlierInstant(a, b Instant) Instant {
	if a.Compare(b) <= 0 {
		return a
	}
	return b
}

func laterInstant(a, b Instant) Instant {
	if a.Compare(b) >= 0 {
		return a
	}
	return b
}

// Shift returns a new Interval with start and end each advanced by d.
func (iv Interval) Shift(d Duration) Interval {
	return Interval{
		start: iv.start.Add(d),
		end:   iv.end.Add(d),
	}
}

// Expand returns a new Interval with start moved back by before and end moved forward by after.
func (iv Interval) Expand(before, after Duration) (Interval, error) {
	if before.IsNegative() {
		return Interval{}, newTimeError(
			ErrInvalidDuration,
			"interval expansion before must not be negative",
			before.String(),
			"provide a non-negative duration for before",
		)
	}
	if after.IsNegative() {
		return Interval{}, newTimeError(
			ErrInvalidDuration,
			"interval expansion after must not be negative",
			after.String(),
			"provide a non-negative duration for after",
		)
	}
	return NewInterval(iv.start.Add(-before), iv.end.Add(after))
}

// String returns the ISO 8601 interval notation "<start>/<end>".
func (iv Interval) String() string {
	return fmt.Sprintf("%s/%s", iv.start.String(), iv.end.String())
}

// MarshalJSON encodes iv as {"kind":"interval","start":"<RFC3339Nano>","end":"<RFC3339Nano>"}.
func (iv Interval) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Kind  string `json:"kind"`
		Start string `json:"start"`
		End   string `json:"end"`
	}{
		Kind:  "interval",
		Start: iv.start.String(),
		End:   iv.end.String(),
	})
}

// UnmarshalJSON decodes iv from {"kind":"interval","start":"<RFC3339Nano>","end":"<RFC3339Nano>"}.
func (iv *Interval) UnmarshalJSON(b []byte) error {
	var wire struct {
		Kind  string `json:"kind"`
		Start string `json:"start"`
		End   string `json:"end"`
	}
	if err := unmarshalJSONWire(b, &wire); err != nil {
		return err
	}
	if err := requireJSONKind("interval", wire.Kind, "interval"); err != nil {
		return err
	}
	if wire.Start == "" || wire.End == "" {
		return fmt.Errorf("%w: missing start or end", errInvalidIntervalValue)
	}
	st, err := time.Parse(time.RFC3339Nano, wire.Start)
	if err != nil {
		return fmt.Errorf("gotime: invalid interval start %q: %w", wire.Start, err)
	}
	et, err := time.Parse(time.RFC3339Nano, wire.End)
	if err != nil {
		return fmt.Errorf("gotime: invalid interval end %q: %w", wire.End, err)
	}
	result, err := NewInterval(InstantFromTime(st), InstantFromTime(et))
	if err != nil {
		return fmt.Errorf("gotime: %w", err)
	}
	*iv = result
	return nil
}
