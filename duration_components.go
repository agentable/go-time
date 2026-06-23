package gotime

// DurationComponents is a Duration broken into renderable clock slots:
// Hours through Nanoseconds. It mirrors the slot shape consumed by external
// duration formatters (e.g. an ECMA-402 Intl.DurationFormat implementation)
// so a caller can copy the fields directly:
//
//	c := d.Decompose()
//	out, _ := f.Format(durationformat.Duration{
//	    Hours:   c.Hours,
//	    Minutes: c.Minutes,
//	    // ...
//	})
//
// gotime never imports any duration-formatter package — this type holds the
// shape, never the rendering. There are intentionally no calendar fields
// (Years/Months/Days): those belong to Period, whose fields are already
// exported, so widening them into a second struct would be duplication.
//
// DurationComponents is a computed bridge, not a wire format. It is not
// marshalled by any value object; Duration's JSON contains only the ISO
// 8601 string. Run Decompose at the call site when you need structured
// slots.
type DurationComponents struct {
	// Hours is the whole-hour slot.
	Hours int64
	// Minutes is the remaining whole-minute slot.
	Minutes int64
	// Seconds is the remaining whole-second slot.
	Seconds int64
	// Milliseconds is the remaining whole-millisecond slot.
	Milliseconds int64
	// Microseconds is the remaining whole-microsecond slot.
	Microseconds int64
	// Nanoseconds is the remaining nanosecond slot.
	Nanoseconds int64
}
