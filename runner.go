package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// generatedConfigPath 返回写入到脚本同目录下、供 startAllSs.sh source 的配置文件。
func generatedConfigPath() string {
	return filepath.Join(filepath.Dir(scriptPath), "startAllSs.config")
}

// RunScript 执行 `sh <script> <action>`，返回合并输出与错误（非零退出也算 err）。
func RunScript(action string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", scriptPath, action)
	cmd.Env = append(os.Environ(), "SAS_CONFIG="+generatedConfigPath())
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(out), fmt.Errorf("执行超时（>%s）", timeout)
	}
	return string(out), err
}

// TailLog 读取日志文件最后 n 行。
func TailLog(path string, n int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n"), nil
}
