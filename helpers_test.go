package gotime

import "time"

func mustDate(year int, month time.Month, day int) Date {
	d, err := NewDate(year, month, day)
	if err != nil {
		panic(err)
	}
	return d
}

func mustTime(hour, minute, second int) Time {
	tm, err := NewTime(hour, minute, second)
	if err != nil {
		panic(err)
	}
	return tm
}

func mustTimeNanos(hour, minute, second, nanosecond int) Time {
	tm, err := NewTimeNanos(hour, minute, second, nanosecond)
	if err != nil {
		panic(err)
	}
	return tm
}

func mustDateTime(d Date, tm Time, z Zone) DateTime {
	dt, err := NewDateTime(d, tm, z)
	if err != nil {
		panic(err)
	}
	return dt
}
