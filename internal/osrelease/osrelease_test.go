package osrelease

import "testing"

func TestParse(t *testing.T) {
	got := Parse("PRETTY_NAME=\"Debian GNU/Linux 13 (trixie)\"\nVERSION_ID=\"13\"\nVERSION_CODENAME=trixie\nID=debian\n# comment\n")
	if got["ID"] != "debian" || got["VERSION_ID"] != "13" || got["VERSION_CODENAME"] != "trixie" ||
		got["PRETTY_NAME"] != "Debian GNU/Linux 13 (trixie)" {
		t.Fatalf("unexpected result: %v", got)
	}
}
