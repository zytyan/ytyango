#!/usr/bin/env bash
set -e

SERVICE_NAME=goytyan
BUILD_DIR=build
EXEC_NAME=ytyan-go
USER=tgbotapi
GROUP=tgbots
SYSTEMD_FILE="/etc/systemd/system/$SERVICE_NAME.service"

function build() {
    local auto_restart=0
    local no_pull=0
    local dry_run=0
    for arg in "$@"; do
        case "$arg" in
            --no-pull) no_pull=1 ;;
            -y) auto_restart=1 ;;
            -n) dry_run=1 ;;
        esac
    done

    [[ "$no_pull" -eq 0 ]] && {
        echo "🔃 Pulling git repo..."
        if [[ "$dry_run" -eq 1 ]]; then
            git clean -fd
        else
            git clean -fdn
            exit 1
        fi
        git pull
    } || echo "🚫 Skipping git pull"

    mkdir -p "$BUILD_DIR"
    go get
    go build -ldflags "-X 'main.compileTime=$(date '+%Y-%m-%d %H:%M:%S')'" -tags=jsoniter -o "$BUILD_DIR/$EXEC_NAME"
    echo "✅ Compile done at $(date '+%Y-%m-%d %H:%M:%S')"

    if [[ "$auto_restart" -eq 1 ]]; then
        echo "🚀 自动重启服务中..."
        sudo systemctl daemon-reload
        sudo systemctl restart "$SERVICE_NAME"
    else
        read -rp "是否要重启服务？[y/n] " isRestart
        [[ "$isRestart" == "y" ]] && {
            sudo systemctl daemon-reload
            sudo systemctl restart "$SERVICE_NAME"
        } || echo "跳过重启。"
    fi
}


function install() {
    echo "🔧 Installing systemd service..."

    if ! id -u "$USER" &>/dev/null; then
        echo "Creating user: $USER"
        sudo groupadd -f "$GROUP"
        sudo useradd -r -g "$GROUP" -d "$(pwd)/$BUILD_DIR" -s /sbin/nologin "$USER"
    else
        echo "User $USER already exists"
    fi

    echo "Setting ownership of $BUILD_DIR to $USER:$GROUP"
    sudo chown -R "$USER:$GROUP" "$BUILD_DIR"

    echo "Writing service file to $SYSTEMD_FILE"
    sudo tee "$SYSTEMD_FILE" >/dev/null <<EOF
[Unit]
Description=Ytyan main bot
After=network.target telegram-bot-api.service
StartLimitIntervalSec=60
StartLimitBurst=5

[Service]
Restart=on-failure
RestartSec=5s
Type=simple
Environment="GOYTYAN_NO_STDOUT=1"
Environment="GOYTYAN_CONFIG=config.yaml"
Environment="GOYTYAN_LOG_FILE=logs/log.log"
Environment="TZ=Asia/Shanghai"
KillSignal=SIGINT
WorkingDirectory=$(pwd)/$BUILD_DIR
ExecStart=$(pwd)/$BUILD_DIR/$EXEC_NAME
User=$USER
Group=$GROUP

[Install]
WantedBy=multi-user.target
EOF

    sudo systemctl daemon-reload
    sudo systemctl enable "$SERVICE_NAME"
    echo "✅ Installed and enabled $SERVICE_NAME"
}

function control_service() {
    action="$1"
    echo "🔄 $action $SERVICE_NAME"
    sudo systemctl "$action" "$SERVICE_NAME"
}

function view_log() {
    LOG_FILE="$BUILD_DIR/logs/log.log"
    local lines=20
    local follow=0

    for arg in "$@"; do
        case "$arg" in
            -f) follow=1 ;;
            --lines=*)
                lines="${arg#--lines=}"
                ;;
        esac
    done

    if [[ ! -f "$LOG_FILE" ]]; then
        echo "❌ Log file not found: $LOG_FILE"
        exit 1
    fi

    if [[ "$follow" -eq 1 ]]; then
        tail -n "$lines" -f "$LOG_FILE"
    else
        tail -n "$lines" "$LOG_FILE"
    fi
}

function usage() {
    cat <<EOF
Usage: $0 <command> [options]

Commands:
  build [--no-pull] [-y] [-n]        拉取代码并构建项目，支持自动重启服务
  install                            安装 systemd 服务，创建用户组并设定权限
  start | stop | restart | status    控制 systemd 服务状态
  log [-f] [--lines=N]               查看日志，支持 -f 追加模式 和 --lines 指定行数
EOF
}

# 主控制逻辑
case "$1" in
    build)
        shift
        build "$@"
        ;;
    install)
        install
        ;;
    start|stop|restart|status)
        control_service "$1"
        ;;
    log)
        shift
        view_log "$@"
        ;;
    *)
        usage
        exit 1
        ;;
esac

