package iface

import "errors"

var (
	ErrInvalidAddress        = errors.New("invalid address")
	ErrInvalidMask           = errors.New("invalid mask")
	ErrInvalidBroadcast      = errors.New("invalid broadcast")
	ErrInvalidGateway        = errors.New("invalid gateway")
	ErrInvalidAddressVersion = errors.New("invalid address version")
	ErrAddressSetWhenDHCP    = errors.New("address set when DHCP enabled")
	ErrAddressNotSetStatic   = errors.New("address not set with static config")
	ErrMaskNotSetStatic      = errors.New("mask not set with static config")
	ErrAdressNotLoopback     = errors.New("address must be loopback when config is loopback")
	ErrInterfaceHasErrors    = errors.New("interface has errors")
	ErrUnallocatedInterface  = errors.New("unallocated interface")
	ErrInvalidIfaceData      = errors.New("invalid interface data provided")
	ErrMultipleInterfaces    = errors.New("multiple interfaces in data provided")
)
