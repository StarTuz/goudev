package udev

import (
	"testing"
)

func TestValidateRules(t *testing.T) {
	valid := Generate([]DeviceID{
		{Vendor: "06a3", Product: "0763"},
	}, Options{})
	if err := ValidateRules(valid); err != nil {
		t.Errorf("expected valid rules to pass: %v", err)
	}

	validWithHidraw := Generate([]DeviceID{
		{Vendor: "231d", Product: "0200"},
	}, Options{IncludeHidraw: true})
	if err := ValidateRules(validWithHidraw); err != nil {
		t.Errorf("expected rules with hidraw to pass: %v", err)
	}

	invalid := "KERNEL==\"event*\", SUBSYSTEM==\"input\", BADLINE"
	if err := ValidateRules(invalid); err == nil {
		t.Error("expected invalid rule to fail validation")
	}
}
