// Package platform provides platform-specific USB device access.
package platform

import (
	"github.com/kits-io/go-podman-usbip/pkg/device"
)

// Discoverer is the interface for USB device discovery.
type Discoverer interface {
	DiscoverDevices() ([]*device.Device, error)
}

// USBDeviceHandle represents an opened USB device for URB transfers.
type USBDeviceHandle interface {
	// SubmitInTransfer performs an IN transfer from the device.
	SubmitInTransfer(endpoint uint8, buffer []byte, timeout int) (actualLength uint32, status int32)
	// SubmitOutTransfer performs an OUT transfer to the device.
	SubmitOutTransfer(endpoint uint8, data []byte, timeout int) (actualLength uint32, status int32)
	// Close closes the device handle.
	Close()
}

//go:generate go run github.com/golang/mock/mockgen -destination=mock/platform_mock.go -package=mock github.com/kits-io/go-podman-usbip/pkg/platform Discoverer

// DiscoverDevices discovers USB devices on the current platform.
// Platform-specific implementations are in platform_*.go files.
func DiscoverDevices() ([]*device.Device, error) {
	// This function is implemented by platform-specific files (platform_darwin.go, platform_linux.go, etc.)
	// which use build tags to only compile on their respective platforms.
	return discoverDevices()
}

// OpenDevice opens a USB device for URB transfers.
// Platform-specific implementations handle the actual device opening.
func OpenDevice(dev *device.Device) (USBDeviceHandle, error) {
	return openDevice(dev)
}