# Bundled runtime rules

`zoneinfo.zip` is copied byte-for-byte from Go 1.27.0's
`lib/time/zoneinfo.zip`. Its `lib/time/update.bash` declares IANA tzcode/tzdata
2026c. Source: https://github.com/golang/go/blob/go1.27.0/lib/time/zoneinfo.zip

SHA-256: `b2d18a7c8fa8142097a48c99609fb3c92db5ee98bc740294e57eab8ae9f94779`

The IANA timezone database is public-domain data (see
https://www.iana.org/time-zones). This archive contains TZif rules, not display
locale data. It replaces the opaque `time/tzdata` fallback so the resolver and
stdlib Location can consume the exact same bytes.

For an intentional update, obtain the archive from a reviewed Go release,
verify its checksum and update this provenance alongside the archive. Run
`go test -race ./internal/zone .` and `task verify`; the bundled corpus test
loads every entry and verifies forward/inverse projection. No build-time or
runtime download occurs.

Load order: `ZONEINFO` directory/ZIP, conventional Unix zoneinfo directories,
then this archive. A Go installation is not needed at runtime. Missing host data therefore
does not prevent IANA loading. Platform-specific packed stores (such as Android
`tzdata`) are not read; deployments can provide a directory/ZIP through
`ZONEINFO`. The environment variable is captured on first load. Each loaded
Zone holds its snapshot; later loads can see replaced files.

The generated identifier catalog has independent provenance in `sources.json`.
Its version does not describe the rules selected for any particular Zone.
