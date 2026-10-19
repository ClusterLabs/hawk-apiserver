package internal

import "testing"

func TestParseOSRelease(t *testing.T) {
	for _, test := range []struct {
		name, contents string
		want           OSRelease
	}{
		{"quoted", "NAME=\"SUSE Linux Enterprise Server\"\nID=\"sles\"\nVERSION_ID=\"15.6\"\n", OSRelease{"sles", "15.6"}},
		{"unquoted", "ID=sles\nVERSION_ID=16.0\n", OSRelease{"sles", "16.0"}},
		{"single quotes", "ID='opensuse-leap'\nVERSION_ID='16.0'\n", OSRelease{"opensuse-leap", "16.0"}},
		{"missing", "# No distribution information\nID_LIKE=sles\nVERSION=16\n", OSRelease{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := parseOSRelease(test.contents); got != test.want {
				t.Fatalf("got %+v, want %+v", got, test.want)
			}
		})
	}
}
