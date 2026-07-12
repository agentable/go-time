package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestGenerate_SortedTerritory001AndStable(t *testing.T) {
	t.Parallel()

	mappings, err := readMappings("testdata/windowsZones.xml")
	if err != nil {
		t.Fatalf("readMappings() error = %v", err)
	}
	if len(mappings) != 2 {
		t.Fatalf("len(mappings) = %d, want 2", len(mappings))
	}
	if mappings[0].windows != "Alpha Standard Time" || mappings[0].iana != "America/Godthab" {
		t.Fatalf("mappings[0] = %+v, want sorted Alpha mapping with source link preserved", mappings[0])
	}

	first, err := render("release-test", "windowsZones.xml", mappings)
	if err != nil {
		t.Fatalf("render() error = %v", err)
	}
	second, err := render("release-test", "windowsZones.xml", mappings)
	if err != nil {
		t.Fatalf("second render() error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("render() is not byte-identical across calls")
	}

	got := string(first)
	alpha := strings.Index(got, `"Alpha Standard Time": "America/Godthab"`)
	zulu := strings.Index(got, `"Zulu Standard Time":  "Etc/UTC"`)
	if alpha < 0 || zulu < 0 || alpha >= zulu {
		t.Fatalf("generated mappings are missing or unsorted:\n%s", got)
	}
	if strings.Contains(got, "Ignored Regional Time") {
		t.Fatalf("generated source contains non-001 mapping:\n%s", got)
	}
	if !strings.Contains(got, "Unicode CLDR release-test") || !strings.Contains(got, `territory "001"`) {
		t.Fatalf("generated header lacks pinned release or territory:\n%s", got)
	}
}

func TestReadMappings_RejectsDuplicateWindowsName(t *testing.T) {
	t.Parallel()

	xml := `<supplementalData><windowsZones><mapTimezones>
<mapZone other="Duplicate" territory="001" type="Etc/UTC"/>
<mapZone other="Duplicate" territory="001" type="Europe/London"/>
</mapTimezones></windowsZones></supplementalData>`
	_, err := decodeMappings(strings.NewReader(xml))
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("decodeMappings() error = %v, want duplicate error", err)
	}
}

func TestReadMappings_RejectsMalformedInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		xml  string
		want string
	}{
		{name: "malformed XML", xml: `<supplementalData>`, want: "decode XML"},
		{name: "wrong hierarchy", xml: `<supplementalData><mapZone other="Bad" territory="001" type="Etc/UTC"/></supplementalData>`, want: "no territory 001"},
		{name: "missing Windows name", xml: `<supplementalData><windowsZones><mapTimezones><mapZone territory="001" type="Etc/UTC"/></mapTimezones></windowsZones></supplementalData>`, want: "missing other"},
		{name: "multiple default targets", xml: `<supplementalData><windowsZones><mapTimezones><mapZone other="Bad" territory="001" type="A B"/></mapTimezones></windowsZones></supplementalData>`, want: "exactly one IANA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := decodeMappings(strings.NewReader(tt.xml))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("decodeMappings() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}
