package main

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

//go:embed plist.template
var plistTemplateFS embed.FS

const launchdLabel = "com.51pwn.startallss"

var startIntervalRe = regexp.MustCompile(`(?s)(<key>StartInterval</key>\s*<integer>)\d+(</integer>)`)

func plistPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/Users/51pwn"
	}
	return home + "/Library/LaunchAgents/" + launchdLabel + ".plist"
}

func launchdTarget() string {
	return fmt.Sprintf("gui/%d/%s", os.Getuid(), launchdLabel)
}

func launchdDomain() string {
	return fmt.Sprintf("gui/%d", os.Getuid())
}

// ReadStartInterval 从 plist 解析当前 StartInterval（找不到返回 0）。
func ReadStartInterval() int {
	data, err := os.ReadFile(plistPath())
	if err != nil {
		return 0
	}
	m := startIntervalRe.FindSubmatch(data)
	if len(m) < 3 {
		return 0
	}
	// m[0] 整体，m[1] 前缀，捕获的数字在 m[0] 里；重新提取
	re2 := regexp.MustCompile(`<integer>(\d+)</integer>`)
	mm := re2.FindStringSubmatch(string(m[0]))
	if len(mm) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(mm[1])
	return n
}

// SetStartInterval 修改（或创建）plist 的 StartInterval。保留 plist 其余内容不变。
func SetStartInterval(sec int) error {
	pp := plistPath()
	if sec <= 0 {
		sec = 300
	}
	data, err := os.ReadFile(pp)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// plist 不存在：用模板生成
		return writePlistFromTemplate(sec)
	}
	if startIntervalRe.Match(data) {
		data = startIntervalRe.ReplaceAll(data, []byte(`${1}`+strconv.Itoa(sec)+`${2}`))
	} else {
		// 在 </dict> 前插入 StartInterval
		insert := []byte(fmt.Sprintf("    <key>StartInterval</key>\n    <integer>%d</integer>\n", sec))
		data = bytes.Replace(data, []byte("</dict>"), append(insert, []byte("</dict>")...), 1)
	}
	return os.WriteFile(pp, data, 0o644)
}

func writePlistFromTemplate(sec int) error {
	tplBytes, err := plistTemplateFS.ReadFile("plist.template")
	if err != nil {
		return err
	}
	tpl, err := template.New("plist").Parse(string(tplBytes))
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, map[string]any{
		"Script":   scriptPath,
		"Interval": sec,
		"Home":     func() string { h, _ := os.UserHomeDir(); return h }(),
	}); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(plistPath()), 0o755); err != nil {
		return err
	}
	return os.WriteFile(plistPath(), buf.Bytes(), 0o644)
}

func runLaunchctl(args ...string) (string, error) {
	cmd := exec.Command("launchctl", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Reload 重新加载 LaunchAgent（bootout + bootstrap）。bootout 失败（未加载）忽略。
func Reload() (string, error) {
	var logs []string
	if out, err := runLaunchctl("bootout", launchdTarget()); err == nil {
		logs = append(logs, "bootout ok: "+strings.TrimSpace(out))
	} else {
		logs = append(logs, "bootout (忽略): "+strings.TrimSpace(out))
	}
	out, err := runLaunchctl("bootstrap", launchdDomain(), plistPath())
	logs = append(logs, "bootstrap: "+strings.TrimSpace(out))
	if err != nil {
		return strings.Join(logs, "\n"), err
	}
	return strings.Join(logs, "\n"), nil
}

// Kickstart 立即触发一次 start。
func Kickstart() (string, error) {
	out, err := runLaunchctl("kickstart", "-k", launchdTarget())
	return strings.TrimSpace(out), err
}

// LaunchdInfo 返回 launchctl print 的原始文本 + 关键字段。
func LaunchdInfo() (map[string]any, error) {
	out, err := runLaunchctl("print", launchdTarget())
	info := map[string]any{"raw": out}
	if err != nil {
		info["error"] = err.Error()
		return info, err
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "state ="):
			info["state"] = strings.TrimSpace(strings.TrimPrefix(line, "state ="))
		case strings.HasPrefix(line, "runs ="):
			info["runs"] = strings.TrimSpace(strings.TrimPrefix(line, "runs ="))
		case strings.HasPrefix(line, "last exit code ="):
			info["lastExitCode"] = strings.TrimSpace(strings.TrimPrefix(line, "last exit code ="))
		case strings.HasPrefix(line, "active count ="):
			info["activeCount"] = strings.TrimSpace(strings.TrimPrefix(line, "active count ="))
		}
	}
	info["startInterval"] = ReadStartInterval()
	return info, nil
}
