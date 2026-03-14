//go:build !linux

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func cmdGUI() *cobra.Command {
	return &cobra.Command{
		Use:   "gui",
		Short: "Open the graphical interface (Linux only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(os.Stderr, "The graphical interface is only available on Linux. Use 'goudev list' and 'goudev install VID:PID' from the terminal.")
			os.Exit(1)
			return nil
		},
	}
}
