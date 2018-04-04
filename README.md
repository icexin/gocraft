# GoCraft

A Minecraft like game written in go, just for fun!

![ScreenShot](https://i.imgur.com/vrGRDg1.png)

## Features

- Basic terrain generation
- Add and Remove blocks.
- Move and fly.

## Dependencies

### For go

- go 1.10+

### For glfw

- On macOS, you need Xcode or Command Line Tools for Xcode (`xcode-select --install`) for required headers and libraries.
- On Ubuntu/Debian-like Linux distributions, you need `libgl1-mesa-dev` and `xorg-dev` packages.
- On CentOS/Fedora-like Linux distributions, you need `libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel mesa-libGL-devel libXi-devel` packages.


## Install

`go get github.com/icexin/gocraft`

## Run

`cd $GOPATH/src/github.com/icexin/gocraft && gocraft`

## How to play

- W, S, A, D to move around.
- TAB to toggle flying mode.
- SPACE to jump.
- Left and right click to add/remove block.
- E,R to cycle through the blocks.

## Roadmap

- [ ] Persistent changed blocks
- [ ] Multiplayer support
- [ ] Ambient Occlusion support

## Implementation Details

Many implementations is inspired by https://github.com/fogleman/Craft, thanks for Fogleman's good work!
