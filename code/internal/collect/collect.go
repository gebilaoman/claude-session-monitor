package collect

import (
	"path/filepath"
	"sort"
	"strings"
)

// Snapshot 一份完整的机器快照。
type Snapshot struct {
	Procs     []Proc
	Projects  []ProjectDisk
	TotalDisk int64
	Errors    []string // 采集中的降级说明
}

// Collect 采集快照: 进程 + 磁盘 + 活动关联。
// selfPID 用于排除本工具自身;root 为 projects 目录。
func Collect(selfPID int, root string) Snapshot {
	var snap Snapshot
	snap.Procs, snap.Errors = ListProcs(selfPID)
	projects, total, errs := ScanProjects(root)
	snap.Projects, snap.TotalDisk = projects, total
	snap.Errors = append(snap.Errors, errs...)

	// 活跃项目的最近动作: 每个项目只解析一次;同项目多进程显示同一条(项目级最新)。
	known := make(map[string]bool, len(projects))
	for _, p := range projects {
		known[p.Name] = true
	}
	activity := make(map[string]string) // 进程 cwd -> 摘要
	for i := range snap.Procs {
		dir := snap.Procs[i].ProjectDir
		if dir == "" {
			continue
		}
		slug := SlugFromPath(dir)
		if !known[slug] {
			continue
		}
		if _, done := activity[dir]; !done {
			activity[dir] = ActivityOfProject(filepath.Join(root, slug))
		}
		snap.Procs[i].LatestActivity = activity[dir]
	}

	sort.Slice(snap.Procs, func(i, j int) bool { return snap.Procs[i].RSS > snap.Procs[j].RSS })
	sort.Slice(snap.Projects, func(i, j int) bool { return snap.Projects[i].Bytes > snap.Projects[j].Bytes })
	return snap
}

// SlugFromPath 把项目路径转为 projects 下的目录名: / 与 \ 都替换为 -。
func SlugFromPath(path string) string {
	path = filepath.Clean(path)
	path = strings.ReplaceAll(path, "/", "-")
	return strings.ReplaceAll(path, "\\", "-")
}
