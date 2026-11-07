package collect

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const activityTailBytes = 64 * 1024

// ActivityOfProject 返回项目最新主会话的最近动作摘要,例: "Bash · 12:03:41"。
// 无主会话或无可解析对话行时返回 ""。
// 末尾若干行为元数据(api-request 等)时,向前回溯至最近一条对话记录。
func ActivityOfProject(projectDir string) string {
	name, _, err := mainsOf(projectDir)
	if err != nil || name == "" {
		return ""
	}
	f, err := os.Open(filepath.Join(projectDir, name))
	if err != nil {
		return ""
	}
	defer f.Close()
	lines := strings.Split(string(tailBytes(f, activityTailBytes)), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if s, ok := summarizeLine([]byte(line)); ok {
			return s
		}
	}
	return ""
}

// mainsOf 统计项目顶层主会话: 返回最新的 jsonl 文件名与总数。
// 只读顶层目录项,不做递归 walk,适配高频调用。
func mainsOf(projectDir string) (latest string, count int, err error) {
	tops, err := os.ReadDir(projectDir)
	if err != nil {
		return "", 0, err
	}
	var latestTime time.Time
	for _, e := range tops {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		count++
		info, ierr := e.Info()
		if ierr != nil {
			continue
		}
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latest = e.Name()
		}
	}
	return latest, count, nil
}

func tailBytes(f *os.File, n int64) []byte {
	size, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return nil
	}
	if size <= n {
		n = size
	}
	if _, err := f.Seek(size-n, io.SeekStart); err != nil {
		return nil
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(f, buf); err != nil {
		return nil
	}
	return buf
}

// summarizeLine 从一条会话记录生成摘要。
// 仅对话记录(user/assistant)产出摘要;元数据行与仅 thinking 的中间态返回 ok=false。
func summarizeLine(line []byte) (string, bool) {
	var rec struct {
		Type      string `json:"type"`
		Timestamp string `json:"timestamp"`
		Message   *struct {
			Role    string `json:"role"`
			Content []struct {
				Type string `json:"type"`
				Name string `json:"name"`
			} `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(line, &rec); err != nil {
		return "", false
	}
	var act string
	switch rec.Type {
	case "assistant":
		if rec.Message == nil {
			return "", false
		}
		hasText := false
		for _, b := range rec.Message.Content {
			if b.Type == "tool_use" && b.Name != "" {
				act = b.Name
				break
			}
			if b.Type == "text" {
				hasText = true
			}
		}
		if act == "" {
			if !hasText {
				return "", false // 仅 thinking 等中间态,回溯上一条
			}
			act = "回复"
		}
	case "user":
		act = "用户"
	default:
		return "", false
	}
	if ts := parseTimestamp(rec.Timestamp); !ts.IsZero() {
		act += " · " + ts.Local().Format("15:04:05")
	}
	return truncateRunes(act, 40), true
}

func parseTimestamp(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
