# cstop

<p align="left"><img src="assets/icon.png" width="120" alt="cstop logo"></p>

Claude Code 会话监控工具。htop 风格的终端 TUI,实时显示本机 Claude Code 的运行会话、内存、磁盘占用与最近活动。

## 功能

- **运行中会话**: PID、项目、内存、运行时长、最近动作(Bash / 回复 / 用户 · 时间)
- **磁盘占用**: `~/.claude` 全目录口径——projects 按项目,plugins / file-history 等一级目录单列;与 `du` 同为磁盘块字节
- **阈值告警**: 超限标黄,超限 1.5 倍标红,状态栏汇总 ⚠;阈值全部可配
- **会话详情**: Enter 查看单个会话最近 20 条活动与完整路径
- **自动更新**: 启动时后台检测新版本,按 `u` 或运行 `cstop update` 一键升级(下载校验 sha256)

macOS / Linux / Windows 三平台。

## 安装

### Homebrew(推荐)

```bash
brew install gebilaoman/tap/cstop
```

### 一键脚本(macOS / Linux)

自动识别平台,下载最新版并校验 sha256,安装到 `~/.local/bin`:

```bash
curl -fsSL https://raw.githubusercontent.com/gebilaoman/claude-session-monitor/main/install.sh | sh
```

### 直接下载

从 [Releases](https://github.com/gebilaoman/claude-session-monitor/releases) 下载对应平台产物,解压后将 `cstop` 放入 PATH:

```
cstop_<版本>_darwin_arm64.tar.gz   # macOS Apple Silicon
cstop_<版本>_darwin_amd64.tar.gz   # macOS Intel
cstop_<版本>_linux_amd64.tar.gz    # Linux x86_64
cstop_<版本>_windows_amd64.zip     # Windows
```

### 源码构建

要求 Go 1.23+:

```bash
git clone https://github.com/gebilaoman/claude-session-monitor.git
cd claude-session-monitor/code
go build -o cstop ./cmd/cstop
```

## 使用

```bash
cstop          # 启动 TUI
cstop --dump   # 打印一份文本快照(脚本化)
cstop update   # 更新到最新版本
```

TUI 键位:

| 键 | 功能 |
|---|---|
| `1` / `2` | 切换 运行中 / 磁盘 tab |
| `↑` `↓` | 滚动 |
| `Enter` | 会话详情 |
| `Esc` | 返回 |
| `r` | 立即刷新 |
| `u` | 有新版本时执行更新 |
| `q` | 退出 |

## 配置

配置文件: Linux/macOS `~/.config/cstop/config.toml`,Windows `%APPDATA%\cstop\config.toml`。`cstop --init-config` 生成。文件不存在时使用内置默认值。

| 键 | 默认 | 含义 |
|---|---|---|
| refresh_procs_sec | 2 | 进程与活动刷新间隔(秒) |
| refresh_disk_sec | 60 | 磁盘扫描间隔(秒) |
| warn_session_rss_mb | 400 | 单会话内存告警线 |
| warn_total_rss_mb | 1500 | 全部会话内存合计告警线 |
| warn_disk_project_mb | 500 | 单项磁盘告警线 |
| warn_disk_total_mb | 2000 | ~/.claude 总量告警线 |
| warn_session_count | 6 | 运行会话数告警线 |
| update_check_on_start | true | 启动时后台检测新版本 |

## 已知限制

- 经包管理器(brew)安装时,`cstop update` 会替换包管理器管理的文件;此类安装请用 `brew upgrade` 更新。
- Windows 上进程 cwd 常取不到,项目列回退显示 `--resume` 的会话 id。

## 设计文档

见 [docs/20261108-cstop-设计文档.md](docs/20261108-cstop-设计文档.md)。
