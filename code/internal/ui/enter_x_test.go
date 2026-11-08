package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"cstop/internal/collect"
	"cstop/internal/config"
)

// 验证 "\r" 输入被解析为 enter 并打开详情页。
func TestCarriageReturnOpensDetail(t *testing.T) {
	m := New(config.Default(), "/tmp/definitely-not-exist", 1, "0.9.9")
	m.width, m.height = 120, 30
	m.procs = []collect.Proc{{PID: 42, ProjectDir: "/nowhere"}}
	p := tea.NewProgram(m, tea.WithInput(strings.NewReader("\rq")), tea.WithoutRenderer())
	final, err := p.Run()
	if err != nil {
		t.Fatalf("Run() 错误: %v", err)
	}
	fm, ok := final.(Model)
	if !ok {
		t.Fatalf("最终模型类型错误: %T", final)
	}
	if !fm.detail {
		t.Fatal("\\r 未打开详情页;detail = false")
	}
}
