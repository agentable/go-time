package gotime

import (
	"errors"
	"fmt"
	"testing"

	"golang.org/x/text/language"
)

func TestParseSlashDistinctDates(t *testing.T) {
	t.Parallel()
	for _, locale := range []language.Tag{language.Und, language.English, language.AmericanEnglish, language.BritishEnglish} {
		for month := 1; month <= 12; month++ {
			input := fmt.Sprintf("%02d/%02d/2026", month, month)
			d, err := ParseDate(input, WithInputLocale(locale))
			if err != nil || int(d.Month()) != month || d.Day() != month {
				t.Errorf("%s %s: %v %v", locale, input, d, err)
			}
		}
	}
	for _, locale := range []language.Tag{language.Und, language.English} {
		r := Parse("04/05/2026", WithInputLocale(locale))
		if r.Status != StatusAmbiguous || len(r.Candidates) != 2 {
			t.Errorf("%s: ambiguity missing", locale)
		}
		if _, err := ParseDate("13/02/2026", WithInputLocale(locale)); err != nil {
			t.Error(err)
		}
		if _, err := ParseDate("31/02/2026", WithInputLocale(locale)); !errors.Is(err, ErrInvalidDate) {
			t.Error(err)
		}
	}
	if _, err := ParseDate("13/02/2026", WithInputLocale(language.AmericanEnglish)); !errors.Is(err, ErrInvalidDate) {
		t.Error(err)
	}
}
