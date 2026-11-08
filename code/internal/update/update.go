// Package update 检测并安装 cstop 新版本(更新源: GitHub Releases)。
package update

import (
	"context"
	"fmt"
	"os"

	selfupdate "github.com/creativeprojects/go-selfupdate"
)

const (
	repoOwner = "gebilaoman"
	repoName  = "claude-session-monitor"
)

func newUpdater() (*selfupdate.Updater, error) {
	return selfupdate.NewUpdater(selfupdate.Config{
		// 发布产物附带 checksums.txt;下载后强制校验。
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
}

// Check 查询最新 Release 并与当前版本比较。
// 返回是否有新版与最新版本号。超时与网络错误由调用方处理。
func Check(ctx context.Context, currentVersion string) (available bool, latest string, err error) {
	up, err := newUpdater()
	if err != nil {
		return false, "", err
	}
	release, found, err := up.DetectLatest(ctx, selfupdate.NewRepositorySlug(repoOwner, repoName))
	if err != nil || !found {
		return false, "", err
	}
	return release.GreaterThan(currentVersion), release.Version(), nil
}

// Run 检测 → 下载 → 校验 sha256 → 原子替换自身。已是最新时打印提示并返回 nil。
// 替换失败保留旧二进制。平台细节(含 Windows 运行中 exe 先改名)由库处理。
func Run(ctx context.Context, currentVersion string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("无法定位当前可执行文件: %w", err)
	}
	up, err := newUpdater()
	if err != nil {
		return err
	}
	release, found, err := up.DetectLatest(ctx, selfupdate.NewRepositorySlug(repoOwner, repoName))
	if err != nil {
		return fmt.Errorf("查询新版本失败: %w", err)
	}
	if !found || !release.GreaterThan(currentVersion) {
		fmt.Printf("已是最新版本 %s\n", currentVersion)
		return nil
	}
	fmt.Printf("发现新版本 %s(当前 %s),下载中…\n", release.Version(), currentVersion)
	if err := up.UpdateTo(ctx, release, exe); err != nil {
		return fmt.Errorf("更新失败,已保留旧版本: %w", err)
	}
	fmt.Printf("已更新: %s → %s\n", currentVersion, release.Version())
	return nil
}
