# GoUdev — Project Management: Research, Assessment & Analysis

## 1. Executive Summary

**Goal:** A Go application that makes it easy for Linux users (especially those coming from Windows or new to udev) to set udev rules for joysticks, throttles, trim wheels, and other HID/gaming devices so they work in simulators (e.g. X-Plane 12) and games without running as root.

**Target user story (from your example):**  
“lsusb sees my VirtualFly TQ6 and Saitek trim wheel, I added some rules with vendor/device IDs, but nothing shows up in XP12. I’ve restarted several times.”  
**Root cause:** Missing or incorrect udev rules for the **input** (and sometimes **hidraw**) interfaces, so `/dev/input/event*` (and possibly `/dev/hidraw*`) stay root-only and the app can’t access them.

**Deliverables:** Easy-to-use, robust tool + **installer / AppImage** for distribution (e.g. Linux Mint, Ubuntu, Fedora).

---

## 2. Research Summary

### 2.1 Why Devices Show in `lsusb` but Not in Games/Sims

- **lsusb** lists USB devices at the **bus level**; it does not depend on udev permissions for `/dev/input/*` or `/dev/hidraw*`.
- **X-Plane and many games** use the **evdev “input” interface** (`/dev/input/event*`), not the legacy joystick interface (`/dev/input/js*`).
- **Linux** restricts access to input devices (keyboards, mice, etc.). Devices that **look like joysticks** (e.g. two axes + buttons) often get automatic ACLs; **rudder pedals, trim wheels, throttles** often have **no buttons** or **one axis**, so they are **not** auto-identified as joysticks and stay **root-only** unless udev rules fix permissions.

**References:**  
[X-Plane Linux Joystick Permissions](https://developer.x-plane.com/2012/09/linux-joystick-permissions), [Ask Ubuntu – udev USB HID](https://askubuntu.com/questions/15570/configure-udev-to-change-permissions-on-usb-hid-device), [Arch – udev rules not working](https://bbs.archlinux.org/viewtopic.php?id=241612).

### 2.2 Correct udev Rule Types

| Interface | Use case | Example rule (concept) |
|----------|----------|-------------------------|
| **input (event)** | Games/sims using evdev (X-Plane, SDL2, etc.) | `KERNEL=="event*", SUBSYSTEM=="input", ATTRS{idVendor}=="XXXX", ATTRS{idProduct}=="YYYY", MODE="0666"` or `MODE="0660", GROUP="plugdev"` |
| **hidraw** | Raw HID access (some controllers, VR, custom) | `KERNEL=="hidraw*", SUBSYSTEM=="hidraw", ATTRS{idVendor}=="XXXX", ATTRS{idProduct}=="YYYY", MODE="0666"` or same with `GROUP="plugdev"` |
| **usb** | Less precise; can open entire USB device | `SUBSYSTEM=="usb", ATTRS{idVendor}=="XXXX", ATTRS{idProduct}=="YYYY", MODE="0666"` |

- Prefer **input** for joysticks/throttles/trim wheels so X-Plane and similar see them.
- Add **hidraw** only when needed (e.g. device not exposed as evdev or app uses hidraw).
- **GROUP="plugdev"** (with user in `plugdev`) is safer than **MODE="0666"**; support both and default to plugdev when possible.

### 2.3 Finding Vendor / Product IDs

- **lsusb:** `Bus 001 Device 00N: ID xxxx:yyyy` → vendor=xxxx, product=yyyy (hex).
- **lsinput** (input-utils): `sudo lsinput` shows `/dev/input/event*` with vendor/product (e.g. `0x6a3`, `0x763`); use **4-digit zero-padded** in rules (e.g. `06a3`, `0763`).
- **udevadm:**  
  `udevadm info --query=all --attribute-walk --name=/dev/input/eventN`  
  or  
  `udevadm info -a -n /dev/input/eventN`  
  to see attributes and confirm `idVendor`/`idProduct` for the **input** device.
- **Sysfs:** `/sys/bus/usb/devices/*/idVendor`, `idProduct` (and nested interfaces under the same USB device).

### 2.4 Applying Rules Without Reboot

- Reload rules: `udevadm control --reload-rules`
- Re-trigger: `udevadm trigger`
- For already-connected devices, **replug** the USB device after reload+trigger so the new permissions take effect (no full reboot required).

**Reference:** [How to reload udev rules without reboot](https://unix.stackexchange.com/questions/39370/how-to-reload-udev-rules-without-reboot).

### 2.5 Rule File Location and Naming

- **Preferred path:** `/etc/udev/rules.d/` (user/backup friendly; distro-agnostic).
- **Naming:** `NN-name.rules` (e.g. `99-joystick.rules`, `99-goudev.rules`). Lower numbers run first; our app can use a high number (e.g. 85 or 99) to avoid overriding system rules.

### 2.6 Existing Tools and Gaps

- **X-Plane article** references [perse](https://github.com/vranki/perse) — “Permission settings manager GUI for Linux UDev” (24 stars). Useful reference; our tool can be more focused (joystick/HID), better documented, and distributed as AppImage/installer.
- **game-devices-udev** (Codeberg): Curated udev rules per device; manual copy to `/etc/udev/rules.d/`. Our app can **generate** rules from **live USB enumeration** and **install** them, which is easier for non-technical users.
- **Steam/Valve:** steam-devices list; similar idea — we automate rule creation and installation.

---

## 3. Assessment

### 3.1 User Needs

| Need | Priority | Notes |
|------|----------|--------|
| See connected USB HID/input devices with vendor/product ID and name | Must | So user can pick “this is my TQ6 / trim wheel” |
| Generate correct udev rules (input, optionally hidraw) | Must | Event-based rules for sims/games; optional hidraw |
| Install rules to `/etc/udev/rules.d/` (with root) | Must | Single “Apply” or “Install” action |
| Reload udev and optionally prompt to replug | Must | So rules apply without reboot |
| Safe defaults (plugdev vs 0666) and clear explanation | Should | Prefer GROUP="plugdev"; option for MODE="0666" |
| Robustness: bad rule syntax, missing dirs, udev failures | Must | Validate before write; clear errors; no silent failures |
| Installer / AppImage for Linux (Mint, Ubuntu, etc.) | Must | One-click or one-command install / run |

### 3.2 Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Breaking existing udev rules | Write only our file (e.g. `85-goudev.rules`); never overwrite without user confirmation; backup before replace |
| Wrong VID/PID (keyboard/mouse) | Show device name/serial; warn if device looks like keyboard/mouse; optional “expert” mode |
| Distro differences (udev, plugdev) | Prefer standard udev paths and plugdev; document “add user to plugdev” and fallback to 0666 |
| No root / no sudo | Detect and show clear message: “Installation of udev rules requires administrator rights” and show manual instructions |

### 3.3 Non-Goals (Scope)

- Editing arbitrary udev rules (only our generated rules).
- Supporting non-Linux (Windows/macOS) for udev (obviously N/A).
- Device firmware or driver installation.

---

## 4. Technical Analysis and Architecture

### 4.1 Technology Choices

| Area | Choice | Rationale |
|------|--------|-----------|
| Language | **Go** | Single binary, easy cross-compile, good for CLI and small GUIs; fits “installer/AppImage” story. |
| USB enumeration | **Sysfs** (`/sys/bus/usb/devices`) + **/dev/input** (optional: parse `evdev` name) | No CGO; no libusb; works in restricted/container environments; sufficient for VID/PID and device names. |
| Optional: USB names | **usb.ids** (optional embed or read from system) | Human-readable names; can ship a trimmed copy or use system file. |
| GUI | **Fyne** or **CLI + TUI (bubbletea)** | Fyne: pure Go, good for AppImage. TUI: lighter, scriptable. Recommendation: **CLI first**, then optional Fyne GUI. |
| Rule file handling | Plain text write to `/etc/udev/rules.d/` | Simple; backup before overwrite; validate with `udevadm test` or dry-run. |
| Reload udev | `exec` of `udevadm control --reload-rules` and `udevadm trigger` (with root) | Standard and reliable. |

### 4.2 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        GoUdev Application                         │
├─────────────────────────────────────────────────────────────────┤
│  CLI / (optional) GUI                                            │
│    │                                                              │
│    ├──► Device discovery (sysfs + /dev/input)                    │
│    │      → List USB HID/input devices (VID, PID, name, event?)   │
│    │                                                              │
│    ├──► Rule generator                                           │
│    │      → Build udev rules (input [+ hidraw]) for selected IDs  │
│    │      → Option: plugdev vs 0666                               │
│    │                                                              │
│    ├──► Rule installer (needs root)                               │
│    │      → Write to /etc/udev/rules.d/85-goudev.rules            │
│    │      → Backup existing if present                            │
│    │                                                              │
│    └──► Udev reload                                               │
│           → udevadm control --reload-rules && udevadm trigger      │
│           → Instruct user to replug device                        │
└─────────────────────────────────────────────────────────────────┘
```

### 4.3 Data Flow (Typical Run)

1. **Discover:** Scan `/sys/bus/usb/devices/` for USB devices; optionally correlate with `/dev/input/event*` (via symlinks or udev) to show “this USB device is event5”.
2. **Select:** User selects one or more devices (by VID:PID or by name).
3. **Generate:** Build rule lines (input; optionally hidraw); add comment header with app name and date.
4. **Install:** If running as root (or via sudo), write file, backup, then run udevadm; else print rules to stdout and show “run with sudo to install” + copy-paste instructions.
5. **Confirm:** Message: “Rules installed. Please unplug and replug your device(s).”

### 4.4 Dependency and CGO

- **No CGO** for core logic: use only sysfs, file I/O, and shell out to `udevadm`. Keeps static binary and AppImage simple.
- Optional: **gousb** (libusb) only if we need more than sysfs (e.g. detailed product strings from device); can be a build tag so default build stays CGO-free.

---

## 5. Packaging: AppImage and Installer

### 5.1 AppImage

- **Goal:** Single file `goudev-<version>.AppImage` that runs on most desktop Linux (Mint, Ubuntu, Fedora, etc.).
- **Build:** Use [go-appimage](https://github.com/probonopd/go-appimage) or AppImageKit (e.g. `appimagetool`): pack the Go binary + optional Fyne runtime (if GUI) into an AppDir, then produce the AppImage.
- **Execution:** AppImage runs as user; **writing to `/etc/udev/rules.d/` still requires root**. Options:
  - **pkexec** or **sudo** from inside the app (e.g. “Install rules” button runs helper with polkit/sudo), or
  - **CLI:** user runs `sudo ./goudev.AppImage install` or runs the AppImage and is prompted for password when clicking “Install”.
- **Static binary:** Prefer building with `CGO_ENABLED=0` so the AppImage has minimal dependencies.

### 5.2 Installer (.deb for Mint/Ubuntu)

- **Goal:** `.deb` package for Debian/Ubuntu/Mint so users can `sudo dpkg -i goudev_*.deb` or install via a store/script.
- **Content:** Place binary in `/usr/bin/goudev` (or `/usr/local/bin`); optional desktop file and icon.
- **Dependencies:** `Depends:` udev, coreutils (or none beyond udev). No libusb required if we use sysfs.
- **Scripts:** Postinst can optionally add user to `plugdev` (with user consent) or just print a reminder.
- **Build:** Manual `dpkg-deb` or `nfpm` / `goreleaser` for .deb creation.

### 5.3 Documentation for End Users

- **README:** Problem statement (“devices in lsusb but not in X-Plane”), solution (one sentence), install (AppImage vs .deb), usage (list → select → install → replug).
- **In-app:** Short help text or “?” explaining: “We add a udev rule so your user can access this device; you may need to be in the plugdev group; unplug and replug after installing.”

---

## 6. Implementation Plan (Phased)

### Phase 1 — Core (CLI, no GUI)

1. **Device discovery**  
   - Enumerate USB devices from sysfs (vendor, product, optional product string).  
   - Optionally map to `/dev/input/event*` and show which node corresponds to which device.
2. **Rule generation**  
   - Given one or more (vendor, product) pairs, output udev rules (input; optional hidraw) with MODE or GROUP.
3. **Install path**  
   - Write to `/etc/udev/rules.d/85-goudev.rules` (backup if exists), then `udevadm control --reload-rules && udevadm trigger`.
4. **CLI**  
   - Subcommands: `list`, `add VID:PID [VID:PID...]`, `install` (with root check), `rules` (print only, no install).

### Phase 2 — Robustness and UX ✅

5. **Validation**  
   - Validate rule format before write; check udevadm is in PATH before install.
6. **Safety**  
   - Backup before overwrite; on udev reload failure, revert file and restore backup; clear errors for no root / udevadm missing.
7. **Docs**  
   - README: “Why don’t my devices show in X-Plane”, troubleshooting table (replug, plugdev, udev failed).

### Phase 3 — Packaging and Optional GUI ✅

8. **AppImage**  
   - Build script and/or CI to produce `goudev-*.AppImage`. (Done)
9. **.deb**  
   - Package for Mint/Ubuntu (nfpm or goreleaser). (Done)
10. **Optional GUI**  
    - Fyne or TUI wrapper around list → add → install flow for “easy to use” one-click experience. (Done)

---

## 7. Success Criteria

- **Easy:** A user who only knows “lsusb sees my device” can list devices, select theirs, click/run “Install”, and after replug see the device in X-Plane (or similar).
- **Robust:** No silent failures; backup before overwrite; clear messages when root is missing or udev fails.
- **Distributable:** At least one of: AppImage or .deb installer, with brief install and usage instructions.

---

## 8. References

- [X-Plane – Linux Joystick Permissions](https://developer.x-plane.com/2012/09/linux-joystick-permissions)
- [X-Plane – Using Joysticks in X-Plane 11 on Linux](https://www.x-plane.com/kb/using-joysticks-x-plane-11-linux-systems/)
- [Ask Ubuntu – Configure udev for USB HID](https://askubuntu.com/questions/15570/configure-udev-to-change-permissions-on-usb-hid-device)
- [Reload udev rules without reboot](https://unix.stackexchange.com/questions/39370/how-to-reload-udev-rules-without-reboot)
- [game-devices-udev (Codeberg)](https://codeberg.org/fabiscafe/game-devices-udev)
- [perse – Permission settings manager GUI for UDev](https://github.com/vranki/perse)
- [go-appimage](https://github.com/probonopd/go-appimage)
- [Fyne – Packaging for Desktop](https://docs.fyne.io/started/packaging.html)

---

*Document version: 1.2 — Research, assessment and analysis for GoUdev.*
