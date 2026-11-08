//go:build windows

package collect

import "io/fs"

// diskUsage Windows 无通用块占用语义,用文件大小(NTFS 压缩/稀疏文件会偏大)。
func diskUsage(info fs.FileInfo) int64 { return info.Size() }
