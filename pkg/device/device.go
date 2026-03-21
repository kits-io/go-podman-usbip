// Package device provides USB device discovery and management.
package device

import "fmt"

// Device represents a USB device.
type Device struct {
	BusNum    uint32
	DevNum    uint32
	VendorID  uint16
	ProductID uint16
	Path      string
	BusID     string
	Speed     uint32
	Class     byte
	SubClass  byte
	Protocol  byte
}

// String returns a human-readable description of the device.
func (d *Device) String() string {
	return fmt.Sprintf("%s: %04x:%04x (bus=%d, dev=%d)",
		d.BusID, d.VendorID, d.ProductID, d.BusNum, d.DevNum)
}

// BusPath returns the bus path for USB/IP protocol.
func (d *Device) BusPath() string {
	return fmt.Sprintf("%d-%d", d.BusNum, d.DevNum)
}
