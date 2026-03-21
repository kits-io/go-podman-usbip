//go:build darwin

// Package platform provides platform-specific USB device access for macOS.
package platform

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/kits-io/go-podman-usbip/pkg/device"
)

// discoverDevicesImpl discovers USB devices on macOS using system_profiler.
func discoverDevicesImpl() ([]*device.Device, error) {
	// Try system_profiler first (macOS built-in)
	devices, err := discoverWithSystemProfiler()
	if err == nil && len(devices) > 0 {
		return devices, nil
	}

	// TODO: Fall back to libusb when implemented
	return nil, fmt.Errorf("usb discovery not fully implemented - found %d devices via system_profiler", len(devices))
}

func discoverWithSystemProfiler() ([]*device.Device, error) {
	// Run system_profiler SPUSBDataType
	cmd := exec.Command("system_profiler", "SPUSBDataType", "-xml")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run system_profiler: %w", err)
	}

	// Parse the XML output to extract USB devices
	return parseSystemProfilerOutput(string(output))
}

func parseSystemProfilerOutput(xml string) ([]*device.Device, error) {
	var devices []*device.Device

	// Simple regex-based parsing to extract device info
	// This is a simplified approach - in production, use proper XML parsing
	lines := strings.Split(xml, "\n")
	var currentDevice *device.Device
	busNum := uint32(1)
	devNum := uint32(1)

	// Pattern to match device information
	vendorProductPattern := regexp.MustCompile(`Vendor ID:\s*0x([0-9a-fA-F]+).*Product ID:\s*0x([0-9a-fA-F]+)`)
	pathPattern := regexp.MustCompile(`<key>_name</key>\s*<string>([^<]+)</string>`)

	for _, line := range lines {
		// Match vendor and product IDs
		if matches := vendorProductPattern.FindStringSubmatch(line); len(matches) == 3 {
			vendorID, _ := strconv.ParseUint(matches[1], 16, 16)
			productID, _ := strconv.ParseUint(matches[2], 16, 16)

			if currentDevice != nil && currentDevice.VendorID != 0 {
				devices = append(devices, currentDevice)
			}

			currentDevice = &device.Device{
				VendorID:  uint16(vendorID),
				ProductID: uint16(productID),
				BusNum:    busNum,
				DevNum:    devNum,
				BusID:     fmt.Sprintf("%d-%d", busNum, devNum),
				Path:      "/dev/cu.usbserial", // Simplified
				Speed:     device.SpeedHigh,
			}
			devNum++
		}

		// Match device name
		if currentDevice != nil && currentDevice.Path == "/dev/cu.usbserial" {
			if matches := pathPattern.FindStringSubmatch(line); len(matches) == 2 {
				// Extract device name if possible
				_ = matches[1]
			}
		}
	}

	// Add the last device
	if currentDevice != nil && currentDevice.VendorID != 0 {
		devices = append(devices, currentDevice)
	}

	return devices, nil
}

// Mock implementation for testing
func testMockDevices() []*device.Device {
	return []*device.Device{
		{
			BusNum:       1,
			DevNum:       2,
			VendorID:     0x1a86,
			ProductID:    0x55d4,
			BcdDevice:    0x0100,
			DeviceClass:  0x02,
			DeviceSub:    0x00,
			Protocol:     0x00,
			ConfigValue:  1,
			NumConfigs:   1,
			NumInterfaces: 1,
			Path:         "/dev/cu.usbserial-0001",
			BusID:        "1-2",
			Speed:        device.SpeedFull,
			Interfaces: []device.Interface{
				{Class: 0x02, SubClass: 0x00, Protocol: 0x00},
			},
		},
	}
}

// Mock implementation for testing without hardware
func createMockDevice(vendorID, productID uint16, busNum, devNum uint32, name string) *device.Device {
	return &device.Device{
		BusNum:       busNum,
		DevNum:       devNum,
		VendorID:     vendorID,
		ProductID:    productID,
		BcdDevice:    0x0100,
		DeviceClass:  0x02,
		DeviceSub:    0x00,
		Protocol:     0x00,
		ConfigValue:  1,
		NumConfigs:   1,
		NumInterfaces: 1,
		Path:         fmt.Sprintf("/dev/cu.usbserial-%04x%04x", vendorID, productID),
		BusID:        fmt.Sprintf("%d-%d", busNum, devNum),
		Speed:        device.SpeedFull,
		Interfaces: []device.Interface{
			{Class: 0x02, SubClass: 0x00, Protocol: 0x00},
		},
	}
}

// ListUSBSerialPorts lists all USB serial ports on macOS
func ListUSBSerialPorts() ([]string, error) {
	// Use ioreg to list serial ports
	cmd := exec.Command("ioreg", "-p", "IOUSB", "-w", "0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run ioreg: %w", err)
	}

	// Parse output to extract serial port paths
	var ports []string
	lines := strings.Split(string(output), "\n")
	serialPattern := regexp.MustCompile(`IODialinDevice.*=.*"(.*)"`)

	for _, line := range lines {
		if matches := serialPattern.FindStringSubmatch(line); len(matches) == 2 {
			ports = append(ports, matches[1])
		}
	}

	return ports, nil
}

// Mock for testing: check if running in test mode
func useMock() bool {
	return os.Getenv("MOCK_USB") == "1"
}

// GetMockDevices returns mock USB devices for testing
func GetMockDevices() ([]*device.Device, error) {
	return testMockDevices(), nil
}

// Debug logging
func debugLog(format string, args ...interface{}) {
	if os.Getenv("DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}
