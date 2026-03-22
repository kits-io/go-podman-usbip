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
// Platform-specific implementations are in platform_*.go files.
func DiscoverDevices() ([]*device.Device, error) {
	// This function is implemented by platform-specific files (platform_darwin.go, platform_linux.go, etc.)
	// which use build tags to only compile on their respective platforms.
	return discoverDevices()
}