package gotime_test

import (
	"fmt"
	"time"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"golang.org/x/text/language"

	gotime "github.com/agentable/go-time"
)

func mustDate(year int, month time.Month, day int) gotime.Date {
	d, err := gotime.NewDate(year, month, day)
	if err != nil {
		panic(err)
	}
	return d
}

func mustTime(hour, minute, second int) gotime.Time {
	tm, err := gotime.NewTime(hour, minute, second)
	if err != nil {
		panic(err)
	}
	return tm
}

func mustDateTime(d gotime.Date, tm gotime.Time, z gotime.Zone) gotime.DateTime {
	dt, err := gotime.NewDateTime(d, tm, z)
	if err != nil {
		panic(err)
	}
	return dt
}

// Parse accepts ISO 8601, RFC 3339, and natural language. The three-status
// result model (Resolved / Ambiguous / Invalid) lets callers handle any
// input without panicking.
func ExampleParse() {
	r := gotime.Parse("2026-03-27T13:00:00+09:00")
	if r.Status != gotime.StatusResolved {
		fmt.Println("unexpected status:", r.Status)
		return
	}
	i, _ := r.Instant()
	fmt.Println(i)

	// Output:
	// 2026-03-27T04:00:00Z
}

// WithInputLocale enables natural-language parsing in the given language.
// Use a golang.org/x/text/language Tag — the standard Go BCP-47 type —
// to avoid binding gotime to any specific i18n library.
func ExampleWithInputLocale() {
	ref := gotime.InstantFromTime(time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC))
	r := gotime.Parse("明天",
		gotime.WithInputLocale(language.SimplifiedChinese),
		gotime.WithZone(gotime.MustLoadZone("Asia/Shanghai")),
		gotime.WithReference(ref),
	)
	if r.Status != gotime.StatusResolved {
		return
	}
	d, _ := r.Date()
	fmt.Println(d)

	// Output:
	// 2026-03-31
}

// Calendar math preserves wall-clock time across DST transitions and clamps
// end-of-month overflows. Exact math always adds precise durations.
func ExampleDateTime_AddPeriod() {
	zone := gotime.MustLoadZone("America/New_York")

	jan31 := mustDateTime(
		mustDate(2026, time.January, 31),
		mustTime(12, 0, 0),
		zone,
	)
	feb := jan31.AddPeriod(gotime.Months(1))
	fmt.Println("Jan 31 + 1 month:", feb.Date())

	exact := jan31.Add(48 * gotime.Hour)
	fmt.Println("Jan 31 + 48h:", exact.Date(), exact.Clock())

	// Output:
	// Jan 31 + 1 month: 2026-02-28
	// Jan 31 + 48h: 2026-02-02 12:00:00
}

// Std returns the underlying time.Time. Use it to bridge a DateTime to any
// API expecting a stdlib time.Time — for example, a github.com/agentable/go-intl
// DateTimeFormat or any other formatter.
func ExampleDateTime_Std() {
	zone := gotime.MustLoadZone("Asia/Tokyo")
	dt := mustDateTime(
		mustDate(2026, time.March, 27),
		mustTime(13, 0, 0),
		zone,
	)

	// Hand off to stdlib formatter.
	fmt.Println(dt.Std().Format("2006-01-02 15:04 MST"))

	// Output:
	// 2026-03-27 13:00 JST
}

// Decompose breaks a Duration into the ECMA-402 DurationFormat slots
// (Hours/Minutes/Seconds/Milliseconds/Microseconds/Nanoseconds). The shape
// mirrors durationformat.Duration so a struct conversion is enough to hand
// it off — gotime never imports the formatter.
func ExampleDuration_Decompose() {
	d := 1*gotime.Hour + 30*gotime.Minute + 250*gotime.Millisecond
	c := d.Decompose()
	fmt.Printf("Hours=%d Minutes=%d Milliseconds=%d\n", c.Hours, c.Minutes, c.Milliseconds)

	// Output:
	// Hours=1 Minutes=30 Milliseconds=250
}

// Value objects serialize to stable JSON schemas. All types round-trip
// through json.Marshal / json.Unmarshal without losing precision.
func Example_jsonRoundTrip() {
	zone := gotime.MustLoadZone("Asia/Tokyo")
	dt := mustDateTime(
		mustDate(2026, time.March, 27),
		mustTime(13, 0, 0),
		zone,
	)

	b, _ := json.Marshal(dt, jsontext.WithIndent("  "))
	fmt.Println(string(b))

	// Output:
	// {
	//   "kind": "datetime",
	//   "value": "2026-03-27T13:00:00+09:00",
	//   "zone": "Asia/Tokyo",
	//   "calendar": "iso8601"
	// }
}
