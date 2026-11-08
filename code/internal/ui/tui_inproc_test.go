package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"cstop/internal/config"
)

// 进程内验证事件循环: q 键应让程序正常退出。
func TestProgramQuitsOnQ(t *testing.T) {
	m := New(config.Default(), "/tmp/definitely-not-exist", 1, "0.9.9")
	p := tea.NewProgram(m,
		tea.WithInput(strings.NewReader("q")),
		tea.WithoutRenderer(),
	)
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() 错误: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("q 键未使程序退出(5 秒超时)")
	}
}
