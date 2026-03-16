# GoUdev — Requirements & Architecture

## Functional Requirements

### FR1 — Device discovery
- Enumerate USB devices visible to the system (via sysfs: `/sys/bus/usb/devices/`).
- For each device, expose: **Vendor ID**, **Product ID**, **product name** (if available), and optional **/dev/input/eventN** path when applicable.
- Exclude or clearly mark system devices (e.g. root hubs) to avoid accidental rules.

### FR2 — Rule generation
- Generate udev rules for **input** subsystem (evdev) using `ATTRS{idVendor}` and `ATTRS{idProduct}`.
- Optional: generate rules for **hidraw** for the same VID:PID when user requests or when device has no event node.
- Support two permission modes: **GROUP="plugdev"** (recommended) and **MODE="0666"** (fallback).
- Rule file name: `85-goudev.rules` (or configurable); single file managed by the app.

### FR3 — Rule installation
- Write generated rules to `/etc/udev/rules.d/`.
- If target file exists, create a timestamped backup before overwriting.
- After write: run `udevadm control --reload-rules` and `udevadm trigger` (requires root).
- If not root: output rules to stdout and print instructions for manual install.

### FR4 — User guidance
- After install: instruct user to **unplug and replug** the device(s).
- Optional: remind user to add themselves to group `plugdev` when using GROUP mode.

### FR5 — CLI interface
- **list** — List USB HID/input devices with VID, PID, name, and optional event device.
- **add** — Add one or more devices by VID:PID (e.g. `06a3:0763`) to the rule set; output or install.
- **install** — Write rules to `/etc/udev/rules.d/`, reload udev, and show replug instructions.
- **rules** — Print generated rules to stdout without installing.

### FR6 — Optional GUI (Phase 3)
- Show list of detected devices; user selects which to add.
- One-click “Install rules” with clear success/error and “replug device” message.
- [x] **Done.**

---

## Non-Functional Requirements

### NFR1 — Robustness
- Validate rule syntax before writing (e.g. dry-run or udevadm validation).
- Do not overwrite `/etc/udev/rules.d/` without backup.
- Clear error messages when: path not writable, udevadm missing, reload fails.

### NFR2 — Portability
- Linux only; depend only on udev and sysfs (no libusb/CGO in default build).
- Single static binary where possible (CGO_ENABLED=0).

### NFR3 — Packaging
- Provide **AppImage** for generic Linux (run without install).
- Provide **.deb** for Debian/Ubuntu/Linux Mint.
- [x] **Done.**

---

## Architecture (Summary)

- **Discovery:** Read `/sys/bus/usb/devices/*/idVendor`, `idProduct`, `product`; optionally scan `/dev/input/` and match by sysfs path.
- **Rules:** Template-based generation; one line per VID:PID for input (and optionally hidraw).
- **Install:** Write file, backup, exec `udevadm control --reload-rules`, `udevadm trigger`.
- **CLI:** `cobra` or `urfave/cli` for subcommands; minimal dependencies.

---

## Out of Scope

- Editing arbitrary udev rules (only app-generated rule file).
- Windows/macOS support for this tool.
- Driver or firmware installation.
