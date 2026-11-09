// Package osrelease reads /etc/os-release, the standard file where Linux
// distributions describe themselves (ID, VERSION_ID, VERSION_CODENAME...).
package osrelease

import "strings"

// Path is where the file lives on every supported system.
const Path = "/etc/os-release"

// Parse reads the KEY=value lines of an os-release file. Quotes around values
// are removed and comment lines are ignored.
func Parse(data string) map[string]string {
	values := map[string]string{}
	for _, line := range strings.Split(data, "\n") {
		key, value, found := strings.Cut(strings.TrimSpace(line), "=")
		if !found || strings.HasPrefix(key, "#") {
			continue
		}
		values[key] = strings.Trim(value, `"'`)
	}
	return values
}
