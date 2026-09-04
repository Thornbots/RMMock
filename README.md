# RMMock

A mock tool for the RoboMaster custom-client protocol — a third-party implementation of the relevant parts of the protocol manual. It supplies simulated video feed and match data so that custom-client development can be done entirely without the organizing committee's competition engine, which also means no Windows requirement and no video-feed hardware.

## Communication protocol

See the [official communication protocol manual](https://bbs.robomaster.com/wiki/20204847/811363). This currently targets protocol V1.0.0.

Points where this implementation may diverge from the protocol:

- The protocol fixes the server IP at 192.168.12.1. This project does not modify the device's network configuration, so the IP address has to be set at the system level.
- The protocol only specifies a 2-byte frame number per packet, i.e. a ceiling of about sixty thousand frames, which is fairly easy to hit; here it wraps back to 0 once it reaches the ceiling.

## What works today

- Capture from a camera or a local video stream and send it in the format the protocol specifies
- Publish `GameStatus` over MQTT, where the "time elapsed in the current stage" carries the server start time, the current round number and total round count are both 1, and everything else is 0 (the other interfaces will be written as soon as possible)

## Requirements

This project mainly depends on:

- OpenCV >= 4.5
- FFmpeg
- Golang >= 1.24 (needed only to build)

### Using Nix (recommended, optional)

If you have [Nix](https://nixos.org/) installed, you can use it instead of installing the dependencies yourself.
- **Run the project**: `nix run`
- **Enter the dev shell**: `nix develop`
- **Use direnv**: with [direnv](https://direnv.net/) installed, `direnv allow` configures every environment variable and dependency automatically when you enter the project directory.

This project is only tested on Linux. In theory Windows (WSL2) and macOS will also run it once the environment is set up.

Building with the `opencvstatic` tag makes OpenCV a compile-time dependency, but the resulting binary is very large, so it is not recommended.

## Usage

### Building the traditional way

```shell
git clone https://github.com/stydxm/RMMock.git
cd RMMock
go build .
```

### Building with Nix (optional)

```shell
nix build
# the built binary lands at ./result/bin/rmmock
```

### Running

```shell
./RMMock # or ./result/bin/rmmock after a nix build
```

> The video source is currently `0`, i.e. the first camera. Edit `streamSource` in `main.go` to use any OpenCV-compatible source, including local files and URLs.

> `scripts/receive_video.py` is an AI-written script that decodes and plays the stream with ffplay once it has been received.
> Note that the "custom client" is the UDP *server* here; this mock — and the "competition engine" during a real match — is the *client* of the video feed.
