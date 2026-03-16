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

func TestRulesFileNameUsesDeviceName(t *testing.T) {
	got := RulesFileName([]string{"Thrustmaster TA320 Pilot"}, []DeviceID{{Vendor: "044f", Product: "0405"}})
	want := "85-thrustmaster-ta320-pilot.rules"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRulesFileNameFallsBackToVIDPID(t *testing.T) {
	got := RulesFileName(nil, []DeviceID{{Vendor: "131d", Product: "0159"}})
	want := "85-device-131d-0159.rules"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
