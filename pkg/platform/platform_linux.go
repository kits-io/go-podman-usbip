//go:build linux

// Package platform provides platform-specific USB device access for Linux.
package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kits-io/go-podman-usbip/pkg/device"
)

// discoverDevices discovers USB devices on Linux using sysfs.
func discoverDevices() ([]*device.Device, error) {
	// USB devices are in /sys/bus/usb/devices/
	usbPath := "/sys/bus/usb/devices"

	entries, err := os.ReadDir(usbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", usbPath, err)
	}

	var devices []*device.Device
	for _, entry := range entries {
		// Skip non-device entries (like "usb1", "usb2" which are hubs)
		if !strings.Contains(entry.Name(), "-") {
			continue
		}

		dev, err := parseDevice(filepath.Join(usbPath, entry.Name()))
		if err != nil {
			// Skip devices that can't be parsed
			continue
		}

		devices = append(devices, dev)
	}

	return devices, nil
}

// parseDevice parses a USB device from sysfs path.
func parseDevice(path string) (*device.Device, error) {
	// Parse bus ID from directory name (e.g., "1-2.3" -> bus=1, dev=2)
	busNum, devNum, err := parseBusID(filepath.Base(path))
	if err != nil {
		return nil, err
	}

	// Read vendor ID
	vendorID, err := readFileUint16(filepath.Join(path, "idVendor"))
	if err != nil {
		return nil, fmt.Errorf("failed to read vendor ID: %w", err)
	}

	// Read product ID
	productID, err := readFileUint16(filepath.Join(path, "idProduct"))
	if err != nil {
		return nil, fmt.Errorf("failed to read product ID: %w", err)
	}

	// Read device class
	class, err := readFileUint8(filepath.Join(path, "bDeviceClass"))
	if err != nil {
		class = 0
	}

	// Read device subclass
	subClass, err := readFileUint8(filepath.Join(path, "bDeviceSubClass"))
	if err != nil {
		subClass = 0
	}

	// Read device protocol
	protocol, err := readFileUint8(filepath.Join(path, "bDeviceProtocol"))
	if err != nil {
		protocol = 0
	}

	// Read speed
	speed, err := readFileSpeed(filepath.Join(path, "speed"))
	if err != nil {
		speed = device.SpeedUnknown
	}

	// Read bcdDevice
	bcdDevice, err := readFileUint16(filepath.Join(path, "bcdDevice"))
	if err != nil {
		bcdDevice = 0
	}

	// Read configuration value
	configValue, err := readFileUint8(filepath.Join(path, "bConfigurationValue"))
	if err != nil {
		configValue = 1
	}

	// Read number of configurations
	numConfigs, err := readFileUint8(filepath.Join(path, "bNumConfigurations"))
	if err != nil {
		numConfigs = 1
	}

	// Get bus ID
	busID := filepath.Base(path)

	// Build device path
	devPath := fmt.Sprintf("/dev/bus/usb/%03d/%03d", busNum, devNum)

	// Parse interfaces
	interfaces, err := parseInterfaces(path)
	if err != nil {
		interfaces = []device.Interface{}
	}

	return &device.Device{
		BusNum:         busNum,
		DevNum:         devNum,
		VendorID:       vendorID,
		ProductID:      productID,
		BcdDevice:      bcdDevice,
		DeviceClass:    class,
		DeviceSub:      subClass,
		Protocol:       protocol,
		ConfigValue:    configValue,
		NumConfigs:     numConfigs,
		NumInterfaces:  byte(len(interfaces)),
		Path:           devPath,
		BusID:          busID,
		Speed:          speed,
		Interfaces:     interfaces,
	}, nil
}

// parseBusID parses bus ID from directory name (e.g., "1-2.3" -> bus=1, dev=2)
func parseBusID(busID string) (uint32, uint32, error) {
	parts := strings.Split(busID, "-")
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("invalid bus ID format: %s", busID)
	}

	busNum, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid bus number: %s", parts[0])
	}

	// Parse device number (e.g., "2.3" -> dev=2)
	devStr := parts[1]
	if idx := strings.Index(devStr, "."); idx != -1 {
		devStr = devStr[:idx]
	}

	devNum, err := strconv.ParseUint(devStr, 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid device number: %s", devStr)
	}

	return uint32(busNum), uint32(devNum), nil
}

// parseInterfaces parses USB interfaces from sysfs.
func parseInterfaces(path string) ([]device.Interface, error) {
	// Look for interface directories (e.g., "1-2:1.0")
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var interfaces []device.Interface
	for _, entry := range entries {
		// Check if this is an interface directory (contains ":")
		if !strings.Contains(entry.Name(), ":") {
			continue
		}

		ifacePath := filepath.Join(path, entry.Name())
		iface, err := parseInterface(ifacePath)
		if err != nil {
			continue
		}
		interfaces = append(interfaces, iface)
	}

	return interfaces, nil
}

// parseInterface parses a USB interface from sysfs.
func parseInterface(path string) (device.Interface, error) {
	// Read interface class
	class, err := readFileUint8(filepath.Join(path, "bInterfaceClass"))
	if err != nil {
		class = 0
	}

	// Read interface subclass
	subClass, err := readFileUint8(filepath.Join(path, "bInterfaceSubClass"))
	if err != nil {
		subClass = 0
	}

	// Read interface protocol
	protocol, err := readFileUint8(filepath.Join(path, "bInterfaceProtocol"))
	if err != nil {
		protocol = 0
	}

	return device.Interface{
		Class:    class,
		SubClass: subClass,
		Protocol: protocol,
	}, nil
}

// readFileUint8 reads a file and returns its content as uint8.
func readFileUint8(path string) (uint8, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	val, err := strconv.ParseUint(strings.TrimSpace(string(data)), 16, 8)
	if err != nil {
		return 0, err
	}

	return uint8(val), nil
}

// readFileUint16 reads a file and returns its content as uint16.
func readFileUint16(path string) (uint16, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	val, err := strconv.ParseUint(strings.TrimSpace(string(data)), 16, 16)
	if err != nil {
		return 0, err
	}

	return uint16(val), nil
}

// readFileSpeed reads a speed file and returns the speed value.
func readFileSpeed(path string) (uint32, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return device.SpeedUnknown, err
	}

	speedStr := strings.TrimSpace(string(data))
	speedMap := map[string]uint32{
		"1.5":   device.SpeedLow,
		"12":    device.SpeedFull,
		"480":   device.SpeedHigh,
		"5000":  device.SpeedSuper,
		"10000": device.SpeedSuperPlus,
	}

	if speed, ok := speedMap[speedStr]; ok {
		return speed, nil
	}

	// Try to parse as float
	if val, err := strconv.ParseFloat(speedStr, 64); err == nil {
		if val >= 10000 {
			return device.SpeedSuperPlus, nil
		} else if val >= 5000 {
			return device.SpeedSuper, nil
		} else if val >= 480 {
			return device.SpeedHigh, nil
		} else if val >= 12 {
			return device.SpeedFull, nil
		} else if val >= 1.5 {
			return device.SpeedLow, nil
		}
	}

	return device.SpeedUnknown, nil
}

// CheckVHCIHCD checks if the vhci_hcd kernel module is available.
func CheckVHCIHCD() (bool, error) {
	// Check if module exists
	if _, err := os.Stat("/sys/module/vhci_hcd"); err == nil {
		return true, nil
	}

	// Check if module file exists
	if _, err := os.Stat("/lib/modules/$(uname -r)/kernel/drivers/usb/usbip/vhci_hcd.ko"); err == nil {
		return true, nil
	}

	return false, nil
}

// LoadVHCIHCD loads the vhci_hcd kernel module.
func LoadVHCIHCD() error {
	// Use modprobe to load the module
	// Note: This requires root privileges
	return fmt.Errorf("LoadVHCIHCD: requires root privileges and modprobe")
}