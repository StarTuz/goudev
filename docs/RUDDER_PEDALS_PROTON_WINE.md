# Rudder Pedals and Proton/Wine Detection — Research

This document summarizes research on why rudder pedals (and similar axis-only devices) often fail to be detected in games running through **Steam Proton** or **Wine**, and what udev/runtime fixes help.

---

## 1. The problem

- **Rudder pedals** (e.g. CH Pro Pedals, Saitek/Logitech, Leo Bodnar) show up in `lsusb` and sometimes in `control.exe` or when running Proton from the command line, but **do not appear** when the same game is launched through the **Steam client**.
- Same hardware, same Proton version — only the launch path (Steam vs. `proton run`) differs.
- Often only “classic” joysticks (with buttons + axes) are detected; pedals and other axis-only or button-less devices are missing.

**References:** [Proton #5126](https://github.com/ValveSoftware/Proton/issues/5126), [Super User: Wine/Proton doesn’t recognize sim pedals](https://superuser.com/questions/1835873/wine-and-proton-does-not-recognize-my-sim-pedals-despite-linux-doing-so).

---

## 2. Root causes

### 2.1 SDL classifies axis-only devices as non-joysticks

- **SDL2** on Linux uses heuristics when udev is unavailable or not trusted (e.g. in Steam’s container/runtime).
- Devices with **several axes and no buttons** (e.g. 3-axis rudder pedals) are classified as **accelerometers**, not joysticks.
- So they are **filtered out** from the joystick list even though they are valid input devices.

**Reference:** [SDL #7500 – CH Pro Pedals detected as accelerometer](https://github.com/libsdl-org/SDL/issues/7500), and [SDL evdev capabilities code](https://github.com/libsdl-org/SDL/blob/main/src/core/linux/SDL_evdev_capabilities.c).

### 2.2 udev not available or different in Steam’s environment

- When the game runs under **Steam**, Proton often runs in an environment where **udev** (or full udev device info) is not available or is different from your desktop.
- SDL then falls back to the heuristic above, so pedals are misclassified.
- Running the same game via `proton run` (or similar) outside the Steam client can expose a different environment (e.g. real udev, different `LD_LIBRARY_PATH`) where devices are detected correctly.

**Reference:** [Proton #5126](https://github.com/ValveSoftware/Proton/issues/5126) (environment comparison, Steam vs. script).

### 2.3 Steam Input / virtual controller

- With **Steam Input** enabled, Steam may claim HID devices and present them as a “virtual” controller; the game then sees the virtual device, not the raw pedals.
- Logs can show lines like: *“try_add_device hidraw … ignoring device 068e/00f2 with virtual Steam controller”*.
- Disabling Steam Input for the game can change what is visible, but behavior is inconsistent (sometimes no joysticks at all).

**Reference:** [Proton #5126](https://github.com/ValveSoftware/Proton/issues/5126).

### 2.4 hidraw vs evdev

- **Wine/Proton** can use either **evdev** (`/dev/input/event*`) or **hidraw** (`/dev/hidraw*`) for joysticks/HID.
- If the game or Wine bus expects **hidraw** and the device is only exposed or permitted on **evdev** (or the opposite), the device may not be seen.
- Proper **udev rules for both** (evdev/input and hidraw) plus **plugdev** (or MODE) ensure the process can open the device regardless of which path is used.

**References:** [WineHQ Hid](https://wiki.winehq.org/Hid), [Proton #5194](https://github.com/ValveSoftware/Proton/issues/5194).

---

## 3. What helps: udev

### 3.1 Permissions (what GoUdev does today)

- **Input (evdev):**  
  `KERNEL=="event*", SUBSYSTEM=="input", ATTRS{idVendor}=="XXXX", ATTRS{idProduct}=="YYYY", MODE="0660", GROUP="plugdev"`  
  (or `MODE="0666"`).
- **Hidraw (optional):**  
  `KERNEL=="hidraw*", SUBSYSTEM=="hidraw", ATTRS{idVendor}=="XXXX", ATTRS{idProduct}=="YYYY", MODE="0660", GROUP="plugdev"`.

These ensure the user (or Steam/Proton process) can open the device. **Necessary but not always sufficient** for Proton/SDL to list the device as a joystick.

### 3.2 Tagging the device as a joystick: `ID_INPUT_JOYSTICK`

- **libinput** and some **SDL** code paths use the udev property **`ID_INPUT_JOYSTICK`** to decide if a device is a joystick.
- If the kernel/udev doesn’t set it (e.g. for axis-only pedals), you can **force** it with a rule:

  `ENV{ID_INPUT_JOYSTICK}="1"`

- Example for a specific pedal device (by VID:PID):

  ```  
  KERNEL=="event*", SUBSYSTEM=="input", ATTRS{idVendor}=="068e", ATTRS{idProduct}=="00f2", MODE="0660", GROUP="plugdev", ENV{ID_INPUT_JOYSTICK}="1"
  ```

- This can help SDL (and thus Proton) treat the device as a joystick instead of an accelerometer when udev info is available.

**References:** [Arch Wiki – Gamepad](https://wiki.archlinux.org/title/Controller), [libinput udev configuration](https://wayland.freedesktop.org/libinput/doc/1.26.0/device-configuration-via-udev.html).

---

## 4. Other workarounds (no code changes)

| Approach | Notes |
|----------|--------|
| **Run outside Steam** | e.g. `proton run /path/to/game.exe` with the same Proton version — often devices appear because udev/env differ. |
| **Older Proton** | Some users report pedals working on Proton 5.0.x and failing on 6.x+ (SDL/heuristic changes). |
| **Steam Input** | Try “Steam Input Disabled” for the game; sometimes more raw devices are visible, sometimes fewer. |
| **Wine registry** | `HKEY_LOCAL_MACHINE\...\winebus`: “Enable SDL” = 0, “Map Controllers” = 0 — can change how Wine enumerates joysticks. |
| **SDL_JOYSTICK_DEVICE** | e.g. `SDL_JOYSTICK_DEVICE=/dev/input/eventX` to force a specific device; not practical for most users. |
| **SDL_JOYSTICK_DISABLE_UDEV** | `SDL_JOYSTICK_DISABLE_UDEV=1` forces SDL to use another discovery path; can help in container/Steam runtime. |
| **steam-devices / udev** | Install distro’s `steam-devices` (or equivalent) so Steam’s udev rules grant access to common controllers. |
| **PROTON_ENABLE_HIDRAW** | In **Proton-GE** (and some builds), this can enable hidraw for specific VID/PID; useful if the game uses hidraw. |

**References:** [Proton #5126](https://github.com/ValveSoftware/Proton/issues/5126), [Virpil + Steam/Proton (mattmoore.io)](https://mattmoore.io/posts/virpil-controls-linux-steam/), [Steam-for-linux #7746](https://github.com/ValveSoftware/steam-for-linux/issues/7746).

---

## 5. Relevance to GoUdev

- **Permissions (input + optional hidraw)** — Already covered: our rules fix “permission denied” so the game/Proton can open the device.
- **Tag as joystick** — Adding **`ENV{ID_INPUT_JOYSTICK}="1"`** to our generated rules can improve the chance that SDL (and thus Proton) lists rudder pedals and similar devices as joysticks, especially when udev is present.
- **Proton/Wine** — We cannot fix Steam’s runtime or SDL’s container fallback inside GoUdev, but we can:
  - Document that **rudder pedals + Proton** often need both **permissions** and **ID_INPUT_JOYSTICK** (and optionally hidraw).
  - Offer an option (e.g. “Tag as joystick for Proton/SDL”) that appends `ENV{ID_INPUT_JOYSTICK}="1"` to the rule.

---

## 6. References (summary)

| Topic | Link |
|-------|------|
| Proton: pedals work via wine/proton, not via Steam client | [Proton #5126](https://github.com/ValveSoftware/Proton/issues/5126) |
| Wine/Proton doesn’t recognize sim pedals | [Super User](https://superuser.com/questions/1835873/wine-and-proton-does-not-recognize-my-sim-pedals-despite-linux-doing-so) |
| SDL: CH Pro Pedals as accelerometer | [SDL #7500](https://github.com/libsdl-org/SDL/issues/7500) |
| Proton: HID devices not recognized | [Proton #5194](https://github.com/ValveSoftware/Proton/issues/5194) |
| Wine HID (hidraw, registry) | [WineHQ Hid](https://wiki.winehq.org/Hid) |
| Steam udev rules (steam-devices) | [ValveSoftware/steam-devices](https://github.com/ValveSoftware/steam-devices) |
| Virpil + Steam/Proton (udev, options) | [mattmoore.io](https://mattmoore.io/posts/virpil-controls-linux-steam/) |
| Arch: controller udev | [Arch Wiki – Gamepad](https://wiki.archlinux.org/title/Controller) |
