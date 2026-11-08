package collect

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ProjectDisk 磁盘上一项占用: 会话项目、~/.claude 其他一级目录或顶层文件。
type ProjectDisk struct {
	Name       string    // 目录名(项目 slug 或一级目录/文件名)
	Kind       string    // project 会话项目 / dir 其他一级目录 / file 顶层文件
	Sessions   int       // 主会话 jsonl 数(不含 subagents);仅 project 有意义
	Bytes      int64     // 全部文件字节
	LastActive time.Time // 全部文件的最近 mtime
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
		pd.Name, pd.Kind = e.Name(), "project"
		projects = append(projects, pd)
		total += pd.Bytes
	}
	return projects, total, errs
}

// ScanAll 统计 Claude 全目录占用: root(projects)下按项目,
// 其余一级目录与文件(plugins、file-history 等)各成一行。合计为 ~/.claude 总量。
func ScanAll(root string) (items []ProjectDisk, total int64, errs []string) {
	items, total, errs = ScanProjects(root)
	base := filepath.Dir(root)
	entries, err := os.ReadDir(base)
	if err != nil {
		return items, total, append(errs, "claude 目录不可读: "+err.Error())
	}
	skip := filepath.Base(root)
	for _, e := range entries {
		name := e.Name()
		if name == skip || strings.HasPrefix(name, ".") {
			continue
		}
		info, ierr := e.Info()
		if ierr != nil {
			continue
		}
		if e.IsDir() {
			pd, derr := scanProject(filepath.Join(base, name))
			if derr != nil {
				errs = append(errs, "目录扫描失败 "+name+": "+derr.Error())
				continue
			}
			pd.Name, pd.Kind, pd.Sessions, pd.LatestMain = name, "dir", 0, ""
			items = append(items, pd)
			total += pd.Bytes
		} else {
			items = append(items, ProjectDisk{
				Name: name, Kind: "file",
				Bytes: diskUsage(info), LastActive: info.ModTime(),
			})
			total += diskUsage(info)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Bytes > items[j].Bytes })
	return items, total, errs
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
		info, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		pd.Bytes += diskUsage(info) // 文件与目录均按磁盘块计,与 du 口径一致
		if !d.IsDir() && info.ModTime().After(pd.LastActive) {
			pd.LastActive = info.ModTime()
		}
		return nil
	})
	return pd, err
}
