//go:build !linux

package doctor

import "errors"

func diskUsage(_ string) (free, total uint64, err error) {
	return 0, 0, errors.New("disk usage is only implemented on Linux")
}
