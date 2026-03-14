//go:build !linux

package usb

// Device represents a USB device (Linux only).
type Device struct {
	VendorID   string
	ProductID  string
	Product    string
	EventPath  string
	HidrawPath string
	SysPath    string
}

// List returns an error on non-Linux; discovery is Linux-only.
func List() ([]Device, error) {
	return nil, nil
}
