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
	Hours        int64
	Minutes      int64
	Seconds      int64
	Milliseconds int64
	Microseconds int64
	Nanoseconds  int64
}
