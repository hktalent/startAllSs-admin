package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:embed web
var webFS embed.FS

var (
	scriptPath string
	listenHost string
	listenPort int
	adminToken string
)

func main() {
	flag.StringVar(&scriptPath, "script", "/Users/51pwn/MyWork/goSqlite_gorm/startAllSs.sh", "startAllSs.sh 脚本路径")
	flag.StringVar(&listenHost, "host", "127.0.0.1", "Web 监听地址（默认仅本机）")
	flag.IntVar(&listenPort, "port", 8787, "Web 监听端口")
	flag.StringVar(&adminToken, "token", "", "可选：访问 /api 需要的令牌（X-Admin-Token 或 ?token=）")
	flag.Parse()

	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(sub)))

	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/meta", handleMeta)
	mux.HandleFunc("/api/config", handleConfig)
	mux.HandleFunc("/api/render", handleRender)
	mux.HandleFunc("/api/apply", handleApply)
	mux.HandleFunc("/api/run", handleRun)
	mux.HandleFunc("/api/status", handleStatus)
	mux.HandleFunc("/api/logs", handleLogs)
	mux.HandleFunc("/api/launchd", handleLaunchd)

	addr := fmt.Sprintf("%s:%d", listenHost, listenPort)
	log.Printf("startAllSs-admin 启动: http://%s  (script=%s)", addr, scriptPath)
	log.Printf("配置文件: %s", configPath())
	log.Printf("生成配置: %s", generatedConfigPath())
	log.Printf("LaunchAgent: %s", plistPath())
	if err := http.ListenAndServe(addr, withAuth(mux)); err != nil {
		log.Fatal(err)
	}
}

// withAuth 若设置了 token，则对 /api/* 校验令牌。
func withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if adminToken == "" || !strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}
		tok := r.Header.Get("X-Admin-Token")
		if tok == "" {
			tok = r.URL.Query().Get("token")
		}
		if tok != adminToken {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"ok": true, "time": time.Now().Format(time.RFC3339)})
}

func handleMeta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"scriptPath":        scriptPath,
		"generatedConfig":   generatedConfigPath(),
		"configPath":        configPath(),
		"plistPath":         plistPath(),
		"plistStartInterval": ReadStartInterval(),
		"launchdLabel":      launchdLabel,
		"launchdTarget":     launchdTarget(),
	})
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := loadConfig()
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		writeJSON(w, cfg)
	case http.MethodPut, http.MethodPost:
		var cfg Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, 400, "JSON 解析失败: "+err.Error())
			return
		}
		if err := saveConfig(cfg); err != nil {
			writeError(w, 500, err.Error())
			return
		}
		writeJSON(w, map[string]any{"ok": true, "saved": configPath()})
	default:
		writeError(w, 405, "method not allowed")
	}
}

func handleRender(w http.ResponseWriter, r *http.Request) {
	preview, err := RenderPreview()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]string{"preview": preview})
}

func handleApply(w http.ResponseWriter, r *http.Request) {
	res, err := Apply()
	if err != nil && res == nil {
		writeError(w, 500, err.Error())
		return
	}
	out := map[string]any{}
	if res != nil {
		out = map[string]any{
			"ok":              err == nil,
			"configPath":      res.ConfigPath,
			"generatedPath":   res.GeneratedPath,
			"plistPath":       res.PlistPath,
			"startInterval":   res.StartInterval,
			"launchdReloaded": res.LaunchdReloaded,
			"reloadLog":       res.ReloadLog,
			"preview":         res.Preview,
		}
	}
	if err != nil {
		out["error"] = err.Error()
	}
	writeJSON(w, out)
}

func actionTimeout(action string) time.Duration {
	switch action {
	case "start", "restart":
		return 900 * time.Second
	case "stop":
		return 180 * time.Second
	case "status", "check":
		return 300 * time.Second
	default:
		return 90 * time.Second
	}
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "method not allowed")
		return
	}
	var req struct {
		Action string `json:"action"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	switch req.Action {
	case "start", "status", "check", "stop", "restart":
	default:
		writeError(w, 400, "非法 action: "+req.Action)
		return
	}
	out, err := RunScript(req.Action, actionTimeout(req.Action))
	writeJSON(w, map[string]any{
		"action": req.Action,
		"output": out,
		"ok":     err == nil,
		"error":  errString(err),
	})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	out, err := RunScript("status", actionTimeout("status"))
	writeJSON(w, map[string]any{"output": out, "ok": err == nil, "error": errString(err)})
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	lines, _ := strconv.Atoi(r.URL.Query().Get("lines"))
	if lines <= 0 {
		lines = 200
	}
	cfg, _ := loadConfig()
	var path string
	switch file {
	case "compose":
		path = cfg.Global.RunLog
	default:
		path = "/tmp/startAllSs.launchd.log"
	}
	if path == "" {
		writeError(w, 400, "未知日志文件: "+file)
		return
	}
	text, err := TailLog(path, lines)
	writeJSON(w, map[string]any{"file": file, "path": path, "lines": lines, "output": text, "error": errString(err)})
}

func handleLaunchd(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		info, err := LaunchdInfo()
		writeJSON(w, map[string]any{"info": info, "error": errString(err)})
	case http.MethodPost:
		act := r.URL.Query().Get("action")
		switch act {
		case "reload":
			log, err := Reload()
			writeJSON(w, map[string]any{"ok": err == nil, "log": log, "error": errString(err)})
		case "kickstart":
			log, err := Kickstart()
			writeJSON(w, map[string]any{"ok": err == nil, "log": log, "error": errString(err)})
		default:
			writeError(w, 400, "未知 action: "+act)
		}
	default:
		writeError(w, 405, "method not allowed")
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
