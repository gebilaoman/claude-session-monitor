package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(path, body string) error {
	return os.WriteFile(path, []byte(body), 0o644)
}

func TestDefault(t *testing.T) {
	d := Default()
	want := Config{2, 60, 400, 1500, 500, 2000, 6, true}
	if d != want {
		t.Fatalf("Default() = %+v, want %+v", d, want)
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg, warnings := Load(filepath.Join(t.TempDir(), "no.toml"))
	if cfg != Default() {
		t.Fatalf("缺失文件应用默认值,got %+v", cfg)
	}
	if len(warnings) != 0 {
		t.Fatalf("缺失文件不应有告警,got %v", warnings)
	}
}

func TestLoadFull(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	body := `refresh_procs_sec = 5
refresh_disk_sec = 120
warn_session_rss_mb = 300
warn_total_rss_mb = 1000
warn_disk_project_mb = 400
warn_disk_total_mb = 1500
warn_session_count = 3
update_check_on_start = false
`
	if err := writeFile(path, body); err != nil {
		t.Fatal(err)
	}
	cfg, warnings := Load(path)
	want := Config{5, 120, 300, 1000, 400, 1500, 3, false}
	if cfg != want {
		t.Fatalf("Load() = %+v, want %+v", cfg, want)
	}
	if len(warnings) != 0 {
		t.Fatalf("合法配置不应有告警,got %v", warnings)
	}
}

func TestLoadInlineCommentAndBlank(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	body := `# 注释行

refresh_procs_sec = 7 # 行尾注释
update_check_on_start = true
`
	if err := writeFile(path, body); err != nil {
		t.Fatal(err)
	}
	cfg, warnings := Load(path)
	if cfg.RefreshProcsSec != 7 || !cfg.UpdateCheckOnStart {
		t.Fatalf("行尾注释未正确剥离,got %+v", cfg)
	}
	if len(warnings) != 0 {
		t.Fatalf("不应有告警,got %v", warnings)
	}
}

func TestLoadBadLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	body := `没有等号的行
refresh_procs_sec = abc
unknown_key = 1
warn_session_count = -2
update_check_on_start = 不是布尔
`
	if err := writeFile(path, body); err != nil {
		t.Fatal(err)
	}
	cfg, warnings := Load(path)
	d := Default()
	if cfg != d {
		t.Fatalf("坏行对应键应用默认值,got %+v", cfg)
	}
	if len(warnings) != 5 {
		t.Fatalf("应有 5 条告警,got %d: %v", len(warnings), warnings)
	}
}

func TestWriteDefaultRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.toml")
	if err := WriteDefault(path); err != nil {
		t.Fatal(err)
	}
	cfg, warnings := Load(path)
	if cfg != Default() {
		t.Fatalf("生成的配置应与默认值一致,got %+v", cfg)
	}
	if len(warnings) != 0 {
		t.Fatalf("生成的配置不应有告警,got %v", warnings)
	}
	if err := WriteDefault(path); err == nil {
		t.Fatal("重复生成应报错")
	}
}
