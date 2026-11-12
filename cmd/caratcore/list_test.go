package main

import (
	"strings"
	"testing"

	"github.com/BryanAguilar-Dev/CaratCore/internal/service"
)

func TestPrintList(t *testing.T) {
	var b strings.Builder
	printList(&b, []listRow{
		{"adguard-home", service.Status{Name: "not installed"}, nil},
		{"uptime-kuma", service.Status{Name: "update available", Detail: "a:1 -> a:2"}, []string{"/opt/caratcore/uptime-kuma.bak-2026-11-12-153000"}},
	})
	got := b.String()
	for _, want := range []string{
		"SERVICE       STATUS            DETAILS\n",
		"uptime-kuma   update available  a:1 -> a:2\n",
		"  /opt/caratcore/uptime-kuma.bak-2026-11-12-153000\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}
