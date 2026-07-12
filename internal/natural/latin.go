package natural

import (
	"regexp"
	"strconv"
	"strings"
)

// latinVocab holds the vocabulary for one Latin-script (or Cyrillic) locale family.
type latinVocab struct {
	// localePrefix is the BCP 47 language subtag (e.g. "fr", "de").
	localePrefix string
	// Relative-date keywords.
	today, tomorrow, yesterday string
	// futurePat matches "prefix N unit" future expressions.
	// Group 1: number, Group 2: unit word.
	futurePat *regexp.Regexp
	// pastPat matches past expressions.
	// Group 1: number, Group 2: unit word (or number-first for Russian).
	pastPat *regexp.Regexp
	// units maps unit words (lower-case) to canonical unit names.
	units map[string]string
	// unitPrefixes maps a prefix string to a canonical unit name,
	// used for languages with grammatical case variations (e.g. Russian).
	unitPrefixes []struct{ prefix, canonical string }
}

// latinVocabs is the dispatch table for all supported Latin/Cyrillic locales.
var latinVocabs = []latinVocab{
	{
		localePrefix: "fr",
		today:        "aujourd'hui",
		tomorrow:     "demain",
		yesterday:    "hier",
		futurePat:    regexp.MustCompile(`(?i)^dans\s+(\d+)\s+(\S+)$`),
		pastPat:      regexp.MustCompile(`(?i)^il\s+y\s+a\s+(\d+)\s+(\S+)$`),
		units: map[string]string{
			"seconde":  "second",
			"secondes": "second",
			"minute":   "minute",
			"minutes":  "minute",
			"heure":    "hour",
			"heures":   "hour",
			"jour":     "day",
			"jours":    "day",
			"semaine":  "week",
			"semaines": "week",
			"mois":     "month",
		},
	},
	{
		localePrefix: "de",
		today:        "heute",
		tomorrow:     "morgen",
		yesterday:    "gestern",
		futurePat:    regexp.MustCompile(`(?i)^in\s+(\d+)\s+(\S+)$`),
		pastPat:      regexp.MustCompile(`(?i)^vor\s+(\d+)\s+(\S+)$`),
		units: map[string]string{
			"sekunde":  "second",
			"sekunden": "second",
			"minute":   "minute",
			"minuten":  "minute",
			"stunde":   "hour",
			"stunden":  "hour",
			"tag":      "day",
			"tagen":    "day",
			"tage":     "day",
			"woche":    "week",
			"wochen":   "week",
			"monat":    "month",
			"monate":   "month",
			"monaten":  "month",
		},
	},
	{
		localePrefix: "es",
		today:        "hoy",
		tomorrow:     "mañana",
		yesterday:    "ayer",
		futurePat:    regexp.MustCompile(`(?i)^en\s+(\d+)\s+(\S+)$`),
		pastPat:      regexp.MustCompile(`(?i)^hace\s+(\d+)\s+(\S+)$`),
		units: map[string]string{
			"segundo":  "second",
			"segundos": "second",
			"minuto":   "minute",
			"minutos":  "minute",
			"hora":     "hour",
			"horas":    "hour",
			"día":      "day",
			"días":     "day",
			"dia":      "day",
			"dias":     "day",
			"semana":   "week",
			"semanas":  "week",
			"mes":      "month",
			"meses":    "month",
		},
	},
	{
		localePrefix: "pt",
		today:        "hoje",
		tomorrow:     "amanhã",
		yesterday:    "ontem",
		futurePat:    regexp.MustCompile(`(?i)^em\s+(\d+)\s+(\S+)$`),
		pastPat:      regexp.MustCompile(`(?i)^h[aá]\s+(\d+)\s+(\S+)$`),
		units: map[string]string{
			"segundo":  "second",
			"segundos": "second",
			"minuto":   "minute",
			"minutos":  "minute",
			"hora":     "hour",
			"horas":    "hour",
			"dia":      "day",
			"dias":     "day",
			"semana":   "week",
			"semanas":  "week",
			"mês":      "month",
			"mes":      "month",
			"meses":    "month",
		},
	},
	{
		localePrefix: "ru",
		today:        "сегодня",
		tomorrow:     "завтра",
		yesterday:    "вчера",
		// Future: "через N unit"
		futurePat: regexp.MustCompile(`^через\s+(\d+)\s+(\S+)$`),
		// Past: "N unit назад" — number comes first
		pastPat: regexp.MustCompile(`^(\d+)\s+(\S+)\s+назад$`),
		// unitPrefixes handles Russian grammatical case variations.
		// Exact units map is kept minimal; prefix matching covers the rest.
		units: map[string]string{
			"секунду": "second",
			"минуту":  "minute",
			"час":     "hour",
		},
		unitPrefixes: []struct{ prefix, canonical string }{
			{"секунд", "second"},
			{"минут", "minute"},
			{"час", "hour"},
			{"ден", "day"}, // день/дней/дня
			{"дн", "day"},  // дней (alternative stem)
			{"недел", "week"},
			{"месяц", "month"},
		},
	},
}

// latinParser dispatches to the appropriate vocab based on locale.
type latinParser struct{}

func init() {
	register(&latinParser{})
}

func (p *latinParser) canHandle(locale string) bool {
	return latinVocabFor(locale) != nil
}

func (p *latinParser) parse(input string, ctx Context) (Result, bool) {
	v := latinVocabFor(ctx.Locale)
	if v == nil {
		return Result{}, false
	}

	lower := strings.ToLower(strings.TrimSpace(input))
	base := midnight(ctx.RelativeTo)

	switch lower {
	case strings.ToLower(v.today):
		return dateResult(base), true
	case strings.ToLower(v.tomorrow):
		return dateResult(base.AddDate(0, 0, 1)), true
	case strings.ToLower(v.yesterday):
		return dateResult(base.AddDate(0, 0, -1)), true
	}

	if v.futurePat != nil {
		if m := v.futurePat.FindStringSubmatch(input); m != nil {
			n, _ := strconv.ParseInt(m[1], 10, 64)
			if canonical, ok := v.lookupUnit(strings.ToLower(m[2])); ok {
				return relativeUnitResult(canonical, n), true
			}
		}
	}

	if v.pastPat != nil {
		if m := v.pastPat.FindStringSubmatch(input); m != nil {
			n, _ := strconv.ParseInt(m[1], 10, 64)
			if canonical, ok := v.lookupUnit(strings.ToLower(m[2])); ok {
				return relativeUnitResult(canonical, -n), true
			}
		}
	}

	return Result{}, false
}

// latinVocabFor returns the vocab entry matching the given locale, or nil.
func latinVocabFor(locale string) *latinVocab {
	for i := range latinVocabs {
		v := &latinVocabs[i]
		if matchesLocalePrefix(locale, v.localePrefix) {
			return v
		}
	}
	return nil
}

// lookupUnit resolves a (lowercased) unit word to its canonical name.
// It checks the exact units map first, then falls back to prefix matching.
func (v *latinVocab) lookupUnit(word string) (string, bool) {
	if canonical, ok := v.units[word]; ok {
		return canonical, true
	}
	for _, p := range v.unitPrefixes {
		if strings.HasPrefix(word, p.prefix) {
			return p.canonical, true
		}
	}
	return "", false
}
