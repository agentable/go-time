package gotime

import (
	"testing"

	"github.com/go-json-experiment/json"
)

func TestJSONUnmarshalRejectsWrongKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		decode func([]byte) error
	}{
		{
			name:   "date",
			input:  `{"kind":"time","value":"2026-03-27","calendar":"iso8601"}`,
			decode: func(b []byte) error { var v Date; return json.Unmarshal(b, &v) },
		},
		{
			name:   "duration",
			input:  `{"kind":"period","iso":"PT1H"}`,
			decode: func(b []byte) error { var v Duration; return json.Unmarshal(b, &v) },
		},
		{
			name:   "zone",
			input:  `{"kind":"instant","id":"UTC"}`,
			decode: func(b []byte) error { var v Zone; return json.Unmarshal(b, &v) },
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := tc.decode([]byte(tc.input)); err == nil {
				t.Fatalf("Unmarshal(%s) error = nil, want error", tc.input)
			}
		})
	}
}

func TestJSONUnmarshalRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		decode func([]byte) error
	}{
		{
			name:   "duration legacy parts",
			input:  `{"kind":"duration","iso":"PT1H","parts":{"hours":1}}`,
			decode: func(b []byte) error { var v Duration; return json.Unmarshal(b, &v) },
		},
		{
			name:   "zone snapshot data",
			input:  `{"kind":"zone","id":"UTC","offset_now":"+00:00","dst":false}`,
			decode: func(b []byte) error { var v Zone; return json.Unmarshal(b, &v) },
		},
		{
			name:   "date extra field",
			input:  `{"kind":"date","value":"2026-03-27","calendar":"iso8601","weekday":"Friday"}`,
			decode: func(b []byte) error { var v Date; return json.Unmarshal(b, &v) },
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := tc.decode([]byte(tc.input)); err == nil {
				t.Fatalf("Unmarshal(%s) error = nil, want error", tc.input)
			}
		})
	}
}

func TestJSONUnmarshalRejectsContradictoryDerivedFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		decode func([]byte) error
	}{
		{
			name:   "instant epoch mismatch",
			input:  `{"kind":"instant","iso":"1970-01-01T00:00:00Z","epoch_ms":1}`,
			decode: func(b []byte) error { var v Instant; return json.Unmarshal(b, &v) },
		},
		{
			name:   "time precision mismatch",
			input:  `{"kind":"time","value":"13:30:45.123456789","precision":"second"}`,
			decode: func(b []byte) error { var v Time; return json.Unmarshal(b, &v) },
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := tc.decode([]byte(tc.input)); err == nil {
				t.Fatalf("Unmarshal(%s) error = nil, want error", tc.input)
			}
		})
	}
}

func TestJSONUnmarshalRejectsUnsupportedCalendar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		decode func([]byte) error
	}{
		{
			name:   "date",
			input:  `{"kind":"date","value":"2026-03-27","calendar":"gregory"}`,
			decode: func(b []byte) error { var v Date; return json.Unmarshal(b, &v) },
		},
		{
			name:   "datetime",
			input:  `{"kind":"datetime","value":"2026-03-27T13:00:00+09:00","zone":"Asia/Tokyo","calendar":"gregory"}`,
			decode: func(b []byte) error { var v DateTime; return json.Unmarshal(b, &v) },
		},
		{
			name:   "local datetime",
			input:  `{"kind":"local_datetime","value":"2026-03-27T13:00:00","calendar":"gregory"}`,
			decode: func(b []byte) error { var v LocalDateTime; return json.Unmarshal(b, &v) },
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := tc.decode([]byte(tc.input)); err == nil {
				t.Fatalf("Unmarshal(%s) error = nil, want error", tc.input)
			}
		})
	}
}

func TestJSONUnmarshalRejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		decode func([]byte) error
	}{
		{
			name:   "date value",
			input:  `{"kind":"date","calendar":"iso8601"}`,
			decode: func(b []byte) error { var v Date; return json.Unmarshal(b, &v) },
		},
		{
			name:   "datetime calendar",
			input:  `{"kind":"datetime","value":"2026-03-27T13:00:00+09:00","zone":"Asia/Tokyo"}`,
			decode: func(b []byte) error { var v DateTime; return json.Unmarshal(b, &v) },
		},
		{
			name:   "duration iso",
			input:  `{"kind":"duration"}`,
			decode: func(b []byte) error { var v Duration; return json.Unmarshal(b, &v) },
		},
		{
			name:   "instant epoch",
			input:  `{"kind":"instant","iso":"1970-01-01T00:00:00Z"}`,
			decode: func(b []byte) error { var v Instant; return json.Unmarshal(b, &v) },
		},
		{
			name:   "time precision",
			input:  `{"kind":"time","value":"13:30:45"}`,
			decode: func(b []byte) error { var v Time; return json.Unmarshal(b, &v) },
		},
		{
			name:   "zone id",
			input:  `{"kind":"zone"}`,
			decode: func(b []byte) error { var v Zone; return json.Unmarshal(b, &v) },
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := tc.decode([]byte(tc.input)); err == nil {
				t.Fatalf("Unmarshal(%s) error = nil, want error", tc.input)
			}
		})
	}
}
