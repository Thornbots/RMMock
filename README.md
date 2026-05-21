# RMMock

用于RoboMaster自定义客户端协议的Mock工具，是协议手册中相关部分的第三方实现，提供模拟的图传画面和比赛数据，使自定义客户端开发流程可以完全脱离组委会提供的赛事引擎，因此也无Windows环境要求、无图传等硬件需求

## 通信协议

见[官方的通信协议手册](https://bbs.robomaster.com/wiki/20204847/811363)，当前适配 V1.0.0 的协议

实现上与协议中实现可能有出入的点：

- 协议中服务器 IP 为固定的 192.168.12.1，而本项目不修改设备的网络配置，IP 地址需在系统中设置
- 协议中仅提及每个包有 2byte 的帧编号，也就是上限六万多帧，比较容易到达上限，这里在达到上限后重置为0

## 已有功能

- 采集摄像头或本地视频流，以通信协议中约定的格式发送
- 通过mqtt发送 `GameStatus` 消息，其中除了“当前阶段已过时间”的值为服务器启动时间、当前局号和总局数为1，其余均为0（其他接口将尽快完成编写）

## 环境依赖

本项目主要依赖以下环境：

- OpenCV >= 4.5
- FFmpeg
- Golang >= 1.24 (仅编译需要)

### 使用 Nix (推荐，可选)

如果你安装了 [Nix](https://nixos.org/), 可以使用下面的方式而无需安装依赖.
- **启动项目** `nix run`
- **进入开发环境**: `nix develop`
- **使用 direnv**: 如果安装了 [direnv](https://direnv.net/)，执行 `direnv allow` 即可在进入项目目录时自动配置好所有环境变量和依赖。

本项目仅在 Linux 下测试，理论上 Windows (WSL2) 和 macOS 在配置好环境后也可运行。

编译时加上 tag `opencvstatic` 可以使 OpenCV 变为编译期依赖，但编译出来的二进制文件会非常大，不建议这样操作

## 使用方法

### 传统编译方式

```shell
git clone https://github.com/stydxm/RMMock.git
cd RMMock
go build .
```

### 使用 Nix 编译 (可选)

```shell
nix build
# 编译后的二进制文件位于 ./result/bin/rmmock
```

### 运行

```shell
./RMMock # 或者使用 nix build 后的 ./result/bin/rmmock
```

> 视频画面来源目前为 `0`，即第一个摄像头，编辑 `main.go` 中的 `streamSource` 可以设置为 OpenCV 兼容的所有来源，包括本地文件、URL等

> `scripts/receive_video.py` 是一个 AI 写的脚本，获取到视频流后使用 ffplay 解码播放
> 要注意的是“自定义客户端”在这里是 UDP 服务端，本 Mock 端以及比赛时的“赛事引擎”反而是图传视频流的客户端
