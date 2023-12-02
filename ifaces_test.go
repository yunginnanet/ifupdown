package iface

import (
	"testing"
)

func TestParse_SimpleValidData(t *testing.T) {
	mp := NewMultiParser()
	data := []byte(`# The loopback network interface
auto lo
iface lo inet loopback

# The primary network interface
allow-hotplug eth0
iface eth0 inet dhcp
`)
	_, err := mp.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	err = mp.Parse()
	if err != nil {
		t.Fatalf("Expected nil error, got: %v", err)
	}

	if len(mp.Interfaces) != 2 {
		t.Fatalf("Expected 2 interfaces, got: %d", len(mp.Interfaces))
	}

	loIface, ok := mp.Interfaces["lo"]
	if !ok || loIface.Name != "lo" {
		t.Fatalf("Expected to find interface 'lo', got: %+v", loIface)
	}

	eth0Iface, ok := mp.Interfaces["eth0"]
	if !ok || eth0Iface.Name != "eth0" {
		t.Fatalf("Expected to find interface 'eth0', got: %+v", eth0Iface)
	}
}

// Add more test functions here to check other aspects and edge cases.
