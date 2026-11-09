package doctor

import (
	"os"
	"testing"
)

func fakeEnv(files map[string]string) Env {
	return Env{
		GOOS:   "linux",
		GOARCH: "amd64",
		CPUs:   4,
		Euid:   1000,
		ReadFile: func(path string) ([]byte, error) {
			if content, ok := files[path]; ok {
				return []byte(content), nil
			}
			return nil, os.ErrNotExist
		},
	}
}

func TestParseMeminfo(t *testing.T) {
	total, available, err := parseMeminfo("MemTotal:        5880000 kB\nMemFree:  100 kB\nMemAvailable:    5000000 kB\n")
	if err != nil {
		t.Fatal(err)
	}
	if total != 5880000*1024 || available != 5000000*1024 {
		t.Fatalf("got total=%d available=%d", total, available)
	}
	if _, _, err := parseMeminfo("MemTotal: 10 kB\n"); err == nil {
		t.Fatal("expected an error when MemAvailable is missing")
	}
}

func TestParseDockerVersion(t *testing.T) {
	if got := parseDockerVersion("Docker version 29.9.1, build dfc520d\n"); got != "29.9.1" {
		t.Fatalf("got %q", got)
	}
	if got := parseDockerVersion("nonsense"); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestParseTailscaleState(t *testing.T) {
	state, err := parseTailscaleState("warning on stderr\n{\"BackendState\": \"Running\", \"Other\": 1}\n")
	if err != nil || state != "Running" {
		t.Fatalf("got %q, %v", state, err)
	}
	if _, err := parseTailscaleState("not json"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCheckPlatform(t *testing.T) {
	tests := []struct {
		name  string
		goos  string
		osrel string
		want  Status
	}{
		{"debian 13", "linux", "ID=debian\nVERSION_ID=\"13\"\n", StatusOK},
		{"debian 12", "linux", "ID=debian\nVERSION_ID=\"12\"\n", StatusWarn},
		{"windows", "windows", "", StatusFail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := fakeEnv(map[string]string{"/etc/os-release": tt.osrel})
			env.GOOS = tt.goos
			if got := checkPlatform(env).Status; got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckMemoryThresholds(t *testing.T) {
	tests := []struct {
		name         string
		availableKiB string
		want         Status
	}{
		{"plenty", "4000000", StatusOK},
		{"low", "800000", StatusWarn},
		{"critical", "300000", StatusFail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := fakeEnv(map[string]string{
				"/proc/meminfo": "MemTotal: 5000000 kB\nMemAvailable: " + tt.availableKiB + " kB\n",
			})
			if got := checkMemory(env).Status; got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckPorts(t *testing.T) {
	free := fakeEnv(map[string]string{"/proc/net/tcp": "  sl  local_address rem_address   st\n"})
	if got := checkPorts(free).Status; got != StatusOK {
		t.Fatalf("free ports: got %v", got)
	}
	busy := fakeEnv(map[string]string{
		"/proc/net/udp": "  sl  local_address rem_address   st\n   0: 0100007F:0035 00000000:0000 07 00000000:00000000\n",
	})
	if got := checkPorts(busy).Status; got != StatusWarn {
		t.Fatalf("busy port 53: got %v", got)
	}
	if got := checkPorts(fakeEnv(nil)).Status; got != StatusWarn {
		t.Fatalf("no /proc/net: got %v", got)
	}
}

func TestReportVerdict(t *testing.T) {
	report := Report{Checks: []Check{{Status: StatusOK}, {Status: StatusInfo}}}
	if report.Worst() != StatusInfo || report.Verdict() != "ready" {
		t.Fatalf("unexpected: %v %q", report.Worst(), report.Verdict())
	}
	report.Checks = append(report.Checks, Check{Status: StatusFail})
	if report.Worst() != StatusFail {
		t.Fatal("a FAIL must make the report fail")
	}
}
