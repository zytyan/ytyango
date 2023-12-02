#!/usr/bin/env bash
set -e

function cmd_not_exists_exit() {
    if ! [ -x "$(command -v "$1")" ]; then
        echo 'Error: '"$1"' is not installed.' >&2
    fi
}

function check_dependencies() {
    # 需要用到的依赖： go ffmpeg ffprobe yt-dlp libwebp-devel
    # 由于使用了CGO(webp, sqlite)，还需要C compiler，懒狗这里先不检查了
    cmd_not_exists_exit "go"
    cmd_not_exists_exit "ffmpeg"
    cmd_not_exists_exit "ffprobe"
    cmd_not_exists_exit "yt-dlp"

}

function install_service() {
    # 安装服务
    cp ./goytyan.service.template ./goytyan.service
    sed -i "s|/CUR_PATH|$(pwd)|g" ./goytyan.service
    cp ./goytyan.service /etc/systemd/system/ # 拷贝服务文件到systemd目录, 这里可能需要sudo权限
    rm -rf ./goytyan.service
    systemctl daemon-reload
    systemctl enable goytyan
    systemctl start goytyan
    systemctl status goytyan
}
# 上面的还没有完成，但应该基本没问题，以后再去做测试吧

function compile_and_restart() {
    cd ..
    go get
    go build -ldflags "-X \"main.compileTime=$(date '+%Y-%m-%d %H:%M:%S')\"" -tags=jsoniter -o ./build/ytyan-go
    echo "Compile done at"  "$(date '+%Y-%m-%d %H:%M:%S')"
    # 询问是否要重启服务
    read -rp "是否要重启服务？[y/n]" isRestart
    if [ "$isRestart" == "y" ]; then
       systemctl daemon-reload
       systemctl restart goytyan
    else
       echo "不重启服务"
    fi
}

compile_and_restart