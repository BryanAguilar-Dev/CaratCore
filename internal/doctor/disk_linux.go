//go:build linux

package doctor

import "syscall"

func diskUsage(path string) (free, total uint64, err error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0, err
	}
	blockSize := uint64(st.Bsize)
	return uint64(st.Bavail) * blockSize, uint64(st.Blocks) * blockSize, nil
}
