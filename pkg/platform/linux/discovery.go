//go:build linux

// Package platform provides platform-specific USB device access for Linux.
package platform

import (
	"fmt"

	"github.com/kits-io/go-podman-usbip/pkg/device"
)

// DiscoverDevices discovers USB devices on Linux using sysfs.
func DiscoverDevices() ([]*device.Device, error) {
	// TODO: implement using sysfs
	return nil, fmt.Errorf("usb discovery not implemented for linux")
}

// AttachUSBIP attaches a USB device via vhci_hcd.
func AttachUSBIP(serverAddr, busID string) error {
	// TODO: implement vhci_hcd attachment
	return fmt.Errorf("vhci_hcd attachment not implemented")
}
