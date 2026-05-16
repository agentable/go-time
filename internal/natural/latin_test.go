package natural

import (
	"testing"
	"time"
)

var latinBase = time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)

// ── French ───────────────────────────────────────────────────────────────────

func TestFrenchRelativeDates(t *testing.T) {
	ctx := Context{Locale: "fr", ZoneID: "UTC", RelativeTo: latinBase}
	tests := []struct {
		input    string
		wantDays int
	}{
		{"aujourd'hui", 0},
		{"demain", 1},
		{"hier", -1},
	}
	for _, tt := range tests {
		r, ok := Parse(tt.input, ctx)
		if !ok {
			t.Errorf("Parse(%q) not ok", tt.input)
			continue
		}
		if r.Kind != KindDate {
			t.Errorf("Parse(%q) kind=%v want KindDate", tt.input, r.Kind)
			continue
		}
		wantDate := time.Date(2026, 3, 2+tt.wantDays, 0, 0, 0, 0, time.UTC)
		if !r.Time.Equal(wantDate) {
			t.Errorf("Parse(%q) time=%v want %v", tt.input, r.Time, wantDate)
		}
	}
}

func TestFrenchFutureDuration(t *testing.T) {
	ctx := Context{Locale: "fr", ZoneID: "UTC", RelativeTo: latinBase}
	tests := []struct {
		input     string
		wantNanos int64
	}{
		{"dans 2 heures", 2 * int64(time.Hour)},
		{"dans 1 minute", int64(time.Minute)},
		{"dans 3 jours", 3 * 24 * int64(time.Hour)},
		{"dans 1 semaine", 7 * 24 * int64(time.Hour)},
	}
	for _, tt := range tests {
		r, ok := Parse(tt.input, ctx)
		if !ok {
			t.Errorf("Parse(%q) not ok", tt.input)
			continue
		}
		if r.Kind != KindDuration {
			t.Errorf("Parse(%q) kind=%v want KindDuration", tt.input, r.Kind)
			continue
		}
		if r.DurNanos != tt.wantNanos {
			t.Errorf("Parse(%q) nanos=%d want %d", tt.input, r.DurNanos, tt.wantNanos)
		}
	}
}

func TestFrenchPastDuration(t *testing.T) {
	ctx := Context{Locale: "fr", ZoneID: "UTC", RelativeTo: latinBase}
	tests := []struct {
		input     string
		wantNanos int64
	}{
		{"il y a 3 jours", -3 * 24 * int64(time.Hour)},
		{"il y a 1 heure", -int64(time.Hour)},
	}
	for _, tt := range tests {
		r, ok := Parse(tt.input, ctx)
		if !ok {
			t.Errorf("Parse(%q) not ok", tt.input)
			continue
		}
		if r.Kind != KindDuration {
			t.Errorf("Parse(%q) kind=%v want KindDuration", tt.input, r.Kind)
			continue
		}
		if r.DurNanos != tt.wantNanos {
			t.Errorf("Parse(%q) nanos=%d want %d", tt.input, r.DurNanos, tt.wantNanos)
		}
	}
}

// ── German ───────────────────────────────────────────────────────────────────

func TestGermanTier2(t *testing.T) {
	ctx := Context{Locale: "de", ZoneID: "UTC", RelativeTo: latinBase}
	tests := []struct {
		input     string
		wantKind  Kind
		wantDays  int
		wantNanos int64
	}{
		{"heute", KindDate, 0, 0},
		{"morgen", KindDate, 1, 0},
		{"gestern", KindDate, -1, 0},
		{"in 2 Stunden", KindDuration, 0, 2 * int64(time.Hour)},
		{"in 1 Minute", KindDuration, 0, int64(time.Minute)},
		{"vor 3 Tagen", KindDuration, 0, -3 * 24 * int64(time.Hour)},
		{"vor 1 Stunde", KindDuration, 0, -int64(time.Hour)},
	}
	for _, tt := range tests {
		r, ok := Parse(tt.input, ctx)
		if !ok {
			t.Errorf("Parse(%q) not ok", tt.input)
			continue
		}
		if r.Kind != tt.wantKind {
			t.Errorf("Parse(%q) kind=%v want %v", tt.input, r.Kind, tt.wantKind)
			continue
		}
		if tt.wantKind == KindDate {
			wantDate := time.Date(2026, 3, 2+tt.wantDays, 0, 0, 0, 0, time.UTC)
			if !r.Time.Equal(wantDate) {
				t.Errorf("Parse(%q) time=%v want %v", tt.input, r.Time, wantDate)
			}
		} else if r.DurNanos != tt.wantNanos {
			t.Errorf("Parse(%q) nanos=%d want %d", tt.input, r.DurNanos, tt.wantNanos)
		}
	}
}

// ── Spanish ──────────────────────────────────────────────────────────────────

func TestSpanishTier2(t *testing.T) {
	ctx := Context{Locale: "es", ZoneID: "UTC", RelativeTo: latinBase}
	tests := []struct {
		input     string
		wantKind  Kind
		wantDays  int
		wantNanos int64
	}{
		{"hoy", KindDate, 0, 0},
		{"mañana", KindDate, 1, 0},
		{"ayer", KindDate, -1, 0},
		{"en 2 horas", KindDuration, 0, 2 * int64(time.Hour)},
		{"en 1 minuto", KindDuration, 0, int64(time.Minute)},
		{"hace 3 días", KindDuration, 0, -3 * 24 * int64(time.Hour)},
		{"hace 1 hora", KindDuration, 0, -int64(time.Hour)},
	}
	for _, tt := range tests {
		r, ok := Parse(tt.input, ctx)
		if !ok {
			t.Errorf("Parse(%q) not ok", tt.input)
			continue
		}
		if r.Kind != tt.wantKind {
			t.Errorf("Parse(%q) kind=%v want %v", tt.input, r.Kind, tt.wantKind)
			continue
		}
		if tt.wantKind == KindDate {
			wantDate := time.Date(2026, 3, 2+tt.wantDays, 0, 0, 0, 0, time.UTC)
			if !r.Time.Equal(wantDate) {
				t.Errorf("Parse(%q) time=%v want %v", tt.input, r.Time, wantDate)
			}
		} else if r.DurNanos != tt.wantNanos {
			t.Errorf("Parse(%q) nanos=%d want %d", tt.input, r.DurNanos, tt.wantNanos)
		}
	}
}

// ── Portuguese ───────────────────────────────────────────────────────────────

func TestPortugueseTier2(t *testing.T) {
	ctx := Context{Locale: "pt", ZoneID: "UTC", RelativeTo: latinBase}
	tests := []struct {
		input     string
		wantKind  Kind
		wantDays  int
		wantNanos int64
	}{
		{"hoje", KindDate, 0, 0},
		{"amanhã", KindDate, 1, 0},
		{"ontem", KindDate, -1, 0},
		{"em 2 horas", KindDuration, 0, 2 * int64(time.Hour)},
		{"em 1 minuto", KindDuration, 0, int64(time.Minute)},
		{"há 3 dias", KindDuration, 0, -3 * 24 * int64(time.Hour)},
		{"ha 1 hora", KindDuration, 0, -int64(time.Hour)},
	}
	for _, tt := range tests {
		r, ok := Parse(tt.input, ctx)
		if !ok {
			t.Errorf("Parse(%q) not ok", tt.input)
			continue
		}
		if r.Kind != tt.wantKind {
			t.Errorf("Parse(%q) kind=%v want %v", tt.input, r.Kind, tt.wantKind)
			continue
		}
		if tt.wantKind == KindDate {
			wantDate := time.Date(2026, 3, 2+tt.wantDays, 0, 0, 0, 0, time.UTC)
			if !r.Time.Equal(wantDate) {
				t.Errorf("Parse(%q) time=%v want %v", tt.input, r.Time, wantDate)
			}
		} else if r.DurNanos != tt.wantNanos {
			t.Errorf("Parse(%q) nanos=%d want %d", tt.input, r.DurNanos, tt.wantNanos)
		}
	}
}

// ── Russian ───────────────────────────────────────────────────────────────────

func TestRussianTier2(t *testing.T) {
	ctx := Context{Locale: "ru", ZoneID: "UTC", RelativeTo: latinBase}
	tests := []struct {
		input     string
		wantKind  Kind
		wantDays  int
		wantNanos int64
	}{
		{"сегодня", KindDate, 0, 0},
		{"завтра", KindDate, 1, 0},
		{"вчера", KindDate, -1, 0},
		{"через 2 часа", KindDuration, 0, 2 * int64(time.Hour)},
		{"через 1 минуту", KindDuration, 0, int64(time.Minute)},
		{"3 дня назад", KindDuration, 0, -3 * 24 * int64(time.Hour)},
		{"1 час назад", KindDuration, 0, -int64(time.Hour)},
	}
	for _, tt := range tests {
		r, ok := Parse(tt.input, ctx)
		if !ok {
			t.Errorf("Parse(%q) not ok", tt.input)
			continue
		}
		if r.Kind != tt.wantKind {
			t.Errorf("Parse(%q) kind=%v want %v", tt.input, r.Kind, tt.wantKind)
			continue
		}
		if tt.wantKind == KindDate {
			wantDate := time.Date(2026, 3, 2+tt.wantDays, 0, 0, 0, 0, time.UTC)
			if !r.Time.Equal(wantDate) {
				t.Errorf("Parse(%q) time=%v want %v", tt.input, r.Time, wantDate)
			}
		} else if r.DurNanos != tt.wantNanos {
			t.Errorf("Parse(%q) nanos=%d want %d", tt.input, r.DurNanos, tt.wantNanos)
		}
	}
}

// ── exact seconds and calendar months ────────────────────────────────────────

func TestFrenchSecondsAndMonths(t *testing.T) {
	ctx := Context{Locale: "fr", ZoneID: "UTC", RelativeTo: latinBase}
	durationTests := []struct {
		input string
		want  int64
	}{
		{"dans 30 secondes", 30 * int64(time.Second)},
		{"il y a 1 seconde", -int64(time.Second)},
	}
	for _, tt := range durationTests {
		r, ok := Parse(tt.input, ctx)
		if !ok {
			t.Errorf("Parse(%q) not ok", tt.input)
			continue
		}
		if r.Kind != KindDuration {
			t.Errorf("Parse(%q) kind=%v want KindDuration", tt.input, r.Kind)
			continue
		}
		if r.DurNanos != tt.want {
			t.Errorf("Parse(%q) nanos=%d want %d", tt.input, r.DurNanos, tt.want)
		}
	}

	periodTests := []struct {
		input      string
		wantMonths int32
	}{
		{"dans 2 mois", 2},
		{"il y a 3 mois", -3},
	}
	for _, tt := range periodTests {
		r, ok := Parse(tt.input, ctx)
		if !ok {
			t.Errorf("Parse(%q) not ok", tt.input)
			continue
		}
		if r.Kind != KindPeriod {
			t.Errorf("Parse(%q) kind=%v want KindPeriod", tt.input, r.Kind)
			continue
		}
		if r.PeriodMonths != tt.wantMonths {
			t.Errorf("Parse(%q) months=%d want %d", tt.input, r.PeriodMonths, tt.wantMonths)
		}
	}
}

// ── Russian prefix-based unit lookup ─────────────────────────────────────────

func TestRussianUnitPrefixes(t *testing.T) {
	ctx := Context{Locale: "ru", ZoneID: "UTC", RelativeTo: latinBase}
	tests := []struct {
		input      string
		wantNanos  int64
		wantMonths int32
	}{
		// недели/недель covers prefix "недел"
		{input: "через 2 недели", wantNanos: 2 * 7 * 24 * int64(time.Hour)},
		{input: "1 неделю назад", wantNanos: -7 * 24 * int64(time.Hour)},
		// дней/дня covers prefix "ден" and "дн"
		{input: "5 дней назад", wantNanos: -5 * 24 * int64(time.Hour)},
		{input: "через 1 день", wantNanos: 24 * int64(time.Hour)},
		// месяца/месяцев covers prefix "месяц"
		{input: "через 2 месяца", wantMonths: 2},
		{input: "3 месяца назад", wantMonths: -3},
		// секунд/секунды covers prefix "секунд"
		{input: "через 10 секунд", wantNanos: 10 * int64(time.Second)},
		// минут covers prefix "минут"
		{input: "через 5 минут", wantNanos: 5 * int64(time.Minute)},
	}
	for _, tt := range tests {
		r, ok := Parse(tt.input, ctx)
		if !ok {
			t.Errorf("Parse(%q) not ok", tt.input)
			continue
		}
		if tt.wantMonths != 0 {
			if r.Kind != KindPeriod {
				t.Errorf("Parse(%q) kind=%v want KindPeriod", tt.input, r.Kind)
				continue
			}
			if r.PeriodMonths != tt.wantMonths {
				t.Errorf("Parse(%q) months=%d want %d", tt.input, r.PeriodMonths, tt.wantMonths)
			}
			continue
		}
		if r.Kind != KindDuration {
			t.Errorf("Parse(%q) kind=%v want KindDuration", tt.input, r.Kind)
			continue
		}
		if r.DurNanos != tt.wantNanos {
			t.Errorf("Parse(%q) nanos=%d want %d", tt.input, r.DurNanos, tt.wantNanos)
		}
	}
}

// ── Locale prefix matching ────────────────────────────────────────────────────

func TestLatinLocalePrefixMatching(t *testing.T) {
	tests := []struct {
		locale string
		input  string
	}{
		{"fr-FR", "aujourd'hui"},
		{"fr-CA", "demain"},
		{"de-DE", "heute"},
		{"de-AT", "morgen"},
		{"es-ES", "hoy"},
		{"es-MX", "mañana"},
		{"pt-BR", "hoje"},
		{"pt-PT", "amanhã"},
		{"ru-RU", "сегодня"},
	}
	for _, tt := range tests {
		ctx := Context{Locale: tt.locale, ZoneID: "UTC", RelativeTo: latinBase}
		_, ok := Parse(tt.input, ctx)
		if !ok {
			t.Errorf("Parse(%q, locale=%q) not ok — locale prefix matching failed", tt.input, tt.locale)
		}
	}
}
