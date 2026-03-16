# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make build          # Build with GUI (requires CGO for Fyne/OpenGL on Linux)
make build-linux-static  # Build CLI-only static binary (CGO_ENABLED=0)
make test           # Run all tests
make lint           # Run golangci-lint (requires golangci-lint installed)

go test -v ./internal/udev/...   # Run a specific package's tests
go test -run TestValidateRules ./internal/udev/  # Run a single test
```

The GUI (`fyne.io/fyne/v2`) requires CGO and OpenGL libraries. The default `make build` produces a binary with GUI support on Linux. Use `build-linux-static` for a headless/CI binary.

## Architecture

The codebase has three layers:

**`cmd/goudev/`** — CLI entrypoint (cobra). Subcommands: `list`, `rules`, `install`, `gui`. When invoked with no arguments on Linux, defaults to `gui`. The GUI uses `pkexec` for privilege escalation when not already root; if root, it calls `udev.Install` directly.

**`internal/udev/`** — Rule generation and installation.
- `rules.go`: `Generate(ids, opts)` builds udev rule text; `ValidateRules(content)` checks it against a strict regex before writing. `Options` controls hidraw inclusion, joystick tagging (`ENV{ID_INPUT_JOYSTICK}="1"`), and permission mode (plugdev group vs. `0666`).
- `install.go`: `Install(rulesContent)` validates → checks for `udevadm` in PATH → tests write access → backs up existing file → writes → reloads udev. On reload failure, it removes the new file and restores the backup.
- Rules are written to `/etc/udev/rules.d/85-goudev.rules`.

**`internal/usb/`** — USB device discovery.
- `discovery_linux.go`: Reads `/sys/bus/usb/devices` sysfs tree to enumerate devices, correlate `event*` and `hidraw*` nodes to each USB device.
- `discovery_other.go`: Stub returning `nil, nil` on non-Linux platforms.

**Build tags**: GUI code (`gui.go`) uses `//go:build linux`; the stub (`gui_stub.go`) uses `//go:build !linux`. Same pattern for USB discovery. This allows cross-compilation without CGO for CLI-only builds.

## Key Design Decisions

- **No external USB libraries**: Device discovery reads sysfs directly — no libusb dependency.
- **Validation before install**: `ValidateRules` uses a strict regex so only well-formed goudev-generated rules can be written to `/etc/udev/rules.d/`. This is a safety constraint — don't relax it.
- **Restore on failure**: If `udevadm` reload fails after writing, the install function rolls back to the previous state.
- **Joystick tagging on by default**: `TagAsJoystick: true` is the default in the GUI and CLI (unless `--no-tag-joystick` is passed). This helps rudder pedals and axis-only devices work with SDL/Proton.
