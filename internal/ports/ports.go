// Package ports finds which local ports are in use by reading /proc/net.
// Reading /proc needs no root, unlike `ss -p`, and its format does not depend
// on the system language.
package ports

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var sources = []struct {
	path     string
	protocol string
}{
	{"/proc/net/tcp", "tcp"},
	{"/proc/net/tcp6", "tcp"},
	{"/proc/net/udp", "udp"},
	{"/proc/net/udp6", "udp"},
}

// InUse returns the ports that are listening (TCP) or bound (UDP), as keys
// like "53/udp". readFile is os.ReadFile in the real program; tests pass a
// fake. It fails only when none of the /proc/net files can be read.
func InUse(readFile func(path string) ([]byte, error)) (map[string]bool, error) {
	inUse := map[string]bool{}
	readAny := false
	for _, src := range sources {
		data, err := readFile(src.path)
		if err != nil {
			continue
		}
		readAny = true
		for port := range parse(string(data), src.protocol == "udp") {
			inUse[Key(port, src.protocol)] = true
		}
	}
	if !readAny {
		return nil, errors.New("could not read /proc/net to check ports")
	}
	return inUse, nil
}

// Key formats a port as used in InUse, e.g. Key(53, "udp") is "53/udp".
func Key(port int, protocol string) string {
	return fmt.Sprintf("%d/%s", port, protocol)
}

// parse returns the local ports that are listening (TCP, state 0A) or bound
// (UDP, state 07) in a /proc/net/{tcp,udp}[6] file.
func parse(data string, udp bool) map[int]bool {
	wantState := "0A"
	if udp {
		wantState = "07"
	}
	found := map[int]bool{}
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 || fields[3] != wantState {
			continue
		}
		_, hexPort, ok := strings.Cut(fields[1], ":")
		if !ok {
			continue
		}
		port, err := strconv.ParseUint(hexPort, 16, 16)
		if err != nil {
			continue
		}
		found[int(port)] = true
	}
	return found
}
