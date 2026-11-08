#!/bin/sh
# 把 startAllSs-admin Web 管理台安装为 macOS LaunchAgent：
#   - 开机登录后自动启动，崩溃自动重启（KeepAlive）
#   - 仅监听 127.0.0.1，默认端口 8787
# 注意：startAllSs.sh 本身仍由 com.51pwn.startallss 托管，本脚本只安装「管理界面」。
#
# 用法:
#   ./install.sh                 # 安装到 127.0.0.1:8787
#   ./install.sh 9000            # 指定端口
#   ./install.sh 8787 mytoken    # 指定端口 + 访问令牌（需 ?token=mytoken 或 X-Admin-Token）
# 卸载:
#   launchctl bootout gui/$(id -u)/com.51pwn.startallss-admin
set -e
DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$DIR"
mkdir -p bin
go build -o bin/startAllSs-admin .

PORT="${1:-8787}"
TOKEN="${2:-}"
LABEL="com.51pwn.startallss-admin"
PLIST="$HOME/Library/LaunchAgents/$LABEL.plist"
LOG="/tmp/startAllSs-admin.log"
mkdir -p "$(dirname "$PLIST")"

ARGS="<string>$DIR/bin/startAllSs-admin</string>
        <string>--host</string><string>127.0.0.1</string>
        <string>--port</string><string>$PORT</string>"
if [ -n "$TOKEN" ]; then
  ARGS="$ARGS
        <string>--token</string><string>$TOKEN</string>"
fi

cat > "$PLIST" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>$LABEL</string>
    <key>ProgramArguments</key>
    <array>
        $ARGS
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/go/bin:/usr/local/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
        <key>HOME</key>
        <string>$HOME</string>
    </dict>
    <key>StandardOutPath</key>
    <string>$LOG</string>
    <key>StandardErrorPath</key>
    <string>$LOG</string>
</dict>
</plist>
EOF

launchctl bootout "gui/$(id -u)/$LABEL" 2>/dev/null || true
launchctl bootstrap "gui/$(id -u)" "$PLIST"
sleep 1
echo "✅ 已安装 $LABEL"
echo "   管理台: http://127.0.0.1:$PORT${TOKEN:+?token=$TOKEN}"
echo "   日志:   tail -f $LOG"
echo "   状态:   launchctl print gui/$(id -u)/$LABEL"
