//go:build linux

package usb

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const sysfsUSB = "/sys/bus/usb/devices"

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
	seen := make(map[string]bool) // "vid:pid" to avoid duplicates from interfaces

	for _, e := range entries {
		name := e.Name()
		// Skip root hub and pure interface nodes (we want the device node, e.g. "1-2" not "1-2:1.0")
		if name == "usb1" || strings.Contains(name, ":") {
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
		if seen[key] {
			continue
		}
		seen[key] = true

		productName, _ := readTrim(filepath.Join(devPath, "product"))
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
		// input/inputN/events/eventM or input/inputN/eventM
		inputDir := filepath.Join(ifPath, "input")
		inputEntries, err := os.ReadDir(inputDir)
		if err != nil {
			continue
		}
		for _, ie := range inputEntries {
			if !strings.HasPrefix(ie.Name(), "input") {
				continue
			}
			inputBase := filepath.Join(inputDir, ie.Name())
			for _, sub := range []string{"event", "events"} {
				events, _ := os.ReadDir(filepath.Join(inputBase, sub))
				if len(events) > 0 {
					eventName := events[0].Name()
					if strings.HasPrefix(eventName, "event") {
						eventPath = filepath.Join("/dev/input", eventName)
						break
					}
				}
			}
			if eventPath != "" {
				break
			}
		}
		if eventPath != "" {
			break
		}
	}
	// hidraw: under interface we have hidraw/hidrawN
	for _, d := range dirs {
		if !strings.HasPrefix(d.Name(), devName) || !strings.Contains(d.Name(), ":") {
			continue
		}
		hidrawDir := filepath.Join(devPath, d.Name(), "hidraw")
		hr, err := os.ReadDir(hidrawDir)
		if err != nil || len(hr) == 0 {
			continue
		}
		hidrawPath = filepath.Join("/dev", hr[0].Name())
		break
	}
	return eventPath, hidrawPath
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
