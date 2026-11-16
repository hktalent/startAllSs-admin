# startAllSs-admin

`startAllSs.sh` 的可视化配置管理台。用 Go 实现的单二进制 Web 服务，提供可配置的 Web 界面，对 `startAllSs.sh` 进行配置管理（容器、命令、项目根目录、端口、子域名、自愈周期等），并能触发脚本的启动/状态/检查/停止、查看日志、管理 launchd。

> **设计原则**：`startAllSs.sh` 本身**仍然是 macOS 服务**（由 `com.51pwn.startallss` LaunchAgent 托管，行为不变）。本项目只是它的「配置前端」，不替代它的执行职责。

---

## 功能

- **服务配置**：每个被托管服务可编辑 Docker 容器、启动命令、项目根目录、本地协议/Host/端口/探活路径、tmux 会话、公网子域名（可多条）。
- **全局配置**：根域名、Cloudflared 容器、隧道 ID、CF 模式、日志路径、**自愈周期（秒，对应 launchd `StartInterval`）**、各类 `*_ONLY` / `SKIP_*` 跳过开关。
- **应用配置**：一键把 Web 配置写盘 —— 生成 `startAllSs.config`（供脚本 source）+ 更新 plist 的 `StartInterval` + 重载 launchd，立即生效。
- **运行控制**：在界面上执行 `start / status / check / stop / restart`，实时回显输出；可开启自动刷新状态。
- **日志**：tail launchd 日志与 compose 运行日志。
- **Launchd**：查看 `launchctl print` 状态、手动重载、立即触发一次自愈。

---

## 目录结构

```
startAllSs-admin/
├── main.go          # HTTP 服务、路由、API handler
├── config.go        # 配置模型 / 默认值 / 渲染为 shell 配置
├── apply.go         # 应用配置：写 startAllSs.config + 改 plist + 重载 launchd
├── launchd.go       # plist 编辑、launchctl reload/kickstart/print
├── runner.go        # 执行 startAllSs.sh、tail 日志
├── plist.template   # 生成/兜底用的 LaunchAgent plist 模板
├── web/             # 内嵌单页 Web 界面（index.html / app.js / style.css）
├── run.sh           # 前台构建并启动（调试用）
├── install.sh       # 安装为持久 macOS LaunchAgent
└── bin/             # 构建产物（go build 生成）
```

配置文件位置（运行时自动创建）：

| 文件 | 作用 |
|---|---|
| `~/.startAllSs-admin/config.json` | Web 界面的唯一事实来源（保存的配置） |
| `<startAllSs.sh 同目录>/startAllSs.config` | 应用配置后生成，被 `startAllSs.sh` source |
| `~/Library/LaunchAgents/com.51pwn.startallss.plist` | 被管理服务的 plist，其 `StartInterval` 由本工具维护 |

---

## 快速开始

### 环境要求

- macOS（依赖 `launchctl`、`shlock`、`docker`、`tmux`、`cloudflared`，均随 `startAllSs.sh` 既有环境）
- Go 1.21+（构建用；运行二进制不需要 Go）

### 1. 前台启动（调试）

```sh
cd startAllSs-admin
./run.sh
# 等价于：go build -o bin/startAllSs-admin . && ./bin/startAllSs-admin
```

默认监听 `http://127.0.0.1:8787`。

### 2. 命令行参数

| 参数 | 默认值 | 说明 |
|---|---|---|
| `--script` | `/Users/51pwn/MyWork/goSqlite_gorm/startAllSs.sh` | 被管理的 `startAllSs.sh` 路径 |
| `--host` | `127.0.0.1` | Web 监听地址（默认仅本机） |
| `--port` | `8787` | Web 监听端口 |
| `--token` | （空） | 可选访问令牌，设置后访问 `/api/*` 需带 `?token=xxx` 或请求头 `X-Admin-Token` |

> 例如限定端口并加令牌：`./run.sh --port 9000 --token mysecret`，然后访问 `http://127.0.0.1:9000/?token=mysecret`。

### 3. 安装为持久 macOS 服务（推荐）

让管理台开机自启、崩溃自动重启：

```sh
cd startAllSs-admin
./install.sh                 # 安装到 127.0.0.1:8787
./install.sh 9000            # 指定端口
./install.sh 8787 mytoken    # 指定端口 + 访问令牌
```

安装后会注册 LaunchAgent `com.51pwn.startallss-admin`（`RunAtLoad` + `KeepAlive`），日志在 `/tmp/startAllSs-admin.log`。

查看状态：

```sh
launchctl print gui/$(id -u)/com.51pwn.startallss-admin
```

卸载：

```sh
launchctl bootout gui/$(id -u)/com.51pwn.startallss-admin
rm ~/Library/LaunchAgents/com.51pwn.startallss-admin.plist
```

---

## 使用流程（Web 界面）

打开 `http://127.0.0.1:8787`，顶栏实时显示 launchd 状态（运行态 / 自愈周期秒数 / 已运行次数），并提供「⚡ 触发」「↻ 重载」快捷按钮。

五个标签页：

### 1. 服务配置

左侧是服务卡片列表（chat / platform / scrcpy / strategy / qs / vi / gp），点击卡片在右侧编辑。每个服务可配置：

| 字段 | 对应脚本变量 | 说明 |
|---|---|---|
| 展示名 | — | 仅界面显示 |
| 类型 | — | `docker-single` / `docker-compose` / `tmux` |
| 项目根目录 | `*_DIR` | 如 `/Users/51pwn/MyWork/E2E_chat/app` |
| Docker 容器 | `*_CONTAINER` / `*_CONTAINERS` | 逗号分隔，platform 支持多个 |
| 启动命令 | `GP_RUN_CMD` 等 | 如 `python3 ./run.py` |
| 协议 / Host / 端口 | `*_LOCAL_URL` | 拼成 `scheme://host:port` |
| 本地探活路径 | — | 拼到本地 URL 后做健康检查 |
| tmux 会话 | `*_TMUX_SESSION` | tmux 类型服务使用 |
| 公网子域名 | `*_PUBLIC_URL` | 可增删多条；platform 第一条→`www`、第二条→`aitag` |
| 探活路径 | — | 公网探活用 |

`vi` 服务额外字段：`VI_QDRANT_PORT` / `VI_BIN_PATH` / `VI_LOG_DIR` / `VI_STOP_SCRIPT` / `VI_START_SCRIPT`；
`gp` 服务额外字段：`GP_PYTHON`。

> 服务的 `key`（chat/platform/...）决定它映射到脚本的哪组变量，**不可在界面修改**，避免映射错乱。

#### 新增：无域名映射的本地服务

点「＋ 新增本地服务」可以登记任意**没有公网域名**的本地服务（内网页面、本机 Docker 项目、tmux 进程等）：

- 这类服务的 `publics` 留空即可，**不会再被自动补子域名**，也不会参与公网 `https://xxx` 校验。
- 生成配置时不写 `*_PUBLIC_URL`，而是统一输出到 `LOCAL_SERVICE_COUNT` / `LOCAL_SERVICE_<n>`，由 `startAllSs.sh` 的通用逻辑托管：
  - **启动**：容器没跑 → `docker compose up -d`；容器存在但停止 → `docker start`；tmux 会话不存在 → 新建。
  - **修复**：本地 HTTP 探活不通时，自动 `docker compose restart` / `docker restart` / 重建 tmux 会话，再探一次。
  - **状态监测**：`status` 打印每个容器状态与本地 HTTP 码，`check` 只读校验，`stop` 一并停止。
- 支持三种类型：`docker-compose`（可管多个容器）、`docker-single`、`tmux`。
- 内置的 `rcm`（remote-connect-manager，`http://127.0.0.1:8088`）就是这样一个例子：双容器 `remote-connect-manager` + `rcm-guacd`，无域名映射。

### 2. 全局

- 根域名 `baseDomain`：子域名统一拼成 `<subdomain>.<baseDomain>`。
- 隧道相关：`CF_CONTAINER`、`TUNNEL_ID`、`CF_MODE`（container/host）、`CF_CONFIG`、`CF_LOG`、`CF_TUNNEL_PIDFILE`。
- 运行日志 `RUN_LOG`。
- **自愈周期（秒）**：写入 plist 的 `StartInterval`，即 launchd 每隔多久跑一次 `startAllSs.sh start` 来自愈。
- 跳过开关：对应 `STACK_ONLY` / `SKIP_BUILD` / `SCRCPY_ONLY` / `STRATEGY_ONLY` / `QS_ONLY` / `VI_ONLY` / `GP_ONLY` / `SKIP_ADB`。

### 3. 运行控制

按钮对应 `sh startAllSs.sh <action>`：`启动/自愈`、`状态`、`健康检查`、`重启`、`停止`。输出实时回显；可点「自动刷新」每 30s 拉一次状态。

### 4. 日志

切换查看 launchd 日志（`/tmp/startAllSs.launchd.log`）或 compose 运行日志（`RUN_LOG`），可调尾部行数。

### 5. Launchd

展示 `launchctl print` 的关键字段与原始输出，可「重载」（bootout+bootstrap）或「立即触发」（kickstart）。

### 保存与应用

- **「保存」**：仅把界面配置写入 `~/.startAllSs-admin/config.json`，不影响运行中的服务。
- **「应用配置」**：在保存基础上，渲染生成 `startAllSs.config` → 写入脚本同目录 → 更新 plist `StartInterval` → 重载 launchd。修改容器/目录/端口/子域名/自愈周期后，必须点「应用配置」才会对 `startAllSs.sh` 生效。
- **「预览」**：查看将要生成的 `startAllSs.config` 内容，不写盘。

---

## 它如何作用于 startAllSs.sh

`startAllSs.sh` 顶部会 source 同目录下的 `startAllSs.config`（由本工具生成）：

```sh
SAS_SELF_DIR=$(cd "$(dirname "$0")" && pwd)
SAS_CONFIG="${SAS_CONFIG:-$SAS_SELF_DIR/startAllSs.config}"
[ -f "$SAS_CONFIG" ] && . "$SAS_CONFIG" 2>/dev/null || true
```

脚本内所有常量都写成 `${VAR:-默认值}`，因此：

- 有 `startAllSs.config` 时，以其中的值为准；
- 没有该文件（或解析失败）时，完全回退到脚本内置默认值 —— **不配置 = 原行为**。

无域名映射的本地服务用另一套变量传递，每行一条：

```sh
LOCAL_SERVICE_COUNT='1'
LOCAL_SERVICE_1='rcm|remote-connect-manager|docker-compose|/Users/51pwn/MyWork/remote-connect-manager|remote-connect-manager rcm-guacd|http://127.0.0.1:8088|/|-|-'
# key|名称|类型|目录|容器(空格分隔)|本地URL|探活路径|tmux会话|启动命令（空字段写 -）
```

脚本据此逐条执行「拉起 / 修复 / 探活」，不碰任何公网域名相关逻辑；`LOCAL_SERVICE_COUNT=0`（或变量缺失）时该段完全不生效。

自愈周期不写进脚本，而是写到 LaunchAgent plist 的 `StartInterval`，由 launchd 按周期触发 `startAllSs.sh start`。

---

## HTTP API 参考

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/health` | 健康检查 |
| GET | `/api/meta` | 脚本/配置/plist 路径与当前 StartInterval |
| GET | `/api/config` | 读取当前配置 |
| PUT/POST | `/api/config` | 保存配置到 `config.json` |
| GET | `/api/render` | 预览将生成的 `startAllSs.config` |
| POST | `/api/apply` | 应用配置：写配置文件 + 改 plist + 重载 launchd |
| POST | `/api/run` | body `{"action":"start\|status\|check\|stop\|restart"}`，执行脚本 |
| GET | `/api/status` | 等价于 `action=status` |
| GET | `/api/logs?file=launchd\|compose&lines=200` | tail 日志 |
| GET | `/api/launchd` | `launchctl print` 信息 |
| POST | `/api/launchd?action=reload\|kickstart` | 重载 / 立即触发 |

示例：

```sh
# 查看配置
curl -s http://127.0.0.1:8787/api/config | python3 -m json.tool

# 应用配置
curl -s -X POST http://127.0.0.1:8787/api/apply | python3 -m json.tool

# 触发一次自愈
curl -s -X POST 'http://127.0.0.1:8787/api/launchd?action=kickstart'
```

若启动时设置了 `--token`，所有 `/api/*`（`/api/health` 除外）需附带令牌：

```sh
curl -s -H 'X-Admin-Token: mysecret' http://127.0.0.1:8787/api/config
# 或
curl -s 'http://127.0.0.1:8787/api/config?token=mysecret'
```

---

## 常见问题

**Q：改了配置为什么没生效？**
A：「保存」只写 `config.json`；需点「应用配置」才会生成 `startAllSs.config` 并重载 launchd。自愈周期改完也要点应用。

**Q：管理台本身是服务吗？和 startAllSs 的服务什么关系？**
A：两个独立的 LaunchAgent：
- `com.51pwn.startallss` —— 原有的、执行 `startAllSs.sh` 的服务（被管理对象）；
- `com.51pwn.startallss-admin` —— 本管理台（配置前端），由 `install.sh` 安装。

**Q：把配置改坏了怎么办？**
A：`startAllSs.sh` source 配置失败会自动回退默认值（`|| true`），不会导致服务起不来。也可手动删除 `<脚本目录>/startAllSs.config` 让脚本回到纯默认值；`config.json` 删除后下次启动会用内置默认值重建。

**Q：想换被管理的脚本路径？**
A：启动时加 `--script /path/to/startAllSs.sh`；生成的 `startAllSs.config` 会写到该脚本的同目录。
