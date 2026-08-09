package gotime

import "golang.org/x/text/language"

// config accumulates option values for Parse.
type config struct {
	lang         language.Tag
	zone         Zone
	zoneSet      bool
	relativeTo   Instant
	referenceSet bool
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
// For slash dates (e.g. "04/05/2026"), en-US selects month-first while en-GB
// and en-AU select day-first. Unsupported tags, including bare en and en-CA,
// use validity inference: one valid interpretation resolves, two are
// Ambiguous, and zero are Invalid. Unicode -u- extensions do not change the
// supported locale's order.
func WithInputLocale(tag language.Tag) Option {
	return func(c *config) { c.lang = tag }
}

// WithZone sets the timezone used when the input has no explicit offset.
// Relative natural date and datetime expressions require it so WithReference's
// instant can be interpreted in an explicit calendar frame.
func WithZone(zone Zone) Option {
	return func(c *config) {
		c.zone = zone
		c.zoneSet = true
	}
}

// WithReference sets the reference instant for relative-time expressions.
// Relative natural date and datetime expressions also require WithZone.
func WithReference(t Instant) Option {
	return func(c *config) {
		c.relativeTo = t
		c.referenceSet = true
	}
}

// applyOptions applies a slice of options to a config and returns the result.
func applyOptions(opts []Option) config {
	var cfg config
	for _, o := range opts {
		if o == nil {
			continue
		}
		o(&cfg)
	}
	return cfg
}
