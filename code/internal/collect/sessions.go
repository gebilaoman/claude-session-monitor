package collect

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// ProjectDisk 磁盘上一个项目的会话记录占用。
type ProjectDisk struct {
	Name       string    // 项目目录名(项目路径的 slug)
	Sessions   int       // 主会话 jsonl 数(不含 subagents)
	Bytes      int64     // 项目目录全部文件字节
	LastActive time.Time // 项目内全部文件的最近 mtime
	LatestMain string    // 最新的主会话 jsonl 文件名(活动摘要用)
}

// ScanProjects 统计 root(通常 ~/.claude/projects)下全部项目。
// 顶层 *.jsonl 计主会话数;目录内全部文件(含 subagents、tool-results、memory)计入字节。
func ScanProjects(root string) (projects []ProjectDisk, total int64, errs []string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, 0, []string{"projects 目录不可读: " + err.Error()}
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pd, err := scanProject(filepath.Join(root, e.Name()))
		if err != nil {
			errs = append(errs, "项目扫描失败 "+e.Name()+": "+err.Error())
			continue
		}
		pd.Name = e.Name()
		projects = append(projects, pd)
		total += pd.Bytes
	}
	return projects, total, errs
}

func scanProject(dir string) (pd ProjectDisk, err error) {
	pd.LatestMain, pd.Sessions, err = mainsOf(dir)
	if err != nil {
		return pd, err
	}
	err = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		if d.IsDir() {
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		pd.Bytes += info.Size()
		if info.ModTime().After(pd.LastActive) {
			pd.LastActive = info.ModTime()
		}
		return nil
	})
	return pd, err
}
