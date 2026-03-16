//go:build linux

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/goudev/goudev/internal/udev"
	"github.com/goudev/goudev/internal/usb"
	"github.com/spf13/cobra"
)

// dialogTitleForDevices returns a short dialog title using the selected device name(s).
func dialogTitleForDevices(names []string) string {
	if len(names) == 0 {
		return "Rules installed"
	}
	// Trim long names for title
	first := names[0]
	if len(first) > 40 {
		first = first[:37] + "..."
	}
	if len(names) == 1 {
		return "Rules installed for: " + first
	}
	return fmt.Sprintf("Rules installed for %d devices: %s", len(names), first)
}

func cmdGUI() *cobra.Command {
	return &cobra.Command{
		Use:   "gui",
		Short: "Open the graphical interface (device list, select, install)",
		RunE:  runGUI,
	}
}

func runGUI(cmd *cobra.Command, args []string) error {
	a := app.New()
	w := a.NewWindow("GoUdev — udev rules for joysticks & controllers")
	w.Resize(fyne.NewSize(640, 480))

	status := widget.NewLabel("Select your joystick(s), throttle, or game controller. Then click Install rules.")
	status.Wrapping = fyne.TextWrapWord

	var devices []usb.Device
	var selected = make(map[string]bool)

	list := widget.NewList(
		func() int {
			return len(devices)
		},
		func() fyne.CanvasObject {
			return widget.NewCheck("device", nil)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			d := devices[id]
			check := item.(*widget.Check)
			label := fmt.Sprintf("%s:%s  —  %s", d.VendorID, d.ProductID, d.Product)
			if d.Product == "" {
				label = fmt.Sprintf("%s:%s", d.VendorID, d.ProductID)
			}
			check.Text = label
			check.Checked = selected[d.VendorID+":"+d.ProductID]
			check.OnChanged = func(b bool) {
				selected[d.VendorID+":"+d.ProductID] = b
			}
			check.Refresh()
		},
	)

	refresh := func() {
		l, err := usb.List()
		if err != nil {
			status.SetText("Error listing devices: " + err.Error())
			return
		}
		devices = l
		if len(devices) == 0 {
			status.SetText("No USB devices found. Plug in a device and click Refresh.")
		} else {
			status.SetText(fmt.Sprintf("Found %d device(s). Select the ones you want to use in games or X-Plane, then click Install rules.", len(devices)))
		}
		list.Refresh()
	}

	var refreshBtn *widget.Button
	var installBtn *widget.Button

	refreshBtn = widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		refresh()
	})

	installBtn = widget.NewButtonWithIcon("Install rules", theme.ComputerIcon(), func() {
		if len(devices) == 0 {
			dialog.ShowInformation("No devices", "Click Refresh to load devices, then select at least one.", w)
			return
		}
		var ids []udev.DeviceID
		var selectedNames []string
		for _, d := range devices {
			if selected[d.VendorID+":"+d.ProductID] {
				ids = append(ids, udev.DeviceID{Vendor: d.VendorID, Product: d.ProductID})
				name := d.Product
				if name == "" {
					name = d.VendorID + ":" + d.ProductID
				}
				selectedNames = append(selectedNames, name)
			}
		}

		if len(ids) == 0 {
			dialog.ShowInformation("Nothing selected", "Select at least one device (check the box next to it), then click Install rules.", w)
			return
		}
		opts := udev.Options{TagAsJoystick: true, Permission: udev.Mode0666}
		fileName := udev.RulesFileName(selectedNames, ids)
		rules := udev.Generate(ids, opts)

		installBtn.Disable()
		refreshBtn.Disable()
		status.SetText("Installing rules… (enter password if prompted)")

		go func() {
			defer func() {
				installBtn.Enable()
				refreshBtn.Enable()
			}()

			var msg string
			if syscall.Geteuid() == 0 {
				// Already root (e.g. sudo goudev gui): install directly
				res := udev.Install(fileName, rules)
				if res.Err != nil {
					status.SetText("Install failed.")
					dialog.ShowError(res.Err, w)
					return
				}
				if res.BackupPath != "" {
					msg = "Backed up previous rules to: " + res.BackupPath + "\n\n"
				}
				msg += "Configured: " + strings.Join(selectedNames, ", ") + "\n\n"
				msg += "Rules installed to " + res.RulePath + ".\n\nUnplug and replug your device(s) for the new permissions to take effect."
			} else {
				// Need elevation: run ourselves via pkexec
				out, err := runElevatedInstall(ids)
				if err != nil {
					status.SetText("Install failed.")
					dialog.ShowError(err, w)
					return
				}
				msg = "Configured: " + strings.Join(selectedNames, ", ") + "\n\n"
				msg += out + "\n\nUnplug and replug your device(s) for the new permissions to take effect."
			}
			status.SetText("Rules installed successfully.")
			title := dialogTitleForDevices(selectedNames)
			dialog.ShowInformation(title, msg, w)
		}()
	})

	top := container.NewVBox(
		status,
		container.NewHBox(refreshBtn, installBtn),
		widget.NewSeparator(),
		widget.NewLabel("Devices:"),
	)
	content := container.NewPadded(container.NewBorder(top, nil, nil, nil, list))
	w.SetContent(content)
	refresh()
	w.ShowAndRun()
	return nil
}

// runElevatedInstall uses pkexec to run "goudev install" for the given IDs.
func runElevatedInstall(ids []udev.DeviceID) (string, error) {
	exe := os.Getenv("APPIMAGE")
	if exe == "" {
		var err error
		exe, err = os.Executable()
		if err != nil {
			return "", fmt.Errorf("could not find executable: %w", err)
		}
		exe, err = filepath.Abs(exe)
		if err != nil {
			return "", err
		}
	}

	pkArgs := []string{exe, "install"}
	for _, id := range ids {
		pkArgs = append(pkArgs, id.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "pkexec", pkArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := string(out)
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("install failed: %s", errMsg)
	}
	return string(out), nil
}
