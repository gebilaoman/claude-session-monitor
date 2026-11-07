package collect

import "testing"

func TestCmdlineMatchesClaude(t *testing.T) {
	cases := []struct {
		exe  string
		argv []string
		want bool
	}{
		{"", []string{"claude"}, true},
		{"", []string{"claude", "--resume", "abc"}, true},
		{"/usr/local/bin/claude", []string{"/usr/local/bin/claude"}, true},
		{"C:\\tools\\claude.exe", []string{"C:\\tools\\claude.exe"}, true},
		{"/usr/local/bin/node", []string{"node", "cli.js"}, false},
		{"/usr/bin/vim", []string{"vim", "claude.md"}, false},
		{"/usr/bin/man", []string{"man", "claude"}, false},
	}
	for _, c := range cases {
		if got := cmdlineMatchesClaude(c.exe, c.argv); got != c.want {
			t.Errorf("cmdlineMatchesClaude(%q, %v) = %v, want %v", c.exe, c.argv, got, c.want)
		}
	}
}

func TestIsClaudeAppHelper(t *testing.T) {
	helper := "/Applications/Claude.app/Contents/Helpers/chrome-native-host"
	if !isClaudeAppHelper(helper, []string{helper}) {
		t.Error("Claude.app 路径应被排除")
	}
	if isClaudeAppHelper("/usr/local/bin/claude", []string{"claude"}) {
		t.Error("普通 claude 进程不应被排除")
	}
}

func TestSessionIDFromArgv(t *testing.T) {
	if got := sessionIDFromArgv([]string{"claude", "--resume", "abc-123"}); got != "abc-123" {
		t.Errorf("got %q, want abc-123", got)
	}
	if got := sessionIDFromArgv([]string{"claude"}); got != "" {
		t.Errorf("无 --resume 应返回空串,got %q", got)
	}
	if got := sessionIDFromArgv([]string{"claude", "--resume"}); got != "" {
		t.Errorf("--resume 缺参数应返回空串,got %q", got)
	}
}
