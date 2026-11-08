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

// Collect 一次性采集进程与磁盘(--dump 用)。
func Collect(selfPID int, root string) Snapshot {
	procs, errs := ProcsTick(selfPID, root)
	projects, total, derrs := DiskTick(root)
	return Snapshot{
		Procs:     procs,
		Projects:  projects,
		TotalDisk: total,
		Errors:    append(errs, derrs...),
	}
}

// DiskTick 扫描 Claude 全目录占用(~/.claude,含 projects 与其他一级目录)。
// 低频: 间隔 refresh_disk_sec,结果缓存到下一次。
func DiskTick(root string) ([]ProjectDisk, int64, []string) {
	return ScanAll(root)
}

// ProcsTick 采集运行中进程与活动摘要。高频: 间隔 refresh_procs_sec。
// 同项目多进程显示同一条活动摘要(项目级最新)。
func ProcsTick(selfPID int, root string) ([]Proc, []string) {
	procs, errs := ListProcs(selfPID)
	activity := make(map[string]string) // 进程 cwd -> 摘要
	for i := range procs {
		dir := procs[i].ProjectDir
		if dir == "" {
			continue
		}
		if _, done := activity[dir]; !done {
			activity[dir] = ActivityOfProject(filepath.Join(root, SlugFromPath(dir)))
		}
		procs[i].LatestActivity = activity[dir]
	}
	sort.Slice(procs, func(i, j int) bool { return procs[i].RSS > procs[j].RSS })
	return procs, errs
}

// SlugFromPath 把项目路径转为 projects 下的目录名: / 与 \ 都替换为 -。
func SlugFromPath(path string) string {
	path = filepath.Clean(path)
	path = strings.ReplaceAll(path, "/", "-")
	return strings.ReplaceAll(path, "\\", "-")
}
