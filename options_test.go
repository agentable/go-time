package gotime

import (
	"testing"

	"golang.org/x/text/language"
)

func TestWithInputLocale(t *testing.T) {
	cfg := &config{}
	tag := language.MustParse("zh-Hans")
	WithInputLocale(tag)(cfg)
	if cfg.lang != tag {
		t.Errorf("WithInputLocale set lang = %q, want %q", cfg.lang, tag)
	}
}

func TestWithZone(t *testing.T) {
	cfg := &config{}
	zone := MustLoadZone("Asia/Tokyo")
	WithZone(zone)(cfg)
	if !cfg.zoneSet {
		t.Fatal("WithZone did not record option presence")
	}
	if !cfg.zone.Equal(zone) {
		t.Errorf("WithZone set zone = %q, want %q", cfg.zone.ID(), zone.ID())
	}
}

func TestWithReference(t *testing.T) {
	now := Now()
	cfg := &config{}
	WithReference(now)(cfg)
	if !cfg.referenceSet {
		t.Fatal("WithReference did not record option presence")
	}
	if !cfg.relativeTo.Equal(now) {
		t.Error("WithReference did not set relativeTo correctly")
	}
}

func TestOptions_Composable(t *testing.T) {
	cfg := &config{}
	enUS := language.MustParse("en-US")
	for _, opt := range []Option{
		WithInputLocale(enUS),
		WithZone(MustLoadZone("America/New_York")),
	} {
		opt(cfg)
	}
	if cfg.lang != enUS {
		t.Errorf("lang = %q, want %q", cfg.lang, enUS)
	}
	if cfg.zone.ID() != "America/New_York" {
		t.Errorf("zone = %q, want %q", cfg.zone.ID(), "America/New_York")
	}
}

func TestParsers_IgnoreNilOption(t *testing.T) {
	t.Parallel()

	t.Run("diagnostic parser", func(t *testing.T) {
		t.Parallel()

		result := Parse("2026-03-27", nil)
		date, ok := result.Date()
		if result.Status != StatusResolved || !ok || date.String() != "2026-03-27" {
			t.Fatalf("Parse() = %#v, want resolved Date 2026-03-27", result)
		}
	})

	t.Run("typed parser", func(t *testing.T) {
		t.Parallel()

		date, err := ParseDate("2026-03-27", nil)
		if err != nil {
			t.Fatalf("ParseDate() error = %v", err)
		}
		if got := date.String(); got != "2026-03-27" {
			t.Errorf("ParseDate() = %s, want 2026-03-27", got)
		}
	})
}

func TestOptions_LastWriterWins(t *testing.T) {
	cfg := &config{}
	WithInputLocale(language.MustParse("en-US"))(cfg)
	zh := language.MustParse("zh-Hans")
	WithInputLocale(zh)(cfg)
	if cfg.lang != zh {
		t.Errorf("lang = %q, want %q (last writer wins)", cfg.lang, zh)
	}
}
