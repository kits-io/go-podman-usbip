// Package platform provides platform-specific USB device access.
package platform

import (
	"github.com/kits-io/go-podman-usbip/pkg/device"
)

// Discoverer is the interface for USB device discovery.
type Discoverer interface {
	DiscoverDevices() ([]*device.Device, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination=mock/platform_mock.go -package=mock github.com/kits-io/go-podman-usbip/pkg/platform Discoverer

// DiscoverDevices discovers USB devices on the current platform.
// This function is implemented by platform-specific files.
func DiscoverDevices() ([]*device.Device, error) {
	// The actual implementation is in platform-specific files (darwin.go, linux.go, etc.)
	// which use build tags to only compile on their respective platforms.
	return discoverDevicesImpl()
}

// discoverDevicesImpl is the platform-specific implementation.
// Each platform (darwin, linux, etc.) provides its own implementation.
func discoverDevicesImpl() ([]*device.Device, error) {
	// This function should be overridden by platform-specific files
	// If we reach here, it means no platform-specific implementation was found
	panic("discoverDevicesImpl: no platform-specific implementation found")
}