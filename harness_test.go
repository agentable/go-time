package gotime

import (
	"testing"
	"time"
)

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

func mustDateTimeFromTime(t *testing.T, tm time.Time, z Zone) DateTime {
	t.Helper()
	dt, err := DateTimeFromTime(tm, z)
	if err != nil {
		t.Fatalf("DateTimeFromTime(%v, %v) error: %v", tm, z, err)
	}
	return dt
}

func mustInstantIn(t *testing.T, instant Instant, z Zone) DateTime {
	t.Helper()
	dt, err := instant.In(z)
	if err != nil {
		t.Fatalf("Instant.In(%v) error: %v", z, err)
	}
	return dt
}

func mustDateTimeIn(t *testing.T, dt DateTime, z Zone) DateTime {
	t.Helper()
	projected, err := dt.In(z)
	if err != nil {
		t.Fatalf("DateTime.In(%v) error: %v", z, err)
	}
	return projected
}

func mustDateTimeAdd(t *testing.T, dt DateTime, d Duration) DateTime {
	t.Helper()
	result, err := dt.Add(d)
	if err != nil {
		t.Fatalf("DateTime.Add(%v) error: %v", d, err)
	}
	return result
}

func mustDateTimeAddPeriod(t *testing.T, dt DateTime, p Period) DateTime {
	t.Helper()
	resolution, err := dt.AddPeriod(p)
	if err != nil {
		t.Fatalf("DateTime.AddPeriod(%v) error: %v", p, err)
	}
	result, err := resolution.Only()
	if err != nil {
		t.Fatalf("DateTime.AddPeriod(%v).Only() error: %v", p, err)
	}
	return result
}
