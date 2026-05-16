package gotime

import "golang.org/x/text/language"

// config accumulates option values for Parse.
type config struct {
	lang       language.Tag
	zone       Zone
	relativeTo Instant
}

// Option is a functional option for Parse.
type Option func(*config)

// WithInputLocale sets the language used to interpret natural-language input.
// Standard formats (RFC 3339, ISO 8601) are language-independent and ignore it.
//
// The argument is a golang.org/x/text/language Tag — the de facto standard
// representation of BCP-47 language tags in Go. Construct it via
// language.MustParse("zh-Hans"), language.Chinese, or similar.
//
// Only the language identity is consumed; Unicode -u- extensions (hour
// cycle, calendar, numbering system) are ignored because they describe
// display behavior, not parsing behavior.
//
// Chinese requires script subtags: language.SimplifiedChinese (zh-Hans) or
// language.TraditionalChinese (zh-Hant). The bare language.Chinese ("zh")
// tag does not activate the Chinese natural-language parser.
//
// For slash dates (e.g. "04/05/2026"), the locale also disambiguates
// month-first (en-US, en-CA, en-AU) vs day-first ordering. With no locale,
// only-valid interpretations resolve, otherwise the result is Ambiguous
// with both candidates.
func WithInputLocale(tag language.Tag) Option {
	return func(c *config) { c.lang = tag }
}

// WithZone sets the timezone used when the input has no explicit offset.
func WithZone(zone Zone) Option {
	return func(c *config) { c.zone = zone }
}

// WithReference sets the reference instant for relative-time expressions.
func WithReference(t Instant) Option {
	return func(c *config) { c.relativeTo = t }
}

// applyOptions applies a slice of options to a config and returns the result.
func applyOptions(opts []Option) config {
	var cfg config
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}
