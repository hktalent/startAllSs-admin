package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// PublicEntry 一个公网子域名入口（子域名 + 探活路径）。
type PublicEntry struct {
	Subdomain  string `json:"subdomain"`
	HealthPath string `json:"healthPath"`
}

// Service 单个被 startAllSs.sh 托管的服务。
type Service struct {
	Key             string        `json:"key"`               // 脚本变量前缀：chat/platform/scrcpy/strategy/qs/vi/gp（其它 key 走通用本地服务）
	Name            string        `json:"name"`              // 展示名
	Type            string        `json:"type"`              // docker-single | docker-compose | tmux
	Builtin         bool          `json:"builtin,omitempty"` // 内置服务（升级后会按 key 自动补回到旧配置）
	Enabled         bool          `json:"enabled"`           // 是否纳入托管（通用本地服务按此开关渲染）
	Dir             string        `json:"dir"`               // 项目根目录
	Containers      []string      `json:"containers"`        // docker 容器名（可多个，平台栈用）
	Command         string        `json:"command"`           // 启动命令（如 gp: python3 ./run.py）
	LocalScheme     string        `json:"localScheme"`       // http | https
	LocalHost       string        `json:"localHost"`         // 一般 127.0.0.1
	LocalPort       int           `json:"localPort"`         // 本地端口
	LocalHealthPath string        `json:"localHealthPath"`   // 本地探活路径
	TmuxSession     string        `json:"tmuxSession"`       // tmux 会话名（tmux 类型）
	Publics         []PublicEntry `json:"publics"`           // 公网子域名列表
	// vi 专属
	QdrantPort  int    `json:"qdrantPort,omitempty"`
	BinPath     string `json:"binPath,omitempty"`
	LogDir      string `json:"logDir,omitempty"`
	StopScript  string `json:"stopScript,omitempty"`
	StartScript string `json:"startScript,omitempty"`
	// gp 专属
	Python string `json:"python,omitempty"`
}

// Global 脚本全局常量。
type Global struct {
	BaseDomain         string `json:"baseDomain"`
	CfContainer        string `json:"cfContainer"`
	TunnelId           string `json:"tunnelId"`
	CfMode             string `json:"cfMode"` // container | host
	CfConfig           string `json:"cfConfig"`
	CfLog              string `json:"cfLog"`
	CfPidFile          string `json:"cfPidFile"`
	RunLog             string `json:"runLog"`
	AutoFixIntervalSec int    `json:"autoFixIntervalSec"` // launchd StartInterval
}

// SkipFlags 对应脚本的 *_ONLY / SKIP_* 环境变量。
type SkipFlags struct {
	StackOnly    bool `json:"stackOnly"`
	SkipBuild    bool `json:"skipBuild"`
	ScrcpyOnly   bool `json:"scrcpyOnly"`
	StrategyOnly bool `json:"strategyOnly"`
	QsOnly       bool `json:"qsOnly"`
	ViOnly       bool `json:"viOnly"`
	GpOnly       bool `json:"gpOnly"`
	SkipAdb      bool `json:"skipAdb"`
}

// Config 是 Web 界面的唯一事实来源。
type Config struct {
	Global    Global    `json:"global"`
	SkipFlags SkipFlags `json:"skipFlags"`
	Services  []Service `json:"services"`
}

func defaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		Global: Global{
			BaseDomain:         "51pwn.com",
			CfContainer:        "51pwn-cloudflared",
			TunnelId:           "416d8666-6b3c-4232-8fde-a3995eabfa40",
			CfMode:             "container",
			CfConfig:           home + "/.cloudflared/config.yml",
			CfLog:              "/tmp/cloudflared.tunnel.out.log",
			CfPidFile:          "/tmp/cloudflared_tunnel.pid",
			RunLog:             "/tmp/startAllSs.compose.log",
			AutoFixIntervalSec: 300,
		},
		SkipFlags: SkipFlags{},
		Services: []Service{
			{
				Key: "chat", Name: "E2E_chat", Type: "docker-single", Builtin: true, Enabled: true,
				Dir: "/Users/51pwn/MyWork/E2E_chat/app", Containers: []string{"e2e_chat"},
				LocalScheme: "https", LocalHost: "127.0.0.1", LocalPort: 2083, LocalHealthPath: "/",
				Publics: []PublicEntry{{Subdomain: "chat", HealthPath: "/"}},
			},
			{
				Key: "platform", Name: "51pwn 平台栈", Type: "docker-compose", Builtin: true, Enabled: true,
				Dir:         "/Users/51pwn/Documents/Trae-CN/temp_20261103_02/51pwn-docker",
				Containers:  []string{"51pwn-platform", "51pwn-nginx", "51pwn-cloudflared"},
				LocalScheme: "http", LocalHost: "127.0.0.1", LocalPort: 9080, LocalHealthPath: "/",
				Publics: []PublicEntry{
					{Subdomain: "www", HealthPath: "/"},
					{Subdomain: "aitag", HealthPath: "/"},
				},
			},
			{
				Key: "scrcpy", Name: "ws-scrcpy-home-control", Type: "docker-single", Builtin: true, Enabled: true,
				Dir: "/Users/51pwn/MyWork/ws-scrcpy-home-control", Containers: []string{"ws-scrcpy-home"},
				LocalScheme: "http", LocalHost: "127.0.0.1", LocalPort: 8080, LocalHealthPath: "/api/health",
				Publics: []PublicEntry{{Subdomain: "tv", HealthPath: "/"}},
			},
			{
				Key: "strategy", Name: "strategy-web", Type: "docker-single", Builtin: true, Enabled: true,
				Dir: "/Users/51pwn/MyWork/strategy-web", Containers: []string{"strategy-web"},
				LocalScheme: "http", LocalHost: "127.0.0.1", LocalPort: 3000, LocalHealthPath: "/api/meta",
				Publics: []PublicEntry{{Subdomain: "stock", HealthPath: "/"}},
			},
			{
				Key: "qs", Name: "QuietShield", Type: "tmux", Builtin: true, Enabled: true,
				Dir: "/Users/51pwn/MyWork/QuietShield", TmuxSession: "quietshield",
				LocalScheme: "https", LocalHost: "127.0.0.1", LocalPort: 8808, LocalHealthPath: "/",
				Publics: []PublicEntry{{Subdomain: "zs", HealthPath: "/"}},
			},
			{
				Key: "vi", Name: "vi · 见到你", Type: "tmux", Builtin: true, Enabled: true,
				Dir: "/Users/51pwn/MyWork/vi", TmuxSession: "vi",
				LocalScheme: "http", LocalHost: "127.0.0.1", LocalPort: 8688, LocalHealthPath: "/api/health",
				QdrantPort: 6333, BinPath: "/Users/51pwn/MyWork/vi/bin/vi", LogDir: "/Users/51pwn/MyWork/vi/data",
				StopScript: "/Users/51pwn/MyWork/vi/stop.sh", StartScript: "/Users/51pwn/MyWork/vi/start.sh",
				Publics: []PublicEntry{{Subdomain: "vi", HealthPath: "/"}},
			},
			{
				Key: "gp", Name: "gp · 股票系统", Type: "tmux", Builtin: true, Enabled: true,
				Dir: "/Users/51pwn/MyWork/stock-system", TmuxSession: "gp",
				Command: "python3 ./run.py", Python: "python3",
				LocalScheme: "http", LocalHost: "127.0.0.1", LocalPort: 8010, LocalHealthPath: "/",
				Publics: []PublicEntry{{Subdomain: "gp", HealthPath: "/"}},
			},
			{
				// 无域名映射的本地服务示例：docker compose 双容器（应用 + guacd）
				Key: "rcm", Name: "remote-connect-manager", Type: "docker-compose", Builtin: true, Enabled: true,
				Dir:         "/Users/51pwn/MyWork/remote-connect-manager",
				Containers:  []string{"remote-connect-manager", "rcm-guacd"},
				LocalScheme: "http", LocalHost: "127.0.0.1", LocalPort: 8088, LocalHealthPath: "/",
				Publics: []PublicEntry{},
			},
		},
	}
}

func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/Users/51pwn"
	}
	return filepath.Join(home, ".startAllSs-admin", "config.json")
}

func loadConfig() (Config, error) {
	cfg := defaultConfig()
	data, err := os.ReadFile(configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("解析 config.json 失败: %w", err)
	}
	cfg.fillDefaults()
	return cfg, nil
}

func (c *Config) fillDefaults() {
	def := defaultConfig()
	if c.Global.BaseDomain == "" {
		c.Global.BaseDomain = def.Global.BaseDomain
	}
	if c.Global.AutoFixIntervalSec <= 0 {
		c.Global.AutoFixIntervalSec = def.Global.AutoFixIntervalSec
	}
	if c.Global.CfMode == "" {
		c.Global.CfMode = def.Global.CfMode
	}
	for i := range c.Services {
		s := &c.Services[i]
		if s.LocalHealthPath == "" {
			s.LocalHealthPath = "/"
		}
		if s.Type == "" {
			s.Type = "docker-single"
		}
		// 注意：publics 允许为空 —— 空表示「无域名映射的纯本地服务」，由脚本通用逻辑托管，
		// 不要在这里补默认子域名，否则会被误当成需要公网校验的服务。
	}
	c.mergeBuiltins()
}

// mergeBuiltins 把新增的内置服务补进旧 config.json（已存在同 key 的不动，删除过的内置服务会被补回）。
func (c *Config) mergeBuiltins() {
	have := make(map[string]bool, len(c.Services))
	for _, s := range c.Services {
		have[s.Key] = true
	}
	for _, d := range defaultConfig().Services {
		if d.Builtin && !have[d.Key] {
			c.Services = append(c.Services, d)
		}
	}
}

func saveConfig(cfg Config) error {
	cfg.fillDefaults()
	if err := os.MkdirAll(filepath.Dir(configPath()), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0o644)
}

// shellQuote 把值安全地嵌入单引号 shell 赋值。
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func publicURL(scheme, subdomain, base string) string {
	if subdomain == "" {
		return scheme + "://" + base
	}
	return scheme + "://" + subdomain + "." + base
}

func localURL(scheme, host string, port int) string {
	return fmt.Sprintf("%s://%s:%d", scheme, host, port)
}

// RenderShellConfig 把 Config 渲染成 startAllSs.sh 可 source 的 shell 配置文件内容。
func (c *Config) RenderShellConfig() string {
	var b strings.Builder
	b.WriteString("# 本文件由 startAllSs-admin 自动生成，请勿手动编辑。\n")
	b.WriteString("# 生成时间: " + time.Now().Format("2006-01-02 15:04:05") + "\n")
	b.WriteString("# 修改请通过 Web 界面保存并「应用配置」。\n\n")

	w := func(k, v string) {
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(shellQuote(v))
		b.WriteString("\n")
	}
	wInt := func(k string, v int) { w(k, fmt.Sprintf("%d", v)) }
	wBool := func(k string, v bool) {
		if v {
			w(k, "1")
		} else {
			w(k, "0")
		}
	}

	// 跳过 / 模式开关
	wBool("STACK_ONLY", c.SkipFlags.StackOnly)
	wBool("SKIP_BUILD", c.SkipFlags.SkipBuild)
	wBool("SCRCPY_ONLY", c.SkipFlags.ScrcpyOnly)
	wBool("STRATEGY_ONLY", c.SkipFlags.StrategyOnly)
	wBool("QS_ONLY", c.SkipFlags.QsOnly)
	wBool("VI_ONLY", c.SkipFlags.ViOnly)
	wBool("GP_ONLY", c.SkipFlags.GpOnly)
	wBool("SKIP_ADB", c.SkipFlags.SkipAdb)
	w("CF_MODE", c.Global.CfMode)

	// 隧道 / 日志全局
	w("CF_CONTAINER", c.Global.CfContainer)
	w("TUNNEL_ID", c.Global.TunnelId)
	w("CF_CONFIG", c.Global.CfConfig)
	w("CF_LOG", c.Global.CfLog)
	w("CF_TUNNEL_PIDFILE", c.Global.CfPidFile)
	w("RUN_LOG", c.Global.RunLog)

	// 按 key 排序输出，保证稳定
	keys := make([]string, 0, len(c.Services))
	byKey := make(map[string]Service)
	for _, s := range c.Services {
		keys = append(keys, s.Key)
		byKey[s.Key] = s
	}
	sort.Strings(keys)

	writeCommon := func(prefix string, s Service) {
		w(prefix+"_DIR", s.Dir)
		w(prefix+"_LOCAL_URL", localURL(s.LocalScheme, s.LocalHost, s.LocalPort))
		if len(s.Publics) > 0 {
			w(prefix+"_PUBLIC_URL", publicURL("https", s.Publics[0].Subdomain, c.Global.BaseDomain))
		}
	}

	// 非内置 key 的服务 = 无域名映射的本地服务，统一交给脚本的 LOCAL_SERVICE_* 通用逻辑
	localCount := 0
	var localLines strings.Builder

	for _, k := range keys {
		s := byKey[k]
		b.WriteString("\n# --- " + s.Name + " (" + k + ") ---\n")
		switch k {
		case "chat":
			writeCommon("CHAT", s)
			if len(s.Containers) > 0 {
				w("CHAT_CONTAINER", s.Containers[0])
			}
		case "platform":
			writeCommon("PLATFORM", s)
			w("PLATFORM_CONTAINERS", strings.Join(s.Containers, " "))
			if len(s.Publics) > 1 {
				w("AITAG_PUBLIC_URL", publicURL("https", s.Publics[1].Subdomain, c.Global.BaseDomain))
			}
		case "scrcpy":
			writeCommon("SCRCPY", s)
			if len(s.Containers) > 0 {
				w("SCRCPY_CONTAINER", s.Containers[0])
			}
		case "strategy":
			writeCommon("STRATEGY", s)
			if len(s.Containers) > 0 {
				w("STRATEGY_CONTAINER", s.Containers[0])
			}
		case "qs":
			writeCommon("QS", s)
			w("QS_TMUX_SESSION", s.TmuxSession)
		case "vi":
			writeCommon("VI", s)
			w("VI_TMUX_SESSION", s.TmuxSession)
			wInt("VI_QDRANT_PORT", s.QdrantPort)
			// 生成的是 shell 单引号字面量，$VI_DIR 不会展开 → 这里直接落成绝对路径
			expandDir := func(v string) string {
				return strings.ReplaceAll(v, "$VI_DIR", s.Dir)
			}
			w("VI_BIN_PATH", expandDir(s.BinPath))
			w("VI_LOG_DIR", expandDir(s.LogDir))
			w("VI_STOP_SCRIPT", expandDir(s.StopScript))
			w("VI_START_SCRIPT", expandDir(s.StartScript))
		case "gp":
			writeCommon("GP", s)
			w("GP_TMUX_SESSION", s.TmuxSession)
			w("GP_PYTHON", s.Python)
			w("GP_RUN_CMD", s.Command)
		default:
			// 新增的本地服务（无域名映射）：不生成 *_PUBLIC_URL，只登记到 LOCAL_SERVICE_*
			if !s.Enabled {
				b.WriteString("# （已停用，不纳入托管）\n")
				continue
			}
			localCount++
			localLines.WriteString(fmt.Sprintf("LOCAL_SERVICE_%d=%s\n", localCount, shellQuote(localServiceEntry(s))))
		}
	}

	b.WriteString("\n# --- 无域名映射的本地服务（通用托管：拉起 / 修复 / 状态监测）---\n")
	b.WriteString(fmt.Sprintf("LOCAL_SERVICE_COUNT=%s\n", shellQuote(fmt.Sprintf("%d", localCount))))
	b.WriteString(localLines.String())
	return b.String()
}

// localServiceEntry 渲染成脚本可解析的单行条目：
// key|名称|类型|目录|容器(空格分隔)|本地URL|探活路径|tmux会话|启动命令
func localServiceEntry(s Service) string {
	clean := func(v string) string {
		v = strings.ReplaceAll(v, "|", "/")
		v = strings.ReplaceAll(v, "\n", " ")
		v = strings.TrimSpace(v)
		if v == "" {
			return "-"
		}
		return v
	}
	containers := "-"
	if len(s.Containers) > 0 {
		containers = strings.Join(s.Containers, " ")
	}
	scheme := s.LocalScheme
	if scheme == "" {
		scheme = "http"
	}
	host := s.LocalHost
	if host == "" {
		host = "127.0.0.1"
	}
	url := "-"
	if s.LocalPort > 0 {
		url = localURL(scheme, host, s.LocalPort)
	}
	fields := []string{
		clean(s.Key), clean(s.Name), clean(s.Type), clean(s.Dir),
		clean(containers), clean(url), clean(s.LocalHealthPath),
		clean(s.TmuxSession), clean(s.Command),
	}
	return strings.Join(fields, "|")
}
