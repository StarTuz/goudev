# GoUdev — Packaging: DEB, RPM, AppImage

Packaging is driven by [GoReleaser](https://goreleaser.com/) (DEB, RPM, archive) and a custom AppImage step in CI. All three formats are produced on every version tag (`v*`).

## Overview

| Format    | Tool        | Output (example)                    |
|-----------|-------------|-------------------------------------|
| **DEB**   | GoReleaser (nfpm) | `goudev_0.2.0_amd64.deb`        |
| **RPM**   | GoReleaser (nfpm) | `goudev_0.2.0_amd64.rpm`        |
| **AppImage** | appimagetool (in CI) | `goudev-v0.2.0-x86_64.AppImage` |
| **Archive** | GoReleaser | `goudev_0.2.0_linux_amd64.tar.gz`   |

## CI Release (GitHub Actions)

On push of a tag `v*` (e.g. `v0.2.0`):

1. **GoReleaser** builds the Linux amd64 binary (with GUI, CGO enabled), then:
   - Produces **DEB** and **RPM** via nfpm (dependency: `udev`).
   - Produces a **tar.gz** archive.
   - Creates the GitHub Release and uploads these artifacts.

2. **AppImage** step:
   - Uses the binary from `dist/goudev_linux_amd64*/goudev`.
   - Creates an AppDir from `packaging/appimage/` (AppRun, desktop file, binary).
   - Runs [appimagetool](https://github.com/AppImage/appimagetool) and uploads the AppImage to the same release.

Workflow: [.github/workflows/release.yml](../.github/workflows/release.yml).

## Local packaging

### DEB and RPM (snapshot)

Requires [GoReleaser](https://goreleaser.com/install/) (e.g. `go install github.com/goreleaser/goreleaser/v2@latest`).

```bash
make package
```

This runs `goreleaser release --clean --snapshot`. Outputs go to `dist/` (e.g. `dist/goudev_0.2.0-snapshot_amd64.deb`, `dist/goudev_0.2.0-snapshot_amd64.rpm`).

### AppImage (local)

Requires **appimagetool** ([releases](https://github.com/AppImage/appimagetool/releases), e.g. continuous build).

Build the binary first, then:

```bash
make package-appimage VERSION=v0.2.0
```

If you omit `VERSION`, it defaults to `snapshot`. The binary is taken from `./goudev` if present, otherwise from `dist/goudev_linux_amd64*/goudev` (after `make package`). Output: `dist/appimage/goudev-<VERSION>-x86_64.AppImage`.

## Config and AppDir

- **GoReleaser:** [.goreleaser.yml](../.goreleaser.yml) — build (linux/amd64), nfpm (deb + rpm), archive.
- **AppImage AppDir:** [packaging/appimage/](../packaging/appimage/) — `AppRun`, `goudev.desktop`. The binary is copied to `AppDir/usr/bin/goudev`.

## Package details

### DEB / RPM (nfpm)

- **Package name:** goudev  
- **Depends:** udev  
- **Binary:** `/usr/bin/goudev`  
- **Description:** Easy udev rules for joysticks and HID devices on Linux (X-Plane, flight sims, games).

### AppImage

- Single-file executable; no system install. Run e.g. `./goudev-v0.2.0-x86_64.AppImage list` or `./goudev-v0.2.0-x86_64.AppImage gui`.
- Writing to `/etc/udev/rules.d/` still requires root: `sudo ./goudev-*.AppImage install ...` or use the GUI (pkexec/sudo when clicking Install).

## Adding an icon (optional)

To add an icon to the AppImage:

1. Add e.g. `goudev.png` (e.g. 256×256) under `packaging/appimage/` or `AppDir/usr/share/icons/hicolor/256x256/apps/`.
2. In the release workflow, copy the icon into the AppDir before calling appimagetool.
3. The desktop file already references `Icon=goudev`.
