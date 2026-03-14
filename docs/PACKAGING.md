# GoUdev — Packaging: AppImage and Installer

## AppImage

### Purpose
Single-file executable that runs on most desktop Linux distros (Mint, Ubuntu, Fedora, etc.) without system-wide install.

### Build options
1. **Minimal:** Package the static Go binary only (no Fyne/GUI deps). User runs e.g. `./goudev-*.AppImage list|add|install`.
2. **With GUI:** If Fyne is used, include Fyne runtime in AppDir and build AppImage per [Fyne packaging](https://docs.fyne.io/started/packaging.html).

### Tools
- [go-appimage](https://github.com/probonopd/go-appimage) (Go implementation of AppImage tooling).
- Or: AppImageKit (appimagetool) + manual AppDir (binary + optional desktop file, icon).

### Root requirement
Writing to `/etc/udev/rules.d/` requires root. Options:
- **CLI:** User runs `sudo ./goudev.AppImage install` or is prompted when choosing “install” from menu.
- **GUI:** “Install rules” triggers pkexec/sudo for a helper that writes file and runs udevadm.

### Build command (example, static binary)
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o goudev -ldflags="-s -w" ./cmd/goudev
# Then use appimagetool or go-appimage to produce .AppImage
```

---

## .deb Installer (Linux Mint / Ubuntu / Debian)

### Purpose
Familiar “install once” experience; binary in PATH (e.g. `/usr/bin/goudev`).

### Control file (minimal)
- **Package:** goudev
- **Depends:** udev
- **Architecture:** amd64 (or arm64 as needed)
- **Description:** Simple udev rule manager for joysticks and HID devices (e.g. X-Plane, games).

### Build options
- **dpkg-deb:** Manual directory layout + `dpkg-deb -b goudev-deb goudev_1.0.0_amd64.deb`.
- **nfpm:** YAML spec + `nfpm pkg` to produce .deb.
- **goreleaser:** Cross-compile Go + nfpm section for .deb (and optionally AppImage).

### Postinst (optional)
- Print one-time message: “Add your user to plugdev: `sudo usermod -aG plugdev $USER`” (or skip and document in README).

---

## CI (recommended)
- On tag/release: build Linux amd64/arm64 static binary, then:
  - Run appimagetool to produce AppImage.
  - Run nfpm/goreleaser to produce .deb.
- Artifacts: `goudev-<ver>.AppImage`, `goudev_<ver>_amd64.deb`.
