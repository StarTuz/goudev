//go:build linux

package usb

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const sysfsUSB = "/sys/bus/usb/devices"

// knownDeviceNames fallback when sysfs product/manufacturer are empty (e.g. some Bluetooth, TrackIR).
var knownDeviceNames = map[string]string{
	"8087:0aa7": "Intel Wireless-AC 3168 Bluetooth",
	"131d:0159": "Natural Point TrackIR",
}

// Device represents a USB device with vendor/product IDs and optional product name.
type Device struct {
	VendorID   string // 4-char hex, e.g. "06a3"
	ProductID  string // 4-char hex, e.g. "0763"
	Product    string // from product file, may be empty
	EventPath  string // e.g. "/dev/input/event5", empty if no evdev
	HidrawPath string // e.g. "/dev/hidraw0", empty if none
	SysPath    string // e.g. "1-2", for reference
}

// List reads USB devices from sysfs and optionally correlates input/hidraw nodes.
func List() ([]Device, error) {
	entries, err := os.ReadDir(sysfsUSB)
	if err != nil {
		return nil, fmt.Errorf("read sysfs usb devices: %w", err)
	}

	var devices []Device

	for _, e := range entries {
		name := e.Name()
		// Skip interface nodes (e.g. "1-2:1.0"); device nodes have no colon (e.g. "1-2").
		// Root hubs (usb1, usb2, …) are filtered below by missing idVendor/idProduct.
		if strings.Contains(name, ":") {
			continue
		}
		devPath := filepath.Join(sysfsUSB, name)
		vendor, _ := readTrim(filepath.Join(devPath, "idVendor"))
		product, _ := readTrim(filepath.Join(devPath, "idProduct"))
		if vendor == "" || product == "" {
			continue
		}
		vendor = normalizeHex(vendor, 4)
		product = normalizeHex(product, 4)
		key := vendor + ":" + product

		productName := deviceDisplayName(devPath, key)
		eventPath, hidrawPath := findInputAndHidraw(devPath, name)

		devices = append(devices, Device{
			VendorID:   vendor,
			ProductID:  product,
			Product:    productName,
			EventPath:  eventPath,
			HidrawPath: hidrawPath,
			SysPath:    name,
		})
	}

	return devices, nil
}

// findInputAndHidraw walks device subtree to find input/event* and hidraw* nodes.
func findInputAndHidraw(devPath, devName string) (eventPath, hidrawPath string) {
	// Interfaces are named like 1-2:1.0, 1-2:1.1 under the same parent.
	dirs, err := os.ReadDir(devPath)
	if err != nil {
		return "", ""
	}

	for _, d := range dirs {
		if !strings.HasPrefix(d.Name(), devName) || !strings.Contains(d.Name(), ":") {
			continue
		}
		ifPath := filepath.Join(devPath, d.Name())

		if eventPath == "" {
			eventPath = findEventNode(ifPath)
		}
		if hidrawPath == "" {
			hidrawPath = findHidrawNode(ifPath)
		}

		if eventPath != "" && hidrawPath != "" {
			break
		}
	}
	return eventPath, hidrawPath
}

// findEventNode looks for input/event* or input/inputN/event* under an interface path.
func findEventNode(ifPath string) string {
	inputDir := filepath.Join(ifPath, "input")
	inputEntries, err := os.ReadDir(inputDir)
	if err != nil {
		return ""
	}

	for _, ie := range inputEntries {
		if !strings.HasPrefix(ie.Name(), "input") {
			continue
		}
		inputBase := filepath.Join(inputDir, ie.Name())
		// Search both directly in inputN/ and in inputN/event/
		for _, sub := range []string{"event", "events", ""} {
			path := inputBase
			if sub != "" {
				path = filepath.Join(inputBase, sub)
			}
			events, _ := os.ReadDir(path)
			for _, ev := range events {
				if strings.HasPrefix(ev.Name(), "event") {
					return filepath.Join("/dev/input", ev.Name())
				}
			}
		}
	}
	return ""
}

// findHidrawNode looks for hidraw/hidrawN under an interface path.
func findHidrawNode(ifPath string) string {
	hidrawDir := filepath.Join(ifPath, "hidraw")
	hr, err := os.ReadDir(hidrawDir)
	if err != nil || len(hr) == 0 {
		return ""
	}
	return filepath.Join("/dev", hr[0].Name())
}

// deviceDisplayName returns product name from sysfs, with manufacturer fallback and known-device table.
func deviceDisplayName(devPath, vidPidKey string) string {
	product, _ := readTrim(filepath.Join(devPath, "product"))
	manufacturer, _ := readTrim(filepath.Join(devPath, "manufacturer"))
	if product != "" && manufacturer != "" {
		return strings.TrimSpace(manufacturer + " " + product)
	}
	if product != "" {
		return product
	}
	if manufacturer != "" {
		return manufacturer
	}
	if name := knownDeviceNames[vidPidKey]; name != "" {
		return name
	}
	return ""
}

func readTrim(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func normalizeHex(s string, width int) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if strings.HasPrefix(s, "0x") {
		s = s[2:]
	}
	for len(s) < width {
		s = "0" + s
	}
	if len(s) > width {
		s = s[len(s)-width:]
	}
	return s
}
