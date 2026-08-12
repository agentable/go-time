package natural

import (
	"testing"
	"time"
)

var latinBase = time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)

// ── French ───────────────────────────────────────────────────────────────────

func TestFrenchRelativeDates(t *testing.T) {
	ctx := Context{Locale: "fr", RelativeTo: latinBase}
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
		if !equalCivil(r, wantDate) {
			t.Errorf("Parse(%q) time=%v want %v", tt.input, civilTime(r), wantDate)
		}
	}
}

func TestFrenchFutureDuration(t *testing.T) {
	ctx := Context{Locale: "fr", RelativeTo: latinBase}
	tests := []struct {
		input     string
		wantNanos int64
	}{
		{"dans 2 heures", 2 * int64(time.Hour)},
		{"dans 1 minute", int64(time.Minute)},
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

func TestFrenchFuturePeriod(t *testing.T) {
	ctx := Context{Locale: "fr", RelativeTo: latinBase}
	tests := []struct {
		input    string
		wantDays int32
	}{
		{"dans 3 jours", 3},
		{"dans 1 semaine", 7},
	}
	for _, tt := range tests {
		r, ok := Parse(tt.input, ctx)
		if !ok {
			t.Errorf("Parse(%q) not ok", tt.input)
			continue
		}
		if r.Kind != KindPeriod {
			t.Errorf("Parse(%q) kind=%v want KindPeriod", tt.input, r.Kind)
			continue
		}
		if r.PeriodDays != tt.wantDays {
			t.Errorf("Parse(%q) days=%d want %d", tt.input, r.PeriodDays, tt.wantDays)
		}
	}
}

func TestFrenchPastDuration(t *testing.T) {
	ctx := Context{Locale: "fr", RelativeTo: latinBase}
	tests := []struct {
		input     string
		wantNanos int64
	}{
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

func TestFrenchPastPeriod(t *testing.T) {
	ctx := Context{Locale: "fr", RelativeTo: latinBase}
	r, ok := Parse("il y a 3 jours", ctx)
	if !ok {
		t.Fatal("Parse returned ok=false")
	}
	if r.Kind != KindPeriod {
		t.Fatalf("Kind = %v, want KindPeriod", r.Kind)
	}
	if r.PeriodDays != -3 {
		t.Errorf("PeriodDays = %d, want -3", r.PeriodDays)
	}
}

// ── German ───────────────────────────────────────────────────────────────────

func TestGermanTier2(t *testing.T) {
	ctx := Context{Locale: "de", RelativeTo: latinBase}
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
		{"vor 3 Tagen", KindPeriod, -3, 0},
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
		switch tt.wantKind {
		case KindDate:
			wantDate := time.Date(2026, 3, 2+tt.wantDays, 0, 0, 0, 0, time.UTC)
			if !equalCivil(r, wantDate) {
				t.Errorf("Parse(%q) time=%v want %v", tt.input, civilTime(r), wantDate)
			}
		case KindPeriod:
			if r.PeriodDays != int32(tt.wantDays) {
				t.Errorf("Parse(%q) days=%d want %d", tt.input, r.PeriodDays, tt.wantDays)
			}
		case KindDuration:
			if r.DurNanos != tt.wantNanos {
				t.Errorf("Parse(%q) nanos=%d want %d", tt.input, r.DurNanos, tt.wantNanos)
			}
		default:
			t.Fatalf("unhandled wantKind %v", tt.wantKind)
		}
	}
}

// ── Spanish ──────────────────────────────────────────────────────────────────

func TestSpanishTier2(t *testing.T) {
	ctx := Context{Locale: "es", RelativeTo: latinBase}
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
		{"hace 3 días", KindPeriod, -3, 0},
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
		switch tt.wantKind {
		case KindDate:
			wantDate := time.Date(2026, 3, 2+tt.wantDays, 0, 0, 0, 0, time.UTC)
			if !equalCivil(r, wantDate) {
				t.Errorf("Parse(%q) time=%v want %v", tt.input, civilTime(r), wantDate)
			}
		case KindPeriod:
			if r.PeriodDays != int32(tt.wantDays) {
				t.Errorf("Parse(%q) days=%d want %d", tt.input, r.PeriodDays, tt.wantDays)
			}
		case KindDuration:
			if r.DurNanos != tt.wantNanos {
				t.Errorf("Parse(%q) nanos=%d want %d", tt.input, r.DurNanos, tt.wantNanos)
			}
		default:
			t.Fatalf("unhandled wantKind %v", tt.wantKind)
		}
	}
}

// ── Portuguese ───────────────────────────────────────────────────────────────

func TestPortugueseTier2(t *testing.T) {
	ctx := Context{Locale: "pt", RelativeTo: latinBase}
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
		{"há 3 dias", KindPeriod, -3, 0},
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
		switch tt.wantKind {
		case KindDate:
			wantDate := time.Date(2026, 3, 2+tt.wantDays, 0, 0, 0, 0, time.UTC)
			if !equalCivil(r, wantDate) {
				t.Errorf("Parse(%q) time=%v want %v", tt.input, civilTime(r), wantDate)
			}
		case KindPeriod:
			if r.PeriodDays != int32(tt.wantDays) {
				t.Errorf("Parse(%q) days=%d want %d", tt.input, r.PeriodDays, tt.wantDays)
			}
		case KindDuration:
			if r.DurNanos != tt.wantNanos {
				t.Errorf("Parse(%q) nanos=%d want %d", tt.input, r.DurNanos, tt.wantNanos)
			}
		default:
			t.Fatalf("unhandled wantKind %v", tt.wantKind)
		}
	}
}

// ── Russian ───────────────────────────────────────────────────────────────────

func TestRussianTier2(t *testing.T) {
	ctx := Context{Locale: "ru", RelativeTo: latinBase}
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
		{"3 дня назад", KindPeriod, -3, 0},
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
		switch tt.wantKind {
		case KindDate:
			wantDate := time.Date(2026, 3, 2+tt.wantDays, 0, 0, 0, 0, time.UTC)
			if !equalCivil(r, wantDate) {
				t.Errorf("Parse(%q) time=%v want %v", tt.input, civilTime(r), wantDate)
			}
		case KindPeriod:
			if r.PeriodDays != int32(tt.wantDays) {
				t.Errorf("Parse(%q) days=%d want %d", tt.input, r.PeriodDays, tt.wantDays)
			}
		case KindDuration:
			if r.DurNanos != tt.wantNanos {
				t.Errorf("Parse(%q) nanos=%d want %d", tt.input, r.DurNanos, tt.wantNanos)
			}
		default:
			t.Fatalf("unhandled wantKind %v", tt.wantKind)
		}
	}
}

// ── exact seconds and calendar months ────────────────────────────────────────

func TestFrenchSecondsAndMonths(t *testing.T) {
	ctx := Context{Locale: "fr", RelativeTo: latinBase}
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
	ctx := Context{Locale: "ru", RelativeTo: latinBase}
	tests := []struct {
		input      string
		wantNanos  int64
		wantMonths int32
		wantDays   int32
	}{
		// недели/недель covers prefix "недел"
		{input: "через 2 недели", wantDays: 14},
		{input: "1 неделю назад", wantDays: -7},
		// дней/дня covers prefix "ден" and "дн"
		{input: "5 дней назад", wantDays: -5},
		{input: "через 1 день", wantDays: 1},
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
		if tt.wantDays != 0 {
			if r.Kind != KindPeriod {
				t.Errorf("Parse(%q) kind=%v want KindPeriod", tt.input, r.Kind)
				continue
			}
			if r.PeriodDays != tt.wantDays {
				t.Errorf("Parse(%q) days=%d want %d", tt.input, r.PeriodDays, tt.wantDays)
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
		wantOK bool
	}{
		{"fr-FR", "aujourd'hui", true},
		{"fr-CA", "demain", true},
		{"de-DE", "heute", true},
		{"de-AT", "morgen", true},
		{"es-ES", "hoy", true},
		{"es-MX", "mañana", true},
		{"pt-BR", "hoje", true},
		{"pt-PT", "amanhã", true},
		{"ru-RU", "сегодня", true},
		{"fr-FR", "pas une date", false},
	}
	for _, tt := range tests {
		ctx := Context{Locale: tt.locale, RelativeTo: latinBase}
		_, ok := Parse(tt.input, ctx)
		if ok != tt.wantOK {
			t.Errorf("Parse(%q, locale=%q) ok = %v, want %v", tt.input, tt.locale, ok, tt.wantOK)
		}
	}
}
