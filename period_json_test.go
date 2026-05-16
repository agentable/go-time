package gotime

import (
	"testing"

	"github.com/go-json-experiment/json"
)

func TestPeriodMarshalJSON_Months(t *testing.T) {
	p := Months(3)
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	want := `{"kind":"period","iso":"P3M"}`
	if string(b) != want {
		t.Errorf("got %s, want %s", b, want)
	}
}
