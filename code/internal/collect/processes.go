// Package collect 采集本机 Claude Code 会话数据,产出快照。
package collect

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// Proc 一个运行中的 Claude Code 会话进程。
type Proc struct {
	PID            int
	RSS            int64 // 字节
	StartedAt      time.Time
	ProjectDir     string // 进程 cwd;取不到为 ""
	SessionID      string // cmdline 中 --resume 的 id;没有为 ""
	Cmdline        []string
	LatestActivity string // 关联项目最新主会话的动作摘要,由 Collect 回填
}

// ListProcs 枚举运行中的 claude 进程,按内存降序。降级说明记入 errs。
func ListProcs(selfPID int) (procs []Proc, errs []string) {
	pids, err := process.Pids()
	if err != nil {
		return nil, []string{"进程枚举失败: " + err.Error()}
	}
	for _, pid := range pids {
		if int(pid) == selfPID {
			continue
		}
		p, err := process.NewProcess(pid)
		if err != nil {
			continue // 进程已退出,正常竞态
		}
		argv, err := p.CmdlineSlice()
		if err != nil || len(argv) == 0 {
			continue
		}
		exe, _ := p.Exe()
		if isClaudeAppHelper(exe, argv) {
			continue
		}
		if !cmdlineMatchesClaude(exe, argv) {
			continue
		}
		pr := Proc{PID: int(pid), Cmdline: argv, SessionID: sessionIDFromArgv(argv)}
		if mi, err := p.MemoryInfo(); err == nil && mi != nil {
			pr.RSS = int64(mi.RSS)
		}
		if ct, err := p.CreateTime(); err == nil {
			pr.StartedAt = time.UnixMilli(ct)
		}
		if cwd, err := p.Cwd(); err == nil {
			pr.ProjectDir = cwd
		}
		procs = append(procs, pr)
	}
	return procs, errs
}

// cmdlineMatchesClaude 判断是否为 claude 可执行:
// exe 或 argv[0] 的文件名等于 claude / claude.exe。
func cmdlineMatchesClaude(exe string, argv []string) bool {
	for _, s := range []string{exe, argv[0]} {
		if s == "" {
			continue
		}
		// 归一化 Windows 反斜杠,使 filepath.Base 在三平台行为一致。
		switch filepath.Base(strings.ReplaceAll(s, "\\", "/")) {
		case "claude", "claude.exe":
			return true
		}
	}
	return false
}

// isClaudeAppHelper 排除 macOS Claude.app 的助手进程(路径含 Claude.app)。
func isClaudeAppHelper(exe string, argv []string) bool {
	if strings.Contains(exe, "Claude.app") {
		return true
	}
	for _, a := range argv {
		if strings.Contains(a, "Claude.app") {
			return true
		}
	}
	return false
}

// sessionIDFromArgv 取 --resume 的参数;没有返回 ""。
func sessionIDFromArgv(argv []string) string {
	for i := 0; i+1 < len(argv); i++ {
		if argv[i] == "--resume" {
			return argv[i+1]
		}
	}
	return ""
}
