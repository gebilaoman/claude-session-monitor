package collect

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// localHM 把 ISO 时间串转为本地时区的 HH:MM:SS(测试期望值与时区解耦)。
func localHM(ts string) string {
	t, _ := time.Parse(time.RFC3339Nano, ts)
	return t.Local().Format("15:04:05")
}

func TestSummarizeLine(t *testing.T) {
	cases := []struct {
		name string
		line string
		want string
		ok   bool
	}{
		{
			"assistant 工具调用",
			`{"type":"assistant","timestamp":"2026-11-08T12:03:41.000Z","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{}}]}}`,
			"Bash · " + localHM("2026-11-08T12:03:41.000Z"), true,
		},
		{
			"assistant 文本回复",
			`{"type":"assistant","timestamp":"2026-11-08T12:05:02Z","message":{"role":"assistant","content":[{"type":"text","text":"hi"}]}}`,
			"回复 · " + localHM("2026-11-08T12:05:02Z"), true,
		},
		{
			"assistant thinking+tool_use 取工具名",
			`{"type":"assistant","timestamp":"2026-11-08T12:06:00Z","message":{"role":"assistant","content":[{"type":"thinking","thinking":"..."},{"type":"tool_use","name":"Read","input":{}}]}}`,
			"Read · " + localHM("2026-11-08T12:06:00Z"), true,
		},
		{
			"assistant 仅 thinking 回溯",
			`{"type":"assistant","timestamp":"2026-11-08T12:07:00Z","message":{"role":"assistant","content":[{"type":"thinking","thinking":"..."}]}}`,
			"", false,
		},
		{
			"用户消息",
			`{"type":"user","timestamp":"2026-11-08T12:08:00Z","message":{"role":"user","content":[{"type":"text","text":"q"}]}}`,
			"用户 · " + localHM("2026-11-08T12:08:00Z"), true,
		},
		{
			"元数据行不产出",
			`{"type":"api-request","timestamp":"2026-11-08T12:09:00Z","request":{}}`,
			"", false,
		},
		{
			"坏 JSON 不产出",
			`{"type":"assistant","broken`,
			"", false,
		},
		{
			"无时间戳只给动作",
			`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"q"}]}}`,
			"用户", true,
		},
	}
	for _, c := range cases {
		got, ok := summarizeLine([]byte(c.line))
		if ok != c.ok || got != c.want {
			t.Errorf("%s: got (%q,%v), want (%q,%v)", c.name, got, ok, c.want, c.ok)
		}
	}
}

func TestTruncateRunes(t *testing.T) {
	if got := truncateRunes("一二三四五六七八九十一二三4567890123456789012345678901234567890", 40); len([]rune(got)) != 40 {
		t.Errorf("截断后应为 40 runes,got %d", len([]rune(got)))
	}
	if got := truncateRunes("short", 40); got != "short" {
		t.Errorf("短串不应截断,got %q", got)
	}
}

func TestActivityOfProject(t *testing.T) {
	dir := t.TempDir()
	t1 := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 11, 8, 6, 0, 0, 0, time.UTC)

	old := `{"type":"assistant","timestamp":"2026-11-08T11:00:00Z","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{}}]}}`
	// 最新主会话: 末行是坏行与元数据行,应回溯到 assistant 文本行。
	newest := "\n" +
		`{"type":"assistant","timestamp":"2026-11-08T12:01:00Z","message":{"role":"assistant","content":[{"type":"text","text":"hi"}]}}` + "\n" +
		`{"type":"api-request","timestamp":"2026-11-08T12:02:00Z"}` + "\n" +
		`{broken json` + "\n"
	os.WriteFile(filepath.Join(dir, "old.jsonl"), []byte(old), 0o644)
	os.Chtimes(filepath.Join(dir, "old.jsonl"), t1, t1)
	os.WriteFile(filepath.Join(dir, "new.jsonl"), []byte(newest), 0o644)
	os.Chtimes(filepath.Join(dir, "new.jsonl"), t2, t2)

	want := "回复 · " + localHM("2026-11-08T12:01:00Z")
	if got := ActivityOfProject(dir); got != want {
		t.Fatalf("ActivityOfProject = %q, want %q", got, want)
	}
}

func TestActivityOfProjectEmpty(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "sessions-index.json"), []byte("{}"), 0o644)
	if got := ActivityOfProject(dir); got != "" {
		t.Fatalf("无主会话应返回空串,got %q", got)
	}
	if got := ActivityOfProject(filepath.Join(dir, "nope")); got != "" {
		t.Fatalf("目录不存在应返回空串,got %q", got)
	}
}
