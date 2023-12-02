package iface

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestAddressConfig_String(t *testing.T) {
	type test struct {
		name string
		ac   AddressConfig
		want string
	}

	tests := []test{{
		name: "static",
		ac:   AddressConfigStatic,
		want: "static",
	},
		{
			name: "dhcp",
			ac:   AddressConfigDHCP,
			want: "dhcp",
		},
		{
			name: "manual",
			ac:   AddressConfigManual,
			want: "manual",
		},
		{
			name: "loopback",
			ac:   AddressConfigLoopback,
			want: "loopback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ac.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddressType_String(t *testing.T) {
	type test struct {
		name string
		at   AddressVersion
		want string
	}

	tests := []test{
		{
			name: "ipv4",
			at:   AddressVersion4,
			want: "inet",
		},
		{
			name: "ipv6",
			at:   AddressVersion6,
			want: "inet6",
		},
		{
			name: "nil",
			at:   AddressVersionNil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.at.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNetworkInterface_String(t *testing.T) {
	type test struct {
		name       string
		builder    func() *NetworkInterface
		wantErrors []error
		wantString string
	}

	tests := []test{
		{
			name: "static ipv4 #1",
			builder: func() *NetworkInterface {
				return NewNetworkInterface("eth0").
					WithStatic().
					WithAddressVersion(AddressVersion4).
					WithAddress("10.0.0.5").
					WithNetmask(8, 32).
					WithGateway("10.0.0.1")
			},
			wantString: `auto eth0
iface eth0 inet static
	address 10.0.0.5
	netmask 255.0.0.0
	gateway 10.0.0.1
`,
			wantErrors: nil,
		},
		{
			name: "static ipv6 #1",
			builder: func() *NetworkInterface {
				return NewNetworkInterface("eth0").
					WithStatic().
					WithAddressVersion(AddressVersion6).
					WithAddress("2001:db8::1").
					WithNetmask(64, 128).
					WithGateway("2001:db8::2")
			},
			wantString: `auto eth0
iface eth0 inet6 static
	address 2001:db8::1
	netmask ffff:ffff:ffff:ffff::
	gateway 2001:db8::2
`,
			wantErrors: nil,
		},
		{
			name: "static ipv6 #2",
			builder: func() *NetworkInterface {
				return NewNetworkInterface("eth0").
					WithStatic().
					WithAddressVersion(AddressVersion6).
					WithAddress("2001:db8::2").
					WithNetmask(48, 128).
					WithGateway("2001:db8::1")
			},
			wantString: `auto eth0
iface eth0 inet6 static
	address 2001:db8::2
	netmask ffff:ffff:ffff::
	gateway 2001:db8::1
`,
			wantErrors: nil,
		},
		{
			name: "static ipv6 #3",
			builder: func() *NetworkInterface {
				return NewNetworkInterface("eth0").
					WithStatic().
					WithAddressVersion(AddressVersion6).
					WithAddress("fc00:bbbb:bbbb:bb01::31:1927").
					WithNetmask(128, 128).
					WithGateway("fc00:bbbb:bbbb:bb01::1")
			},
			wantString: `auto eth0
iface eth0 inet6 static
	address fc00:bbbb:bbbb:bb01::31:1927
	netmask ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff
	gateway fc00:bbbb:bbbb:bb01::1
`,
			wantErrors: nil,
		},
		{
			name: "dhcp ipv4",
			builder: func() *NetworkInterface {
				return NewNetworkInterface("eth0").
					WithDHCP().
					WithAddressVersion(AddressVersion4)
			},
			wantString: `auto eth0
iface eth0 inet dhcp
`,
			wantErrors: nil,
		},
		{
			name: "dhcp ipv6",
			builder: func() *NetworkInterface {
				return NewNetworkInterface("eth0").
					WithDHCP().
					WithAddressVersion(AddressVersion6)
			},
			wantString: `auto eth0
iface eth0 inet6 dhcp
`,
			wantErrors: nil,
		},
		{
			name: "manual ipv4",
			builder: func() *NetworkInterface {
				return NewNetworkInterface("eth0").
					WithManual().
					WithAddressVersion(AddressVersion4)
			},
			wantString: `auto eth0
iface eth0 inet manual
`,
			wantErrors: nil,
		},
		{
			name: "manual ipv6",
			builder: func() *NetworkInterface {
				return NewNetworkInterface("eth0").WithManual().WithAddressVersion(AddressVersion6)
			},
			wantString: `auto eth0
iface eth0 inet6 manual
`,
			wantErrors: nil,
		},
		{
			name: "loopback ipv4",
			builder: func() *NetworkInterface {
				return NewNetworkInterface("lo").WithLoopback().WithAddressVersion(AddressVersion4)
			},
			wantString: `auto lo
iface lo inet loopback
`,
			wantErrors: nil,
		},
		{
			name: "loopback ipv6",
			builder: func() *NetworkInterface {
				return NewNetworkInterface("lo").
					WithAddressConfig(AddressConfigLoopback).
					WithAddressVersion(AddressVersion6)
			},
			wantString: `auto lo
iface lo inet6 loopback
`,
			wantErrors: nil,
		},
		{
			name: "invalid address config",
			builder: func() *NetworkInterface {
				return NewNetworkInterface("eth0").
					WithAddress("yeeterson")
			},
			wantString: ``,
			wantErrors: []error{fmt.Errorf(": %s", "yeeterson")},
		},
		{
			name: "unallocated interface",
			builder: func() *NetworkInterface {
				return &NetworkInterface{}
			},
			wantString: ``,
			wantErrors: []error{ErrUnallocatedInterface},
		},
		{
			name: "invalid address version",
			builder: func() *NetworkInterface {
				return NewNetworkInterface("eth0").
					WithAddressVersion(3)
			},
			wantString: ``,
			wantErrors: []error{ErrInvalidAddressVersion},
		},
		{
			name: "dirty config",
			builder: func() *NetworkInterface {
				ifa := NewNetworkInterface("eth0").
					WithAddressVersion(3)
				if ifa.Validate() == nil {
					t.Errorf("invalid interface passed validation")
				}
				ifa = ifa.WithAddressVersion(AddressVersion4).
					WithStatic().WithAddress("1.1.1.1").
					WithNetmask(32, 32)
				return ifa
			},
			wantString: `auto eth0
iface eth0 inet static
    address 1.1.1.1
    netmask 255.255.255.255
`,
			wantErrors: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iface := tt.builder()

			if iface == nil {
				t.Fatal("nil interface returned")
			}

			validationErrs := iface.Validate()

			switch {
			case len(tt.wantErrors) < 1 && validationErrs != nil:
				t.Errorf("Validate(): %v", validationErrs)
				t.Logf("%v", iface)
			case len(tt.wantErrors) != 0 && len(iface.errs) == 0:
				t.Errorf("interface error: %v, wanted %+v", iface.errs, tt.wantErrors)
				t.Logf("%v", iface)
			case len(tt.wantErrors) != 0 && len(iface.errs) != 0:
				need := len(tt.wantErrors)
				for _, err := range iface.errs {
					if need == 0 {
						t.Errorf("interface has extra error: %v", err)
						continue
					}

					for _, wantErr := range tt.wantErrors {
						if errors.Is(err, wantErr) {
							need--
							break
						}
					}
				}
			}

			xeroxWant := bufio.NewScanner(strings.NewReader(tt.wantString))
			xeroxGot := bufio.NewScanner(strings.NewReader(iface.String()))
			for xeroxWant.Scan() {
				if !xeroxGot.Scan() {
					t.Errorf("xerox: got %s, want %s", xeroxGot.Text(), xeroxWant.Text())
					continue
				}
				if strings.TrimSpace(xeroxGot.Text()) == "" {
					xeroxGot.Scan()
				}
				if strings.TrimSpace(xeroxGot.Text()) != strings.TrimSpace(xeroxWant.Text()) {
					t.Errorf("xerox: got %s, want %s", xeroxGot.Text(), xeroxWant.Text())
				}
			}

			if len(tt.wantErrors) > 0 {
				return
			}

			b := make([]byte, len(iface.String()))
			n, err := iface.Read(b)
			if err != nil {
				t.Errorf("Read(): = %v", err)
			}
			if n != len(iface.String()) {
				t.Errorf("Read() = %v, want %v", n, len(tt.wantString))
			}
			xeroxWant = bufio.NewScanner(strings.NewReader(tt.wantString))
			xeroxGot = bufio.NewScanner(strings.NewReader(string(b)))
			for xeroxWant.Scan() {
				if !xeroxGot.Scan() {
					t.Errorf("xerox: got %s, want %s", xeroxGot.Text(), xeroxWant.Text())
					continue
				}
				if strings.TrimSpace(xeroxGot.Text()) == "" {
					xeroxGot.Scan()
				}
				if strings.TrimSpace(xeroxGot.Text()) != strings.TrimSpace(xeroxWant.Text()) {
					t.Errorf("xerox: got %s, want %s", xeroxGot.Text(), xeroxWant.Text())
				}
			}

			newIface := &NetworkInterface{}
			if n, err = newIface.Write(b); err != nil {
				t.Errorf("Write(): = %v", err)
			}
			if n != len(b) {
				t.Errorf("Write() = %v, want %v", n, len(tt.wantString))
			}
			newipstr, err := newIface.Address.MarshalText()
			if err != nil {
				t.Errorf("Write() = %v", err)
			}
			ipstr, err := iface.Address.MarshalText()
			if err != nil {
				t.Errorf("Write() = %v", err)
			}
			if !strings.EqualFold(string(newipstr), string(ipstr)) {
				t.Errorf("Write() = %v, want %v", newipstr, ipstr)
			}
			if newIface.Version != iface.Version {
				t.Errorf("Write() = %v, want %v", newIface.Version, iface.Version)
			}
			if newIface.Config != iface.Config {
				t.Errorf("Write() = %v, want %v", newIface.Config, iface.Config)
			}
			if iface.Version == AddressVersion4 && !bytes.Equal(newIface.Netmask, iface.Netmask) {
				t.Errorf("Write() netmask = %v, want %v", newIface.Netmask, iface.Netmask)
			}
			if newIface.Gateway.String() != iface.Gateway.String() {
				t.Errorf("Write() = %v, want %v", newIface.Gateway, iface.Gateway)
			}
			oldIfaceIsValidated := iface.Validate() == nil
			err = newIface.Validate()
			if oldIfaceIsValidated && err != nil {
				t.Errorf("Write() Validate = %v", err)
			}
		})
	}
}
