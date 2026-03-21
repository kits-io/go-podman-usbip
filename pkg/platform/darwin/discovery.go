//go:build darwin

// Package platform provides platform-specific USB device access for macOS.
package platform

import (
	"fmt"

	"github.com/kits-io/go-podman-usbip/pkg/device"
)

// DiscoverDevices discovers USB devices on macOS using libusb.
func DiscoverDevices() ([]*device.Device, error) {
	// TODO: implement using libusb/cgo
	// For now, return placeholder
	return nil, fmt.Errorf("usb discovery not implemented for darwin - requires libusb")
}
