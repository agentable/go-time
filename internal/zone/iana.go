// Package zone provides internal IANA timezone data and DST projection utilities.
package zone

import (
	"slices"
	"time"
)

// DSTStatus describes the DST classification of a projected local time.
type DSTStatus int

const (
	// DSTNormal means the local time maps to exactly one UTC instant.
	DSTNormal DSTStatus = iota
	// DSTNonexistent means the local time falls in a spring-forward gap (no such wall time exists).
	DSTNonexistent
	// DSTAmbiguous means the local time falls in a fall-back overlap (two UTC instants map to it).
	DSTAmbiguous
)

// LocalTimeResult holds the outcome of projecting a local wall-clock time into a timezone.
type LocalTimeResult struct {
	// Status classifies the local time as normal, nonexistent, or ambiguous.
	Status DSTStatus
	// Times holds the matching instants in chronological order.
	Times []time.Time
}

// ProjectLocalTime projects a local wall-clock time into loc and classifies the DST status.
//
// Algorithm:
//  1. Construct t = time.Date(year, month, day, hour, minute, second, 0, loc).
//  2. Spring-forward detection: if t's actual H:M:S differs from requested → DSTNonexistent.
//  3. Fall-back detection: Go may resolve an ambiguous local time to either the first or second
//     UTC instant. We probe both ±1h neighbours; if either has a different UTC offset but the
//     same local H:M:S as the requested time, the time is ambiguous.
//  4. Otherwise → DSTNormal with a single instant.
func ProjectLocalTime(loc *time.Location, year int, month time.Month, day, hour, minute, second int) LocalTimeResult {
	if loc == nil {
		loc = time.UTC
	}

	t := time.Date(year, month, day, hour, minute, second, 0, loc)

	// Spring-forward check: the wall clock jumped forward, so the requested time doesn't exist.
	if t.Hour() != hour || t.Minute() != minute || t.Second() != second {
		return LocalTimeResult{Status: DSTNonexistent}
	}

	_, off0 := t.Zone()

	// Check 1 hour forward in UTC (Go resolved to the first/earlier instance).
	tFwd := t.UTC().Add(time.Hour).In(loc)
	_, offFwd := tFwd.Zone()
	if offFwd != off0 && tFwd.Hour() == hour && tFwd.Minute() == minute && tFwd.Second() == second {
		return LocalTimeResult{
			Status: DSTAmbiguous,
			Times:  []time.Time{t, tFwd},
		}
	}

	// Check 1 hour backward in UTC (Go resolved to the second/later instance).
	tBwd := t.UTC().Add(-time.Hour).In(loc)
	_, offBwd := tBwd.Zone()
	if offBwd != off0 && tBwd.Hour() == hour && tBwd.Minute() == minute && tBwd.Second() == second {
		return LocalTimeResult{
			Status: DSTAmbiguous,
			Times:  []time.Time{tBwd, t},
		}
	}

	return LocalTimeResult{
		Status: DSTNormal,
		Times:  []time.Time{t},
	}
}

// Zones is a sorted list of canonical IANA timezone identifiers embedded at build time.
// Callers must not mutate this slice.
var Zones = func() []string {
	raw := []string{
		// Africa
		"Africa/Abidjan", "Africa/Accra", "Africa/Addis_Ababa", "Africa/Algiers",
		"Africa/Asmara", "Africa/Bamako", "Africa/Bangui", "Africa/Banjul",
		"Africa/Bissau", "Africa/Blantyre", "Africa/Brazzaville", "Africa/Bujumbura",
		"Africa/Cairo", "Africa/Casablanca", "Africa/Ceuta", "Africa/Conakry",
		"Africa/Dakar", "Africa/Dar_es_Salaam", "Africa/Djibouti", "Africa/Douala",
		"Africa/El_Aaiun", "Africa/Freetown", "Africa/Gaborone", "Africa/Harare",
		"Africa/Johannesburg", "Africa/Juba", "Africa/Kampala", "Africa/Khartoum",
		"Africa/Kigali", "Africa/Kinshasa", "Africa/Lagos", "Africa/Libreville",
		"Africa/Lome", "Africa/Luanda", "Africa/Lubumbashi", "Africa/Lusaka",
		"Africa/Malabo", "Africa/Maputo", "Africa/Maseru", "Africa/Mbabane",
		"Africa/Mogadishu", "Africa/Monrovia", "Africa/Nairobi", "Africa/Ndjamena",
		"Africa/Niamey", "Africa/Nouakchott", "Africa/Ouagadougou", "Africa/Porto-Novo",
		"Africa/Sao_Tome", "Africa/Tripoli", "Africa/Tunis", "Africa/Windhoek",

		// America
		"America/Adak", "America/Anchorage", "America/Anguilla", "America/Antigua",
		"America/Araguaina", "America/Argentina/Buenos_Aires", "America/Argentina/Catamarca",
		"America/Argentina/Cordoba", "America/Argentina/Jujuy", "America/Argentina/La_Rioja",
		"America/Argentina/Mendoza", "America/Argentina/Rio_Gallegos", "America/Argentina/Salta",
		"America/Argentina/San_Juan", "America/Argentina/San_Luis", "America/Argentina/Tucuman",
		"America/Argentina/Ushuaia", "America/Aruba", "America/Asuncion", "America/Atikokan",
		"America/Bahia", "America/Bahia_Banderas", "America/Barbados", "America/Belem",
		"America/Belize", "America/Blanc-Sablon", "America/Boa_Vista", "America/Bogota",
		"America/Boise", "America/Cambridge_Bay", "America/Campo_Grande", "America/Cancun",
		"America/Caracas", "America/Cayenne", "America/Cayman", "America/Chicago",
		"America/Chihuahua", "America/Costa_Rica", "America/Creston", "America/Cuiaba",
		"America/Curacao", "America/Danmarkshavn", "America/Dawson", "America/Dawson_Creek",
		"America/Denver", "America/Detroit", "America/Dominica", "America/Edmonton",
		"America/Eirunepe", "America/El_Salvador", "America/Fortaleza", "America/Glace_Bay",
		"America/Goose_Bay", "America/Grand_Turk", "America/Grenada", "America/Guadeloupe",
		"America/Guatemala", "America/Guayaquil", "America/Guyana", "America/Halifax",
		"America/Havana", "America/Hermosillo", "America/Indiana/Indianapolis",
		"America/Indiana/Knox", "America/Indiana/Marengo", "America/Indiana/Petersburg",
		"America/Indiana/Tell_City", "America/Indiana/Vevay", "America/Indiana/Vincennes",
		"America/Indiana/Winamac", "America/Inuvik", "America/Iqaluit", "America/Jamaica",
		"America/Juneau", "America/Kentucky/Louisville", "America/Kentucky/Monticello",
		"America/Kralendijk", "America/La_Paz", "America/Lima", "America/Los_Angeles",
		"America/Lower_Princes", "America/Maceio", "America/Managua", "America/Manaus",
		"America/Marigot", "America/Martinique", "America/Matamoros", "America/Mazatlan",
		"America/Menominee", "America/Merida", "America/Metlakatla", "America/Mexico_City",
		"America/Miquelon", "America/Moncton", "America/Monterrey", "America/Montevideo",
		"America/Montserrat", "America/Nassau", "America/New_York", "America/Nome",
		"America/Noronha", "America/North_Dakota/Beulah", "America/North_Dakota/Center",
		"America/North_Dakota/New_Salem", "America/Nuuk", "America/Ojinaga", "America/Panama",
		"America/Paramaribo", "America/Phoenix", "America/Port-au-Prince", "America/Port_of_Spain",
		"America/Porto_Velho", "America/Puerto_Rico", "America/Punta_Arenas",
		"America/Rankin_Inlet", "America/Recife", "America/Regina", "America/Resolute",
		"America/Rio_Branco", "America/Santarem", "America/Santiago", "America/Santo_Domingo",
		"America/Sao_Paulo", "America/Scoresbysund", "America/Sitka", "America/St_Barthelemy",
		"America/St_Johns", "America/St_Kitts", "America/St_Lucia", "America/St_Thomas",
		"America/St_Vincent", "America/Swift_Current", "America/Tegucigalpa", "America/Thule",
		"America/Tijuana", "America/Toronto", "America/Tortola", "America/Vancouver",
		"America/Whitehorse", "America/Winnipeg", "America/Yakutat", "America/Yellowknife",

		// Antarctica
		"Antarctica/Casey", "Antarctica/Davis", "Antarctica/DumontDUrville",
		"Antarctica/Macquarie", "Antarctica/Mawson", "Antarctica/McMurdo",
		"Antarctica/Palmer", "Antarctica/Rothera", "Antarctica/Syowa",
		"Antarctica/Troll", "Antarctica/Vostok",

		// Arctic
		"Arctic/Longyearbyen",

		// Asia
		"Asia/Aden", "Asia/Almaty", "Asia/Amman", "Asia/Anadyr", "Asia/Aqtau",
		"Asia/Aqtobe", "Asia/Ashgabat", "Asia/Atyrau", "Asia/Baghdad", "Asia/Bahrain",
		"Asia/Baku", "Asia/Bangkok", "Asia/Barnaul", "Asia/Beirut", "Asia/Bishkek",
		"Asia/Brunei", "Asia/Chita", "Asia/Choibalsan", "Asia/Colombo", "Asia/Damascus",
		"Asia/Dhaka", "Asia/Dili", "Asia/Dubai", "Asia/Dushanbe", "Asia/Famagusta",
		"Asia/Gaza", "Asia/Hebron", "Asia/Ho_Chi_Minh", "Asia/Hong_Kong", "Asia/Hovd",
		"Asia/Irkutsk", "Asia/Jakarta", "Asia/Jayapura", "Asia/Jerusalem", "Asia/Kabul",
		"Asia/Kamchatka", "Asia/Karachi", "Asia/Kathmandu", "Asia/Khandyga", "Asia/Kolkata",
		"Asia/Krasnoyarsk", "Asia/Kuala_Lumpur", "Asia/Kuching", "Asia/Kuwait", "Asia/Macau",
		"Asia/Magadan", "Asia/Makassar", "Asia/Manila", "Asia/Muscat", "Asia/Nicosia",
		"Asia/Novokuznetsk", "Asia/Novosibirsk", "Asia/Omsk", "Asia/Oral", "Asia/Phnom_Penh",
		"Asia/Pontianak", "Asia/Pyongyang", "Asia/Qatar", "Asia/Qostanay", "Asia/Qyzylorda",
		"Asia/Riyadh", "Asia/Sakhalin", "Asia/Samarkand", "Asia/Seoul", "Asia/Shanghai",
		"Asia/Singapore", "Asia/Srednekolymsk", "Asia/Taipei", "Asia/Tashkent", "Asia/Tbilisi",
		"Asia/Tehran", "Asia/Thimphu", "Asia/Tokyo", "Asia/Tomsk", "Asia/Ulaanbaatar",
		"Asia/Urumqi", "Asia/Ust-Nera", "Asia/Vientiane", "Asia/Vladivostok", "Asia/Yakutsk",
		"Asia/Yangon", "Asia/Yekaterinburg", "Asia/Yerevan",

		// Atlantic
		"Atlantic/Azores", "Atlantic/Bermuda", "Atlantic/Canary", "Atlantic/Cape_Verde",
		"Atlantic/Faroe", "Atlantic/Madeira", "Atlantic/Reykjavik", "Atlantic/South_Georgia",
		"Atlantic/St_Helena", "Atlantic/Stanley",

		// Australia
		"Australia/Adelaide", "Australia/Brisbane", "Australia/Broken_Hill",
		"Australia/Darwin", "Australia/Eucla", "Australia/Hobart", "Australia/Lindeman",
		"Australia/Lord_Howe", "Australia/Melbourne", "Australia/Perth", "Australia/Sydney",

		// Europe
		"Europe/Amsterdam", "Europe/Andorra", "Europe/Astrakhan", "Europe/Athens",
		"Europe/Belgrade", "Europe/Berlin", "Europe/Bratislava", "Europe/Brussels",
		"Europe/Bucharest", "Europe/Budapest", "Europe/Busingen", "Europe/Chisinau",
		"Europe/Copenhagen", "Europe/Dublin", "Europe/Gibraltar", "Europe/Guernsey",
		"Europe/Helsinki", "Europe/Isle_of_Man", "Europe/Istanbul", "Europe/Jersey",
		"Europe/Kaliningrad", "Europe/Kirov", "Europe/Kyiv", "Europe/Lisbon",
		"Europe/Ljubljana", "Europe/London", "Europe/Luxembourg", "Europe/Madrid",
		"Europe/Malta", "Europe/Mariehamn", "Europe/Minsk", "Europe/Monaco",
		"Europe/Moscow", "Europe/Nicosia", "Europe/Oslo", "Europe/Paris",
		"Europe/Podgorica", "Europe/Prague", "Europe/Riga", "Europe/Rome",
		"Europe/Samara", "Europe/San_Marino", "Europe/Sarajevo", "Europe/Saratov",
		"Europe/Simferopol", "Europe/Skopje", "Europe/Sofia", "Europe/Stockholm",
		"Europe/Tallinn", "Europe/Tirane", "Europe/Ulyanovsk", "Europe/Vaduz",
		"Europe/Vatican", "Europe/Vienna", "Europe/Vilnius", "Europe/Volgograd",
		"Europe/Warsaw", "Europe/Zagreb", "Europe/Zurich",

		// Indian
		"Indian/Antananarivo", "Indian/Chagos", "Indian/Christmas", "Indian/Cocos",
		"Indian/Comoro", "Indian/Kerguelen", "Indian/Mahe", "Indian/Maldives",
		"Indian/Mauritius", "Indian/Mayotte", "Indian/Reunion",

		// Pacific
		"Pacific/Apia", "Pacific/Auckland", "Pacific/Bougainville", "Pacific/Chatham",
		"Pacific/Chuuk", "Pacific/Easter", "Pacific/Efate", "Pacific/Fakaofo",
		"Pacific/Fiji", "Pacific/Funafuti", "Pacific/Galapagos", "Pacific/Gambier",
		"Pacific/Guadalcanal", "Pacific/Guam", "Pacific/Honolulu", "Pacific/Kanton",
		"Pacific/Kiritimati", "Pacific/Kosrae", "Pacific/Kwajalein", "Pacific/Majuro",
		"Pacific/Marquesas", "Pacific/Nauru", "Pacific/Niue", "Pacific/Norfolk",
		"Pacific/Noumea", "Pacific/Pago_Pago", "Pacific/Palau", "Pacific/Pitcairn",
		"Pacific/Pohnpei", "Pacific/Port_Moresby", "Pacific/Rarotonga", "Pacific/Saipan",
		"Pacific/Tahiti", "Pacific/Tarawa", "Pacific/Tongatapu", "Pacific/Wake",
		"Pacific/Wallis",

		// UTC
		"UTC",
	}
	slices.Sort(raw)
	return raw
}()
