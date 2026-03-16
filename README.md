# GoUdev

**Easy udev rules for joysticks and HID devices on Linux** — so your throttle, trim wheel, or gamepad works in X-Plane, flight sims, and games without running as root.

## The problem

You see your device in `lsusb`, you added a rule with vendor and product IDs, but **nothing shows up in X-Plane (or your game)**. That’s usually because:

- The app uses the **input** interface (`/dev/input/event*`), not the raw USB device.
- Devices like rudder pedals or trim wheels often don’t get automatic “joystick” permissions, so they stay root-only.

GoUdev helps you **discover devices**, **generate the right udev rules** (input, and optionally hidraw), and **install them** in one place. After that, unplug and replug the device — no reboot required.

## Status

**Phase 2 complete:** Validation before install, udevadm check, restore-on-failure, and troubleshooting docs. See [docs/PROJECT_MANAGEMENT.md](docs/PROJECT_MANAGEMENT.md) for full research and roadmap.

## Usage

```bash
# List USB devices (vendor:product and name)
goudev list

# Print udev rules for your device(s) without installing (copy-paste if you prefer manual install)
goudev rules 06a3:0763 231d:0200

# Optional: include hidraw rules, or use MODE="0666" instead of GROUP="plugdev"
goudev rules 06a3:0763 --hidraw --mode=0666

# Install rules (requires root), then unplug and replug your device(s)
sudo goudev install 06a3:0763 231d:0200
```

Use `goudev list` to find your device’s vendor and product IDs (first column). Then run `sudo goudev install VID:PID` for each device you want to use in X-Plane or games.

## Why don’t my devices show in X-Plane (or my game)?

- **lsusb** only shows that the USB device is connected. It doesn’t control who can use the **input** device (`/dev/input/event*`) that games and X-Plane use.
- Linux restricts access to input devices. Keyboards and mice are protected; “joystick-like” devices (with axes and buttons) often get automatic access. **Rudder pedals, trim wheels, throttles** often have no buttons or only one axis, so they are **not** treated as joysticks and stay **root-only** unless you add udev rules.
- GoUdev adds rules so your user can access those input devices. After installing rules, **unplug and replug** the device (a full reboot is not required).

## Troubleshooting

| Issue | What to do |
|-------|------------|
| **“cannot write to /etc/udev/rules.d”** | Install must run as root: `sudo goudev install VID:PID ...` |
| **“udevadm not found in PATH”** | Install udev (or systemd). On Debian/Ubuntu/Mint: `udev` is part of the system; if you’re in a minimal chroot or container, install the `udev` package. |
| **Device still not visible in X-Plane after install** | 1) Unplug and replug the device. 2) If you used the default `plugdev` mode, add your user to the group: `sudo usermod -aG plugdev $USER`, then log out and back in (or reboot). 3) Try `--mode=0666` so any user can access the device: `sudo goudev install --mode=0666 VID:PID`. |
| **“udev reload failed (rules were reverted)”** | udev couldn’t reload. Check `journalctl -u systemd-udevd` or run `sudo udevadm control --reload-rules` and `sudo udevadm trigger` manually to see the error. Your previous rules file was restored. |
| **Wrong device or too many devices in list** | Use the **Vendor:Product** and **Product name** columns to pick the right one (e.g. “Bravo Throttle Quadrant”). Don’t add keyboards, mice, or hubs. |
| **GUI: “install failed” or no password prompt** | Install PolicyKit: `sudo apt install policykit-1` (Debian/Ubuntu/Mint). Or run the app as root: `sudo goudev gui`, then click Install (no prompt). |

## Proton / Wine and rudder pedals

Rudder pedals (and other axis-only devices) are often **not detected** in games run through **Steam Proton** or Wine: SDL can misclassify them as accelerometers, and Steam’s environment may not see udev. By default we add **`ENV{ID_INPUT_JOYSTICK}="1"`** so the device is tagged as a joystick, which can help. For more causes and workarounds (Steam Input, hidraw, older Proton), see **[docs/RUDDER_PEDALS_PROTON_WINE.md](docs/RUDDER_PEDALS_PROTON_WINE.md)**.

## CI/CD

- **CI** (`.github/workflows/ci.yml`): on push/PR to `main` — runs tests and build, plus `golangci-lint`.
- **Release** (`.github/workflows/release.yml`): on tag `v*` — builds Linux binary, **DEB**, **RPM**, and **AppImage**, then creates a GitHub Release with all assets.
- **Developer/CI Locally**: `make test`, `make lint`, `make build`, `make package` (DEB/RPM snapshot), `make package-appimage` (AppImage).

## Docs

| Document | Content |
|----------|--------|
| [docs/PROJECT_MANAGEMENT.md](docs/PROJECT_MANAGEMENT.md) | Research, user needs, risks, architecture, implementation plan |
| [docs/REQUIREMENTS.md](docs/REQUIREMENTS.md) | Functional and non-functional requirements, architecture summary |
| [docs/PACKAGING.md](docs/PACKAGING.md) | DEB, RPM, and AppImage packaging |
| [docs/RUDDER_PEDALS_PROTON_WINE.md](docs/RUDDER_PEDALS_PROTON_WINE.md) | Rudder pedals and Proton/Wine detection (research and workarounds) |

## Planned deliverables

- **CLI:** `list`, `rules`, `install` (done).
- **Robustness:** Backup before overwrite, rule validation before write, udevadm check, restore on reload failure (done).
- **Packaging:** DEB, RPM, and AppImage on tag (Phase 3).

## References

- [X-Plane – Linux joystick permissions](https://developer.x-plane.com/2012/09/linux-joystick-permissions)
- [game-devices-udev](https://codeberg.org/fabiscafe/game-devices-udev) — community udev rules (we automate generation and install)

## License

MIT — see [LICENSE](LICENSE).
