//go:build !windows

package collect

import (
	"io/fs"
	"syscall"
)

// diskUsage 返回文件占用的磁盘块字节(与 du 一致,含目录 inode);取不到时回退文件大小。
func diskUsage(info fs.FileInfo) int64 {
	if st, ok := info.Sys().(*syscall.Stat_t); ok && st.Blocks >= 0 {
		return st.Blocks * 512
	}
	return info.Size()
}
