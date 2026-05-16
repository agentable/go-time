package gotime

import (
	"testing"
	"time"

	"github.com/go-json-experiment/json"
)

func TestParseResultMarshalResolved(t *testing.T) {
	z := MustLoadZone("Asia/Tokyo")
	dt := mustDateTime(mustDate(2026, 3, 27), mustTime(13, 0, 0), z)
	r := ParseResult{
		Status: StatusResolved,
		Kind:   KindDateTime,
		Input:  "下周五下午三点",
		Zone:   z,
	}
	r.dateTime = dt

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	got := string(b)
	want := `{"kind":"parse_result","status":"resolved","input":"下周五下午三点","value_kind":"datetime","value":{"kind":"datetime","value":"2026-03-27T13:00:00+09:00","zone":"Asia/Tokyo","calendar":"iso8601"},"zone":"Asia/Tokyo"}`
	if got != want {
		t.Errorf("Marshal() = %s, want %s", got, want)
	}
}

func TestParseResultMarshalInvalid(t *testing.T) {
	r := ParseResult{
		Status: StatusInvalid,
		Input:  "04/05/2026",
		Error: &TimeError{
			Code:    CodeAmbiguousDate,
			Message: "Input date is ambiguous",
			Input:   "04/05/2026",
			Hint:    "Use locale to disambiguate",
		},
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	got := string(b)
	want := `{"kind":"parse_result","status":"invalid","input":"04/05/2026","error":{"code":"AMBIGUOUS_DATE","message":"Input date is ambiguous","input":"04/05/2026","hint":"Use locale to disambiguate"}}`
	if got != want {
		t.Errorf("Marshal() = %s, want %s", got, want)
	}
}

func TestParseResultMarshalAmbiguous(t *testing.T) {
	candidate := ParseResult{
		Status: StatusResolved,
		Kind:   KindDate,
		Input:  "04/05/2026",
		Warnings: []Warning{
			{Code: WarnInferredCalendar, Message: "month-first interpretation"},
		},
	}
	candidate.date = mustDate(2026, time.April, 5)

	r := ParseResult{
		Status: StatusAmbiguous,
		Kind:   KindDate,
		Input:  "04/05/2026",
		Warnings: []Warning{
			{Code: WarnInferredCalendar, Message: "date order ambiguous"},
		},
		Candidates: []ParseResult{candidate},
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	got := string(b)
	want := `{"kind":"parse_result","status":"ambiguous","input":"04/05/2026","warnings":[{"code":"inferred_calendar","message":"date order ambiguous"}],"value_kind":"date","candidates":[{"kind":"parse_result","status":"resolved","input":"04/05/2026","warnings":[{"code":"inferred_calendar","message":"month-first interpretation"}],"value_kind":"date","value":{"kind":"date","value":"2026-04-05","calendar":"iso8601"}}]}`
	if got != want {
		t.Errorf("Marshal() = %s, want %s", got, want)
	}
}

func TestParseResultMarshalResolvedDate(t *testing.T) {
	d := mustDate(2026, 3, 27)
	r := ParseResult{
		Status: StatusResolved,
		Kind:   KindDate,
		Input:  "2026-03-27",
	}
	r.date = d

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	got := string(b)
	want := `{"kind":"parse_result","status":"resolved","input":"2026-03-27","value_kind":"date","value":{"kind":"date","value":"2026-03-27","calendar":"iso8601"}}`
	if got != want {
		t.Errorf("Marshal() = %s, want %s", got, want)
	}
}

func TestParseResultMarshalResolvedDuration(t *testing.T) {
	dur := (2 * Hour)
	r := ParseResult{
		Status: StatusResolved,
		Kind:   KindDuration,
		Input:  "PT2H",
	}
	r.duration = dur

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	got := string(b)
	want := `{"kind":"parse_result","status":"resolved","input":"PT2H","value_kind":"duration","value":{"kind":"duration","iso":"PT2H"}}`
	if got != want {
		t.Errorf("Marshal() = %s, want %s", got, want)
	}
}

func TestParseResultMarshalResolvedInstant(t *testing.T) {
	instant := UnixMillis(0)
	r := ParseResult{
		Status: StatusResolved,
		Kind:   KindInstant,
		Input:  "1970-01-01T00:00:00Z",
	}
	r.instant = instant

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	got := string(b)
	want := `{"kind":"parse_result","status":"resolved","input":"1970-01-01T00:00:00Z","value_kind":"instant","value":{"kind":"instant","iso":"1970-01-01T00:00:00Z","epoch_ms":0}}`
	if got != want {
		t.Errorf("Marshal() = %s, want %s", got, want)
	}
}

func TestParseResultMarshalResolvedTime(t *testing.T) {
	tm := mustTime(13, 30, 45)
	r := ParseResult{
		Status: StatusResolved,
		Kind:   KindTime,
		Input:  "13:30:45",
	}
	r.timeVal = tm

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	got := string(b)
	want := `{"kind":"parse_result","status":"resolved","input":"13:30:45","value_kind":"time","value":{"kind":"time","value":"13:30:45","precision":"second"}}`
	if got != want {
		t.Errorf("Marshal() = %s, want %s", got, want)
	}
}

func TestParseResultMarshalResolvedInterval(t *testing.T) {
	start := UnixMillis(0)
	end := UnixMillis(3_600_000)
	iv, err := NewInterval(start, end)
	if err != nil {
		t.Fatalf("NewInterval error: %v", err)
	}
	r := ParseResult{
		Status: StatusResolved,
		Kind:   KindInterval,
		Input:  "1970-01-01T00:00:00Z/1970-01-01T01:00:00Z",
	}
	r.interval = iv

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	got := string(b)
	want := `{"kind":"parse_result","status":"resolved","input":"1970-01-01T00:00:00Z/1970-01-01T01:00:00Z","value_kind":"interval","value":{"kind":"interval","start":"1970-01-01T00:00:00Z","end":"1970-01-01T01:00:00Z"}}`
	if got != want {
		t.Errorf("Marshal() = %s, want %s", got, want)
	}
}
