package collect

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeFile 写文件并设定 mtime,便于断言 LastActive。
func writeFile(path string, size int, mtime time.Time) error {
	data := make([]byte, size)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	return os.Chtimes(path, mtime, mtime)
}

func TestScanProjects(t *testing.T) {
	root := t.TempDir()
	t1 := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 11, 7, 11, 24, 0, 0, time.UTC)
	t3 := time.Date(2026, 11, 8, 6, 0, 0, 0, time.UTC)

	// projA: 2 个主会话 + subagents + tool-results + index
	a := filepath.Join(root, "-Users-x-work-a")
	os.MkdirAll(filepath.Join(a, "uuid1", "subagents"), 0o755)
	os.MkdirAll(filepath.Join(a, "uuid1", "tool-results"), 0o755)
	writeFile(filepath.Join(a, "uuid1.jsonl"), 100, t1)
	writeFile(filepath.Join(a, "uuid2.jsonl"), 200, t2)
	writeFile(filepath.Join(a, "uuid1", "subagents", "s1.jsonl"), 50, t3)
	writeFile(filepath.Join(a, "uuid1", "tool-results", "r1.bin"), 30, t1)
	writeFile(filepath.Join(a, "sessions-index.json"), 10, t1)

	// projB: 仅 index(空项目)
	b := filepath.Join(root, "-Users-x-work-b")
	os.MkdirAll(b, 0o755)
	writeFile(filepath.Join(b, "sessions-index.json"), 5, t1)

	projects, total, errs := ScanProjects(root)
	if len(errs) != 0 {
		t.Fatalf("不应有降级说明: %v", errs)
	}
	if len(projects) != 2 {
		t.Fatalf("应有 2 个项目,got %d", len(projects))
	}
	var pa, pb ProjectDisk
	for _, p := range projects {
		switch p.Name {
		case "-Users-x-work-a":
			pa = p
		case "-Users-x-work-b":
			pb = p
		}
	}
	if pa.Sessions != 2 {
		t.Errorf("projA 主会话数应 2(不含 subagents),got %d", pa.Sessions)
	}
	if want := int64(100 + 200 + 50 + 30 + 10); pa.Bytes != want {
		t.Errorf("projA 字节 = %d, want %d", pa.Bytes, want)
	}
	if !pa.LastActive.Equal(t3) {
		t.Errorf("projA LastActive 应为全部文件最大 mtime %v,got %v", t3, pa.LastActive)
	}
	if pa.LatestMain != "uuid2.jsonl" {
		t.Errorf("projA LatestMain = %q, want uuid2.jsonl", pa.LatestMain)
	}
	if pb.Sessions != 0 || pb.Bytes != 5 {
		t.Errorf("projB 空项目统计错误: %+v", pb)
	}
	if want := pa.Bytes + pb.Bytes; total != want {
		t.Errorf("total = %d, want %d", total, want)
	}
}

func TestScanProjectsMissingRoot(t *testing.T) {
	_, _, errs := ScanProjects(filepath.Join(t.TempDir(), "nope"))
	if len(errs) != 1 {
		t.Fatalf("根目录缺失应有 1 条降级说明,got %v", errs)
	}
}

func TestSlugFromPath(t *testing.T) {
	cases := map[string]string{
		"/Users/x/work/foo":       "-Users-x-work-foo",
		"/Users/x/work/foo/":      "-Users-x-work-foo",
		"C:\\Users\\x\\work\\foo": "C:-Users-x-work-foo",
	}
	for in, want := range cases {
		if got := SlugFromPath(in); got != want {
			t.Errorf("SlugFromPath(%q) = %q, want %q", in, got, want)
		}
	}
}
