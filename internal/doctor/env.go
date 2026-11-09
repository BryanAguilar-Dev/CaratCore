package doctor

import (
	"context"
	"net"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// Env holds everything doctor needs from the outside world (files, commands,
// network). Tests replace these functions with fakes.
type Env struct {
	GOOS, GOARCH string
	CPUs         int
	Euid         int

	Hostname  func() (string, error)
	ReadFile  func(path string) ([]byte, error)
	LookPath  func(file string) (string, error)
	Run       func(ctx context.Context, name string, args ...string) (string, error)
	DiskUsage func(path string) (free, total uint64, err error)
	Dial      func(ctx context.Context, address string) error
}

// DefaultEnv returns an Env wired to the real system.
func DefaultEnv() Env {
	return Env{
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
		CPUs:      runtime.NumCPU(),
		Euid:      os.Geteuid(),
		Hostname:  os.Hostname,
		ReadFile:  os.ReadFile,
		LookPath:  exec.LookPath,
		Run:       runCommand,
		DiskUsage: diskUsage,
		Dial:      dial,
	}
}

// runCommand returns stdout and stderr together, so callers can read error
// messages such as "permission denied".
func runCommand(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(out), err
}

func dial(ctx context.Context, address string) error {
	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	return conn.Close()
}
