//go:build darwin

// Package platform provides platform-specific USB device access for macOS.
package platform

import (
	"fmt"
	"strings"

	"github.com/kits-io/go-podman-usbip/pkg/device"
)

// discoverDevices discovers USB devices on macOS using libusb.
func discoverDevices() ([]*device.Device, error) {
	ctx, err := Init()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize libusb: %w", err)
	}
	defer ctx.Exit()

	// Get device list
	libusbDevs, err := ctx.GetDeviceList()
	if err != nil {
		return nil, fmt.Errorf("failed to get device list: %w", err)
	}

	var devices []*device.Device
	for _, libdev := range libusbDevs {
		dev, err := convertToDevice(ctx, libdev)
		if err != nil {
			// Skip devices that can't be converted
			libdev.Unref()
			continue
		}
		devices = append(devices, dev)
		libdev.Unref()
	}

	return devices, nil
}

// convertToDevice converts a libusb device to our device.Device structure
func convertToDevice(ctx *Context, libdev *Device) (*device.Device, error) {
	// Get device descriptor
	desc, err := libdev.GetDeviceDescriptor()
	if err != nil {
		return nil, err
	}

	// Get bus number and device address
	busNum := uint32(libdev.GetBusNumber())
	devNum := uint32(libdev.GetDeviceAddress())

	// Get device speed
	speed := libdev.GetSpeed()
	speedMap := map[int]uint32{
		SPEED_LOW:         device.SpeedLow,
		SPEED_FULL:        device.SpeedFull,
		SPEED_HIGH:        device.SpeedHigh,
		SPEED_SUPER:       device.SpeedSuper,
		SPEED_SUPER_PLUS:  device.SpeedSuperPlus,
	}
	usbSpeed := speedMap[speed]
	if usbSpeed == 0 {
		usbSpeed = device.SpeedUnknown
	}

	// Get string descriptors
	_ = "" // productName placeholder for future use
	if desc.ProductIndex > 0 {
		_, err = libdev.GetStringDescriptor(desc.ProductIndex)
		_ = err // Ignore error for now
	}

	// Get config descriptor
	configDesc, err := libdev.GetConfigDescriptor()
	if err != nil {
		configDesc = &ConfigDescriptor{}
	}

	// Get interface descriptors
	libusbInterfaces, err := libdev.GetInterfaceDescriptors()
	if err != nil {
		libusbInterfaces = []*InterfaceDescriptor{}
	}

	interfaces := make([]device.Interface, len(libusbInterfaces))
	for i, iface := range libusbInterfaces {
		interfaces[i] = device.Interface{
			Class:    iface.Class,
			SubClass: iface.SubClass,
			Protocol: iface.Protocol,
		}
	}

	// Build device path (simplified for macOS)
	path := fmt.Sprintf("/dev/bus/usb/%03d/%03d", busNum, devNum)

	// Build bus ID
	busID := fmt.Sprintf("%d-%d", busNum, devNum)

	return &device.Device{
		BusNum:         busNum,
		DevNum:         devNum,
		VendorID:       desc.VendorID,
		ProductID:      desc.ProductID,
		BcdDevice:      desc.BcdDevice,
		DeviceClass:    desc.Class,
		DeviceSub:      desc.SubClass,
		Protocol:       desc.Protocol,
		ConfigValue:    configDesc.ConfigurationValue,
		NumConfigs:     desc.NumConfigurations,
		NumInterfaces:  byte(len(interfaces)),
		Path:           path,
		BusID:          busID,
		Speed:          usbSpeed,
		Interfaces:     interfaces,
	}, nil
}

// isUSBSerialDevice checks if a device is a USB serial device
func isUSBSerialDevice(vendorID, productID uint16, class byte) bool {
	// Common USB serial device vendor IDs
	serialVendorIDs := []uint16{
		0x0403, // FTDI
		0x10C4, // Silicon Labs
		0x1A86, // QinHeng Electronics (CH340)
		0x303A, // Espressif
	}

	for _, vid := range serialVendorIDs {
		if vendorID == vid {
			return true
		}
	}

	// Check device class
	if class == 0x02 { // CDC (Communications and CDC Control)
		return true
	}

	return false
}

// formatProductName formats a product name for display
func formatProductName(name string, vendorID, productID uint16) string {
	if name == "" || name == "Unknown" {
		return fmt.Sprintf("%04x:%04x", vendorID, productID)
	}
	return strings.TrimSpace(name)
}