package ifupdown

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strings"
	"sync"
)

type Hooks struct {
	// PreUp of the interface
	PreUp []string `json:"pre_up,omitempty"`
	// PostUp of the interface
	PostUp []string `json:"post_up,omitempty"`
	// PreDown of the interface
	PreDown []string `json:"pre_down,omitempty"`
	// PostDown of the interface
	PostDown []string `json:"post_down,omitempty"`
}

type AddressConfig uint8

const (
	AddressConfigUnset AddressConfig = iota
	AddressConfigLoopback
	AddressConfigDHCP
	AddressConfigStatic
	AddressConfigManual
)

var addressConfigMap = map[AddressConfig]string{
	AddressConfigLoopback: "loopback",
	AddressConfigDHCP:     "dhcp",
	AddressConfigStatic:   "static",
	AddressConfigManual:   "manual",
}

func (ac AddressConfig) String() string {
	return addressConfigMap[ac]
}

type AddressVersion uint8

const (
	AddressVersionNil AddressVersion = iota
	AddressVersion4
	AddressVersion6
)

func (at AddressVersion) String() string {
	switch at {
	case AddressVersion4:
		return "inet"
	case AddressVersion6:
		return "inet6"
	default:
	}
	return ""
}

// NetworkInterface follows the format of ifupdown /etc/network/interfaces
type NetworkInterface struct {
	// Name of the interface.
	Name string `json:"name"`
	// Hotplug determines if hot plugging is allowed.
	Hotplug bool `json:"hotplug,omitempty"`
	// Auto determines if the interface is automatically brought up.
	Auto bool `json:"auto,omitempty"`
	// Address determines the static IP address of the interface.
	Address net.IP `json:"address,omitempty"`

	// Netmask determines the netmask of the interface.
	Netmask net.IPMask `json:"netmask,omitempty"`

	Broadcast net.IP         `json:"broadcast,omitempty"`
	Gateway   net.IP         `json:"gateway,omitempty"`
	Config    AddressConfig  `json:"config,omitempty"`
	Version   AddressVersion `json:"version,omitempty"`

	// DNSServers of the interface.
	DNSServers []net.IP `json:"dns_servers,omitempty"`
	// DNSSearch of the interface.
	DNSSearch []string `json:"dns_search,omitempty"`
	// MACAddress of the interface.
	MACAddress net.HardwareAddr `json:"mac_address,omitempty"`
	// Hooks contains the pre/post up/down hooks.
	Hooks Hooks `json:"hooks,omitempty"`

	dirty     bool
	allocated bool
	errs      []error
	*sync.RWMutex
}

func NewNetworkInterface(name string) *NetworkInterface {
	return &NetworkInterface{
		allocated: false,
		Name:      name,
		Auto:      true,
		dirty:     true,
	}
}

func (iface *NetworkInterface) allocate() {
	if iface.RWMutex == nil {
		iface.RWMutex = &sync.RWMutex{}
	}
	iface.Lock() // gets unlocked in other methods
	iface.dirty = true
	iface.allocated = true

}

func (iface *NetworkInterface) err() error {
	if len(iface.errs) == 0 {
		return nil
	}
	err := ErrInterfaceHasErrors
	for i, e := range iface.errs {
		if err != nil {
			if i == 0 {
				err = fmt.Errorf("%w: %w", err, e)
				continue
			}
			err = fmt.Errorf("%w, %w", err, e)
		}
	}
	return err
}

func (iface *NetworkInterface) Validate() error {
	if !iface.dirty && len(iface.errs) > 0 {
		if err := iface.err(); err != nil {
			return err
		}
	}

	if iface.RWMutex == nil {
		iface.RWMutex = &sync.RWMutex{}
	}

	iface.RLock()
	defer iface.RUnlock()

	iface.errs = iface.errs[:0]

	if iface.allocated != true {
		iface.errs = append(iface.errs, ErrUnallocatedInterface)
		return iface.err()
	}

	if iface.Config == AddressConfigUnset {
		iface.errs = append(iface.errs, fmt.Errorf("address config not set"))
	}

	switch iface.Config {
	case AddressConfigDHCP:
		if iface.Address != nil {
			iface.errs = append(iface.errs, ErrAddressSetWhenDHCP)
		}
	case AddressConfigStatic:
		switch {
		case iface.Address == nil:
			iface.errs = append(iface.errs, ErrAddressNotSetStatic)
		case iface.Netmask == nil && iface.Version == AddressVersion4:
			iface.errs = append(iface.errs, ErrMaskNotSetStatic)
		case iface.Address.IsUnspecified():
			iface.errs = append(iface.errs, ErrInvalidAddress)
		}
	case AddressConfigLoopback:
		if iface.Address != nil && !iface.Address.IsLoopback() {
			iface.errs = append(iface.errs, fmt.Errorf("%w: %v", ErrAdressNotLoopback, iface.Address))
		}
	}

	switch iface.Version {
	case AddressVersion4, AddressVersion6:
		break
	default:
		iface.errs = append(iface.errs, fmt.Errorf(
			"[%s] %w: %v",
			iface.Name, ErrInvalidAddressVersion, iface.Version,
		),
		)
	}

	iface.dirty = false
	return iface.err()
}

func (iface *NetworkInterface) WithAddress(address string) *NetworkInterface {
	iface.allocate()
	_, ipn, err := net.ParseCIDR(address)
	if err == nil {
		iface = iface.WithNetmask(ipn.Mask.Size())
	}
	iface.Address = net.ParseIP(address)
	if iface.Address == nil {
		iface.errs = append(iface.errs, fmt.Errorf("invalid address: %s", address))
		iface.Unlock()
		return iface
	}
	iface.Unlock()
	return iface
}

func (iface *NetworkInterface) WithLoopback() *NetworkInterface {
	iface.allocate()
	iface.Config = AddressConfigLoopback
	iface.Unlock()
	return iface
}

func (iface *NetworkInterface) WithDHCP() *NetworkInterface {
	iface.allocate()
	iface.Config = AddressConfigDHCP
	iface.Unlock()
	return iface
}

func (iface *NetworkInterface) WithStatic() *NetworkInterface {
	iface.allocate()
	iface.Config = AddressConfigStatic
	iface.Unlock()
	return iface
}

func (iface *NetworkInterface) WithManual() *NetworkInterface {
	iface.allocate()
	iface.Config = AddressConfigManual
	iface.Unlock()
	return iface
}

func (iface *NetworkInterface) WithAddressConfig(config AddressConfig) *NetworkInterface {
	iface.allocate()
	iface.Config = config
	iface.Unlock()
	return iface
}

func (iface *NetworkInterface) WithAddressVersion(version AddressVersion) *NetworkInterface {
	iface.allocate()
	iface.Version = version
	iface.Unlock()
	return iface
}

func (iface *NetworkInterface) WithNetmask(mask, bits int) *NetworkInterface {
	iface.allocate()
	var parsedMask net.IPMask
	if iface.Address != nil {
		if iface.Address.To4() != nil || iface.Version == AddressVersion4 {
			parsedMask = net.CIDRMask(mask, bits)
		} else {
			parsedMask = net.CIDRMask(mask, bits)
		}
		if parsedMask == nil {
			iface.errs = append(iface.errs, fmt.Errorf("invalid mask: %d", mask))
			iface.Unlock()
			return iface
		}
	}
	iface.Netmask = parsedMask
	// iface.Netmask = netmask
	iface.Unlock()
	return iface
}

func (iface *NetworkInterface) WithBroadcast(broadcast string) *NetworkInterface {
	iface.allocate()
	iface.Broadcast = net.ParseIP(broadcast)
	if iface.Broadcast == nil {
		iface.errs = append(iface.errs, fmt.Errorf("invalid broadcast: %s", broadcast))
		iface.Unlock()
		return iface
	}
	iface.Unlock()
	return iface
}

func (iface *NetworkInterface) WithGateway(gateway string) *NetworkInterface {
	iface.allocate()
	iface.Gateway = net.ParseIP(gateway)
	if iface.Gateway == nil {
		iface.errs = append(iface.errs, fmt.Errorf("invalid gateway: %s", gateway))
		iface.Unlock()
		return iface
	}
	iface.Unlock()
	return iface
}

func (iface *NetworkInterface) WithConfigMethod(config AddressConfig) *NetworkInterface {
	iface.allocate()
	iface.Config = config
	iface.Unlock()
	return iface
}

func (iface *NetworkInterface) WithVersion(version AddressVersion) *NetworkInterface {
	iface.allocate()
	iface.Version = version
	iface.Unlock()
	return iface
}

func (iface *NetworkInterface) WithDNS(dnsServers []string) *NetworkInterface {
	iface.allocate()
	for _, dns := range dnsServers {
		if nsIP := net.ParseIP(dns); nsIP != nil {
			iface.DNSServers = append(iface.DNSServers, nsIP)
		} else {
			iface.errs = append(iface.errs, fmt.Errorf("invalid dns server: %s", dns))
		}
	}
	iface.Unlock()
	return iface
}

func (iface *NetworkInterface) WithDNSSearch(dnsSearch []string) *NetworkInterface {
	iface.allocate()
	iface.DNSSearch = dnsSearch
	iface.Unlock()
	return iface
}

func (iface *NetworkInterface) WithMACAddress(macAddress string) *NetworkInterface {
	iface.allocate()
	var err error
	iface.MACAddress, err = net.ParseMAC(macAddress)
	if err != nil {
		iface.errs = append(iface.errs, fmt.Errorf("invalid mac address: %s", macAddress))
	}
	iface.Unlock()
	return iface
}

func (iface *NetworkInterface) netMaskString(mask net.IPMask) string {
	if mask == nil {
		return ""
	}
	var m netip.Addr
	if iface.Version == AddressVersion4 {
		m = netip.AddrFrom4([4]byte{mask[0], mask[1], mask[2], mask[3]})
	} else {
		m = netip.AddrFrom16([16]byte{
			mask[0], mask[1], mask[2], mask[3],
			mask[4], mask[5], mask[6], mask[7],
			mask[8], mask[9], mask[10], mask[11],
			mask[12], mask[13], mask[14], mask[15],
		})
	}
	if !m.IsValid() {
		return ""
	}
	return m.String()
}

func (iface *NetworkInterface) String() string {
	if iface.RWMutex == nil {
		iface.RWMutex = new(sync.RWMutex)
	}
	iface.RLock()
	defer iface.RUnlock()
	if !iface.allocated {
		return ""
	}
	if iface.Validate() != nil {
		return ""
	}
	str := pools.Strs.Get()
	defer pools.Strs.Put(str)
	w := func(s string) {
		if len(s) > 0 {
			str.WriteString(s)
		}
	}
	if err := iface.write(w); err != nil && !errors.Is(err, io.EOF) {
		return ""
	}
	return str.String()
}

func (iface *NetworkInterface) Write(p []byte) (int, error) {
	iface.allocate()
	defer iface.Unlock()
	xerox := bufio.NewScanner(bytes.NewReader(p))
	numIfaces := strings.Count(string(p), "iface")
	if numIfaces > 1 {
		return 0, ErrMultipleInterfaces
	}
	for xerox.Scan() {
		normalized := strings.TrimSpace(xerox.Text())
		switch {
		case strings.HasPrefix(normalized, "#"):
			continue
		case strings.HasPrefix(normalized, "auto"):
			iface.Auto = true
			continue
		case strings.HasPrefix(normalized, "allow-hotplug"):
			iface.Hotplug = true
		case strings.HasPrefix(normalized, "iface"):
			for i, fragment := range strings.Fields(normalized) {
				// println(i, fragment)
				switch i {
				case 0:
					if fragment != "iface" {
						return 0, fmt.Errorf("%w: %s", ErrInvalidIfaceData, normalized)
					}
				case 1:
					iface.Name = fragment
				case 2:
					switch fragment {
					case "inet":
						iface.Version = AddressVersion4
						// println("version 4")
					case "inet6":
						iface.Version = AddressVersion6
						// println("version 6")
					default:
					}
				case 3:
					switch fragment {
					case "static":
						iface.Config = AddressConfigStatic
					case "dhcp":
						iface.Config = AddressConfigDHCP
					case "manual":
						iface.Config = AddressConfigManual
					case "loopback":
						iface.Config = AddressConfigLoopback
					default:
						return 0, fmt.Errorf("%w: %s", ErrInvalidIfaceData, normalized)
					}
				default:
					//
				}
			}
		case strings.HasPrefix(normalized, "address"):
			for i, fragment := range strings.Split(normalized, " ") {
				switch i {
				case 0:
					continue
				case 1:
					if strings.Contains(fragment, "/") {
						prfx, _ := netip.ParsePrefix(fragment)
						iface.Address = net.ParseIP(prfx.Addr().String())
						if iface.Address == nil {
							return 0, ErrInvalidIfaceData
						}
						iface.Netmask = net.CIDRMask(prfx.Bits(), 8*len(iface.Address))
						continue
					}
					iface.Address = net.ParseIP(fragment)
					if iface.Address == nil {
						return 0, ErrInvalidIfaceData
					}
				default:
					//
				}
			}
		case strings.HasPrefix(normalized, "netmask"):
			if iface.Netmask != nil || iface.Version != AddressVersion4 {
				continue
			}
			for i, fragment := range strings.Split(normalized, " ") {
				switch i {
				case 0:
					continue
				case 1:
					maskBytes := net.ParseIP(fragment)
					if maskBytes == nil {
						return 0, ErrInvalidIfaceData
					}
					iface.Netmask = net.IPv4Mask(maskBytes[12], maskBytes[13], maskBytes[14], maskBytes[15])
					if iface.Netmask == nil {
						return 0, ErrInvalidIfaceData
					}
				default:
					//
				}
			}
		case strings.HasPrefix(normalized, "gateway"):
			for i, fragment := range strings.Split(normalized, " ") {
				switch i {
				case 0:
					continue
				case 1:
					iface.Gateway = net.ParseIP(fragment)
					if iface.Gateway == nil {
						return 0, ErrInvalidIfaceData
					}
				default:
					//
				}
			}
		case strings.HasPrefix(normalized, "dns-nameservers"):
			for i, fragment := range strings.Split(normalized, " ") {
				switch i {
				case 0:
					continue
				default:
					iface.DNSServers = append(iface.DNSServers, net.ParseIP(fragment))
				}
			}
		case strings.HasPrefix(normalized, "dns-search"):
			for i, fragment := range strings.Split(normalized, " ") {
				switch i {
				case 0:
					continue
				default:
					iface.DNSSearch = append(iface.DNSSearch, fragment)
				}
			}
		case strings.HasPrefix(normalized, "pre-up"):
			hook := strings.TrimPrefix(normalized, "pre-up ")
			if len(hook) == 0 {
				continue
			}
			iface.Hooks.PreUp = append(iface.Hooks.PreUp, hook)
		case strings.HasPrefix(normalized, "post-up"):
			hook := strings.TrimPrefix(normalized, "post-up ")
			if len(hook) == 0 {
				continue
			}
			iface.Hooks.PostUp = append(iface.Hooks.PostUp, hook)
		case strings.HasPrefix(normalized, "pre-down"):
			hook := strings.TrimPrefix(normalized, "pre-down ")
			if len(hook) == 0 {
				continue
			}
			iface.Hooks.PreDown = append(iface.Hooks.PreDown, hook)
		case strings.HasPrefix(normalized, "post-down"):
			hook := strings.TrimPrefix(normalized, "post-down ")
			if len(hook) == 0 {
				continue
			}
			iface.Hooks.PostDown = append(iface.Hooks.PostDown, hook)
		case strings.HasPrefix(normalized, "hwaddress"):
			for i, fragment := range strings.Split(normalized, " ") {
				switch i {
				case 0:
					continue
				case 2:
					var err error
					iface.MACAddress, err = net.ParseMAC(fragment)
					if err != nil {
						return 0, ErrInvalidIfaceData
					}
					continue
				default:
					//
				}

			}
			//
		}
	}

	return len(p), nil
}

func (iface *NetworkInterface) Read(p []byte) (int, error) {
	if err := iface.Validate(); err != nil {
		return 0, err
	}
	var (
		count    = 0
		wChan    = make(chan string)
		errChan  = make(chan error)
		doneChan = make(chan bool)
	)

	w := func(s string) {
		switch {
		case len(s) == 0:
			return
		case len(s) > len(p),
			count+len(s) > len(p):
			errChan <- io.ErrShortBuffer
		default:
			wChan <- s
		}
	}

	go func() {
		errChan <- iface.write(w)
	}()

	for {
		select {
		case <-doneChan:
			return count, nil
		case err := <-errChan:
			if errors.Is(err, io.EOF) || err == nil {
				return count, nil
			}
			if err != nil {
				return count, err
			}
		case s := <-wChan:
			count += copy(p[count:], s)
		}
	}
}

func (iface *NetworkInterface) write(w func(s string)) error {
	if err := iface.Validate(); err != nil {
		return err
	}

	if iface.Auto {
		w("auto ")
		w(iface.Name)
		w("\n")
	}
	if iface.Hotplug {
		w("allow-hotplug ")
		w(iface.Name)
		w("\n")
	}

	w("iface ")
	w(iface.Name)
	w(" ")
	w(iface.Version.String())
	w(" ")
	w(iface.Config.String())
	w("\n")

	if (iface.Address != nil && iface.Netmask != nil && !iface.Address.IsUnspecified()) &&
		(iface.Config == AddressConfigStatic || iface.Config == AddressConfigManual) {
		w("\taddress ")
		w(iface.Address.String())
		w("\n")
		w("\tnetmask ")
		w(iface.netMaskString(iface.Netmask))
		w("\n")
		if iface.Broadcast != nil {
			w("\tbroadcast ")
			w(iface.Broadcast.String())
			w("\n")
		}
		if iface.Gateway != nil {
			w("\tgateway ")
			w(iface.Gateway.String())
			w("\n")
		}
	}

	if len(iface.DNSServers) > 0 {
		w("\tdns-nameservers")
		for _, dns := range iface.DNSServers {
			w(" ")
			w(dns.String())
		}
		w("\n")
	}

	if len(iface.DNSSearch) > 0 {
		w("\tdns-search")
		for _, dns := range iface.DNSSearch {
			w(" ")
			w(dns)
		}
		w("\n")
	}

	if iface.MACAddress != nil {
		w("\thwaddress ether ")
		w(iface.MACAddress.String())
		w("\n")
	}

	if len(iface.Hooks.PreUp) > 0 {
		for _, hook := range iface.Hooks.PreUp {
			w("\tpre-up ")
			w(hook)
			w("\n")
		}
	}

	if len(iface.Hooks.PostUp) > 0 {
		for _, hook := range iface.Hooks.PostUp {
			w("\tpost-up ")
			w(hook)
			w("\n")
		}
	}

	if len(iface.Hooks.PreDown) > 0 {
		for _, hook := range iface.Hooks.PreDown {
			w("\tpre-down ")
			w(hook)
			w("\n")
		}
	}

	if len(iface.Hooks.PostDown) > 0 {
		for _, hook := range iface.Hooks.PostDown {
			w("\tpost-down ")
			w(hook)
			w("\n")
		}
	}
	return io.EOF
}
