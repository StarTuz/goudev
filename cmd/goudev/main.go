package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/goudev/goudev/internal/udev"
	"github.com/goudev/goudev/internal/usb"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "goudev",
		Short: "Easy udev rules for joysticks and HID devices on Linux",
	}
	root.AddCommand(cmdList(), cmdRules(), cmdInstall(), cmdGUI())
	// Default to GUI on Linux when no subcommand (e.g. double-click or "goudev")
	if len(os.Args) == 1 && runtime.GOOS == "linux" {
		os.Args = append(os.Args, "gui")
	}
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func cmdList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List USB HID/input devices (vendor:product, name, event node)",
		RunE:  runList,
	}
}

func runList(cmd *cobra.Command, args []string) error {
	devices, err := usb.List()
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		fmt.Println("No USB devices found. (Discovery is Linux-only; run on a Linux system with USB devices attached.)")
		return nil
	}
	fmt.Println("Vendor:Product  Product name                    Event device")
	fmt.Println("------------------------------------------------------------------------")
	for _, d := range devices {
		name := d.Product
		if len(name) > 32 {
			name = name[:29] + "..."
		}
		event := d.EventPath
		if event == "" {
			event = "(none)"
		}
		fmt.Printf("%s:%s  %-32s  %s\n", d.VendorID, d.ProductID, name, event)
	}
	return nil
}

var (
	flagHidraw        bool
	flagMode          string
	flagNoTagJoystick bool
)

func sharedRuleFlags(c *cobra.Command) {
	c.Flags().BoolVar(&flagHidraw, "hidraw", false, "Also add hidraw rules for raw HID access (Wine/Proton)")
	c.Flags().BoolVar(&flagNoTagJoystick, "no-tag-joystick", false, "Do not set ID_INPUT_JOYSTICK (default: tag as joystick to help Proton/SDL detect pedals)")
	c.Flags().StringVar(&flagMode, "mode", "plugdev", "Permission: plugdev (default) or 0666")
}

func parseVIDPID(s string) (vendor, product string, err error) {
	vendor, product, ok := parseVIDPIDPair(s)
	if !ok {
		return "", "", fmt.Errorf("invalid VID:PID %q (use lowercase hex, e.g. 06a3:0763)", s)
	}
	vendor, product = udev.NormalizeVIDPID(vendor, product)
	return vendor, product, nil
}

func parseVIDPIDPair(s string) (vendor, product string, ok bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			vendor, product = s[:i], s[i+1:]
			return vendor, product, vendor != "" && product != ""
		}
	}
	return "", "", false
}

func cmdRules() *cobra.Command {
	c := &cobra.Command{
		Use:   "rules <VID:PID> [VID:PID ...]",
		Short: "Print udev rules for the given device(s) without installing",
		Long:  "Example: goudev rules 06a3:0763 231d:0200",
		RunE:  runRules,
	}
	sharedRuleFlags(c)
	return c
}

func runRules(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("provide at least one VID:PID (e.g. 06a3:0763). Use 'goudev list' to see devices")
	}
	opts := ruleOpts()
	ids := make([]udev.DeviceID, 0, len(args))
	for _, a := range args {
		v, p, err := parseVIDPID(a)
		if err != nil {
			return err
		}
		ids = append(ids, udev.DeviceID{Vendor: v, Product: p})
	}
	fmt.Println(udev.Generate(ids, opts))
	return nil
}

func ruleOpts() udev.Options {
	opts := udev.Options{
		IncludeHidraw: flagHidraw,
		TagAsJoystick: !flagNoTagJoystick,
	}
	if flagMode == "0666" {
		opts.Permission = udev.Mode0666
	}
	return opts
}

func cmdInstall() *cobra.Command {
	c := &cobra.Command{
		Use:   "install <VID:PID> [VID:PID ...]",
		Short: "Generate udev rules, write to /etc/udev/rules.d/, and reload udev",
		Long:  "Requires root. Example: sudo goudev install 06a3:0763 231d:0200",
		RunE:  runInstall,
	}
	sharedRuleFlags(c)
	return c
}

func runInstall(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("provide at least one VID:PID (e.g. 06a3:0763). Use 'goudev list' to see devices")
	}
	opts := ruleOpts()
	ids := make([]udev.DeviceID, 0, len(args))
	for _, a := range args {
		v, p, err := parseVIDPID(a)
		if err != nil {
			return err
		}
		ids = append(ids, udev.DeviceID{Vendor: v, Product: p})
	}
	fileName := udev.RulesFileName(resolveDeviceNames(ids), ids)
	rules := udev.Generate(ids, opts)
	res := udev.Install(fileName, rules)
	if res.Err != nil {
		return res.Err
	}
	if res.BackupPath != "" {
		fmt.Println("Backed up existing rules to:", res.BackupPath)
	}
	fmt.Println("Rules installed to", res.RulePath)
	fmt.Println("Udev reloaded. Unplug and replug your device(s) for the new permissions to take effect.")
	if opts.Permission == udev.GroupPlugdev {
		fmt.Println("If access is still denied, add your user to the plugdev group: sudo usermod -aG plugdev $USER")
	}
	return nil
}

func resolveDeviceNames(ids []udev.DeviceID) []string {
	devices, err := usb.List()
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		name := ""
		for _, device := range devices {
			if device.VendorID == id.Vendor && device.ProductID == id.Product {
				name = strings.TrimSpace(device.Product)
				if name != "" {
					break
				}
			}
		}
		names = append(names, name)
	}
	return names
}
