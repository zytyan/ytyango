#!/usr/bin/env bash
set -e

function compile_and_restart() {
    mkdir build -p
    go get
    go build -ldflags "-X \"main.compileTime=$(date '+%Y-%m-%d %H:%M:%S')\"" -tags=jsoniter -o ./build/ytyan-go
    echo "Compile done at"  "$(date '+%Y-%m-%d %H:%M:%S')"

    # 检查是否传入了 -y 参数
    if [[ "$1" == "-y" ]]; then
        echo "直接重启服务"
        sudo systemctl daemon-reload
        sudo systemctl restart goytyan
    else
        # 询问是否要重启服务
        read -rp "是否要重启服务？[y/n]" isRestart
        if [ "$isRestart" == "y" ]; then
           sudo systemctl daemon-reload
           sudo systemctl restart goytyan
        else
           echo "不重启服务"
        fi
    fi
}

# 调用函数并传递所有参数
compile_and_restart "$@"
