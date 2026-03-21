// Package device provides USB device discovery and management.
package device

import (
	"fmt"
)

// Device represents a USB device with full configuration information.
type Device struct {
	BusNum       uint32
	DevNum       uint32
	VendorID     uint16
	ProductID    uint16
	BcdDevice    uint16
	DeviceClass  byte
	DeviceSub    byte
	Protocol     byte
	ConfigValue  byte
	NumConfigs   byte
	NumInterfaces byte
	Path         string
	BusID        string
	Speed        uint32
	Interfaces   []Interface
}

// Interface represents a USB interface.
type Interface struct {
	Class    byte
	SubClass byte
	Protocol byte
}

// String returns a human-readable description of the device.
func (d *Device) String() string {
	return fmt.Sprintf("%s: %04x:%04x (bus=%d, dev=%d, speed=%d)",
		d.BusID, d.VendorID, d.ProductID, d.BusNum, d.DevNum, d.Speed)
}

// BusPath returns the bus path for USB/IP protocol.
func (d *Device) BusPath() string {
	return fmt.Sprintf("%d-%d", d.BusNum, d.DevNum)
}

// DeviceID returns the device ID used in USB/IP protocol.
func (d *Device) DeviceID() uint32 {
	return (d.BusNum << 16) | d.DevNum
}

// Speed constants
const (
	SpeedUnknown = 0
	SpeedLow     = 1
	SpeedFull    = 2
	SpeedHigh    = 3
	SpeedWireless = 4
	SpeedSuper = 5
	SpeedSuperPlus = 6
)

// SpeedString returns a string representation of the USB speed.
func SpeedString(speed uint32) string {
	switch speed {
	case SpeedLow:
		return "Low Speed (1.5Mbps)"
	case SpeedFull:
		return "Full Speed (12Mbps)"
	case SpeedHigh:
		return "High Speed (480Mbps)"
	case SpeedSuper:
		return "Super Speed (5Gbps)"
	case SpeedSuperPlus:
		return "Super Speed Plus (10Gbps)"
	default:
		return fmt.Sprintf("Unknown (%d)", speed)
	}
}