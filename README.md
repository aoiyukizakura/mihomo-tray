# Mihomo Tray

一个使用 Go 编写的 Windows 系统托盘工具，用于管理本地 [mihomo](https://github.com/MetaCubeX/mihomo) (Clash Meta) 代理内核及其系统级网络设置。

## 功能概览

- **系统代理控制** — 一键开关 Windows 系统代理（通过注册表 `Internet Settings`），自动配置 `127.0.0.1` 代理地址并绕过本地地址
- **代理模式切换** — 通过 Mihomo REST API 在 **Rule（规则）** / **Global（全局）** / **Direct（直连）** 三种模式间自由切换，单选互斥
- **TUN 模式** — 支持开启/关闭 TUN 虚拟网卡模式，实现全系统流量接管（包括不走系统代理的应用）
- **自动拉起内核** — 启动时自动检测并在后台启动 `mihomo.exe`，无需手动运行
- **动态托盘图标** — 三色状态指示：灰色（未运行/代理关闭）、绿色（代理中）、蓝色（TUN 模式）
- **开机自启动** — 一键注册/取消系统启动项（`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`）
- **配置热重载** — 触发 Mihomo 重新加载 `config.yaml`，无需重启内核
- **状态轮询** — 每 5 秒自动同步系统代理状态、Mihomo 运行模式与 TUN 状态
- **管理员提权** — 启动时自动检测权限，不足时通过 `ShellExecute("runas")` 触发 UAC 提权
- **灵活退出** — 支持「仅退出程序」（保持 mihomo 运行）和「退出并停止 Mihomo」两种方式

## 截图

> 右键系统托盘图标即可弹出控制菜单：

```
┌──────────────────────────────────┐
│ Mihomo 状态: 运行中 (rule)        │  ← 禁用，仅展示
├──────────────────────────────────┤
│ ✔ 系统代理                        │  ← 可切换
│ 代理模式 ▶                        │
│   ├ ✔ Rule (规则)                 │  ← 单选互斥
│   ├   Global (全局)               │
│   └   Direct (直连)               │
│ ✔ TUN 模式                        │  ← 可切换
├──────────────────────────────────┤
│ ✔ 开机自启动                      │  ← 可切换
│ 重载配置                          │  ← 点击触发
├──────────────────────────────────┤
│ 退出 ▶                           │
│   ├ 退出并停止 Mihomo             │
│   └ 仅退出程序                    │
└──────────────────────────────────┘
```

## 图标颜色含义

| 颜色 | 状态 |
|------|------|
| ![#888888](https://placehold.co/16x16/888888/888888.png) **灰色** | 系统代理关闭 / Mihomo 未运行 |
| ![#4CAF50](https://placehold.co/16x16/4CAF50/4CAF50.png) **绿色** | 系统代理已开启，Mihomo 正常运行 |
| ![#2196F3](https://placehold.co/16x16/2196F3/2196F3.png) **蓝色** | TUN 模式已激活 |

## 环境要求

- Windows 10/11 (amd64)
- Go 1.21+（仅构建需要）
- [mihomo](https://github.com/MetaCubeX/mihomo) 已安装并可在命令行中运行

## 快速开始

### 1. 安装 Mihomo

确保 `mihomo.exe` 在 PATH 环境变量中。推荐使用 Scoop：

```bash
scoop install mihomo
```

### 2. 配置文件

在 `%USERPROFILE%\.config\mihomo\config.yaml` 中确保以下字段存在：

```yaml
mixed-port: 7890
external-controller: 127.0.0.1:9090

# ... 其余代理规则配置
```

| 字段 | 说明 | 默认值 |
|------|------|--------|
| `mixed-port` | 混合代理端口（HTTP + SOCKS5） | `7890` |
| `external-controller` | REST API 监听地址 | `127.0.0.1:9090` |

> 找不到配置文件或解析失败时，程序会使用默认端口并继续运行，同时在托盘提示中显示错误信息。

### 3. 构建

```bash
git clone git@github.com:aoiyukizakura/mihomo-tray.git
cd mihomo-tray

# 普通构建（调试用，会显示控制台窗口）
go build -o mihomo-tray.exe .

# 发布构建（隐藏控制台，压缩体积）
go build -ldflags="-H windowsgui -s -w" -o mihomo-tray.exe .
```

| 参数 | 作用 |
|------|------|
| `-H windowsgui` | 隐藏控制台窗口，仅显示 GUI |
| `-s` | 去除符号表 |
| `-w` | 去除 DWARF 调试信息 |

### 4. 启动

双击 `mihomo-tray.exe` 启动，如弹出 UAC 提权窗口，点击「是」以管理员身份运行。

## 项目结构

```
mihomo-tray/
├── main.go              # 程序入口，管理员提权，托盘生命周期
├── go.mod / go.sum      # Go 模块定义
├── config/
│   └── config.go        # 配置文件解析（YAML → 端口）
├── mihomo/
│   ├── api.go           # Mihomo REST API 客户端（模式/TUN/重载）
│   └── process.go       # 进程检测、查找与启停
├── registry/
│   └── registry.go      # Windows 注册表操作（代理/自启动）
├── icons/
│   └── icons.go         # 内存生成托盘图标（ICO，16/24/32px）
├── state/
│   └── state.go         # 后台 5 秒轮询状态管理（发布/订阅模式）
├── menu/
│   └── menu.go          # 托盘菜单构建与实时刷新
└── manifest/
    └── mihomo-tray.exe.manifest  # 管理员权限清单（可选）
```

## 局限性

### 平台限制
- **仅支持 Windows** — 项目深度依赖 Win32 API（注册表、ShellExecute、tasklist/taskkill、systray），无法在其他操作系统上运行

### 权限要求
- **需要管理员权限** — 修改系统代理设置和启动 mihomo 进程均需要管理员权限，程序启动时会自动请求 UAC 提权

### 路径硬编码
- **配置文件路径固定** — Mihomo 配置文件固定为 `%USERPROFILE%\.config\mihomo\config.yaml`，不支持通过命令行参数或环境变量自定义
- **mihomo.exe 查找范围有限** — 仅在 PATH、程序所在目录、`~/scoop/shims/` 三个位置查找，不支持手动指定 mihomo 路径

### 功能限制
- **仅支持单用户** — 所有注册表操作仅作用于 `HKEY_CURRENT_USER`，不支持系统级（`HKLM`）配置
- **无左键交互** — 左键单击托盘图标无任何操作，所有功能仅通过右键菜单访问
- **不支持多实例** — 无法同时管理多个 mihomo 进程或配置文件
- **无日志系统** — 程序本身不产生日志文件，调试依赖控制台输出（需非 `windowsgui` 构建）
- **轮询间隔固定** — 状态刷新间隔固定为 5 秒，无法自定义
- **API 超时硬编码** — 所有 Mihomo API 调用超时固定为 2 秒，不可配置

### 界面限制
- **仅支持中文** — 所有菜单文本、提示信息均为简体中文，不支持国际化/多语言
- **托盘图标固定** — 图标颜色和尺寸（16/24/32px）在编译时确定，不支持自定义主题或图标

### 安全与更新
- **无自动更新** — 不包含版本检查或自动更新机制，需手动下载新版本
- **无签名验证** — 二进制文件无代码签名，Windows SmartScreen 可能弹出警告
- **明文通信** — 与 Mihomo API 之间使用 HTTP（127.0.0.1 环回地址），无 TLS 加密（本地通信通常可接受）

## 注意事项

- 程序退出时默认不结束 `mihomo.exe`，如需同时结束请选择「退出并停止 Mihomo」
- 如 Mihomo 未运行，菜单中「代理模式」、「TUN 模式」等依赖 API 的选项会自动禁用
- 启动时若找不到 `mihomo.exe`，托盘提示会显示具体错误信息

## License

MIT License
