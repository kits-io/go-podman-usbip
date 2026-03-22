//go:build darwin

// Package platform provides CGO bindings to libusb for macOS.
package platform

/*
#cgo CFLAGS: -I/usr/local/include/libusb-1.0
#cgo LDFLAGS: -L/usr/local/lib -lusb-1.0

#include <libusb-1.0/libusb.h>
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"sync"
	"unsafe"
)

// Error codes from libusb
const (
	SUCCESS                = 0
	ERROR_IO               = -1
	ERROR_PARAM            = -2
	ERROR_ACCESS           = -3
	ERROR_NOT_FOUND        = -4
	ERROR_BUSY             = -5
	ERROR_TIMEOUT          = -6
	ERROR_OVERFLOW         = -7
	ERROR_PIPE             = -8
	ERROR_INTERRUPTED      = -9
	ERROR_NO_MEM           = -10
	ERROR_NOT_SUPPORTED    = -11
	ERROR_OTHER            = -12
)

// USB speeds
const (
	SPEED_UNKNOWN = 0
	SPEED_LOW     = 1
	SPEED_FULL    = 2
	SPEED_HIGH    = 3
	SPEED_SUPER   = 4
	SPEED_SUPER_PLUS = 5
)

// Context represents a libusb context
type Context struct {
	ctx *C.libusb_context
}

// Device represents a USB device
type Device struct {
	dev *C.libusb_device
	ctx *C.libusb_context
}

// DeviceDescriptor represents a USB device descriptor
type DeviceDescriptor struct {
	Length            uint8
	Type              uint8
	BcdUSB            uint16
	Class             uint8
	SubClass          uint8
	Protocol          uint8
	MaxPacketSize0    uint8
	VendorID          uint16
	ProductID         uint16
	BcdDevice         uint16
	ManufacturerIndex uint8
	ProductIndex      uint8
	SerialNumberIndex uint8
	NumConfigurations uint8
}

// ConfigDescriptor represents a USB configuration descriptor
type ConfigDescriptor struct {
	Length             uint8
	Type               uint8
	TotalLength        uint16
	NumInterfaces      uint8
	ConfigurationValue uint8
	ConfigIndex        uint8
	Attributes         uint8
	MaxPower           uint8
}

// InterfaceDescriptor represents a USB interface descriptor
type InterfaceDescriptor struct {
	Length      uint8
	Type        uint8
	Number      uint8
	Alternate   uint8
	NumEndpoints uint8
	Class       uint8
	SubClass    uint8
	Protocol    uint8
	Index       uint8
}

var (
	context *Context
	once    sync.Once
	initErr error
)

// Init initializes the libusb library
func Init() (*Context, error) {
	once.Do(func() {
		var ctx *C.libusb_context
		ret := C.libusb_init(&ctx)
		if ret < 0 {
			initErr = fmt.Errorf("libusb_init failed: %d", ret)
			return
		}
		context = &Context{ctx: ctx}
	})
	return context, initErr
}

// Exit deinitializes the libusb library
func (c *Context) Exit() {
	if c.ctx != nil {
		C.libusb_exit(c.ctx)
		c.ctx = nil
	}
}

// GetDeviceList retrieves a list of USB devices
func (c *Context) GetDeviceList() ([]*Device, error) {
	var list **C.libusb_device
	var num C.ssize_t

	num = C.libusb_get_device_list(c.ctx, &list)
	if num < 0 {
		return nil, fmt.Errorf("libusb_get_device_list failed: %d", num)
	}

	devices := make([]*Device, num)
	// Convert the C array to Go slice
	deviceSlice := (*[1 << 28]*C.libusb_device)(unsafe.Pointer(list))[:num:num]
	for i := 0; i < int(num); i++ {
		dev := C.libusb_ref_device(deviceSlice[i])
		devices[i] = &Device{dev: dev, ctx: c.ctx}
	}

	C.libusb_free_device_list(list, 1)
	return devices, nil
}

// GetDeviceDescriptor retrieves the device descriptor
func (d *Device) GetDeviceDescriptor() (*DeviceDescriptor, error) {
	var desc C.struct_libusb_device_descriptor
	ret := C.libusb_get_device_descriptor(d.dev, &desc)
	if ret < 0 {
		return nil, fmt.Errorf("libusb_get_device_descriptor failed: %d", ret)
	}

	return &DeviceDescriptor{
		Length:            uint8(desc.bLength),
		Type:              uint8(desc.bDescriptorType),
		BcdUSB:            uint16(desc.bcdUSB),
		Class:             uint8(desc.bDeviceClass),
		SubClass:          uint8(desc.bDeviceSubClass),
		Protocol:          uint8(desc.bDeviceProtocol),
		MaxPacketSize0:    uint8(desc.bMaxPacketSize0),
		VendorID:          uint16(desc.idVendor),
		ProductID:         uint16(desc.idProduct),
		BcdDevice:         uint16(desc.bcdDevice),
		ManufacturerIndex: uint8(desc.iManufacturer),
		ProductIndex:      uint8(desc.iProduct),
		SerialNumberIndex: uint8(desc.iSerialNumber),
		NumConfigurations: uint8(desc.bNumConfigurations),
	}, nil
}

// GetBusNumber returns the bus number
func (d *Device) GetBusNumber() uint8 {
	return uint8(C.libusb_get_bus_number(d.dev))
}

// GetDeviceAddress returns the device address
func (d *Device) GetDeviceAddress() uint8 {
	return uint8(C.libusb_get_device_address(d.dev))
}

// GetSpeed returns the device speed
func (d *Device) GetSpeed() int {
	return int(C.libusb_get_device_speed(d.dev))
}

// GetStringDescriptor retrieves a string descriptor
func (d *Device) GetStringDescriptor(index uint8) (string, error) {
	var handle *C.libusb_device_handle
	ret := C.libusb_open(d.dev, &handle)
	if ret < 0 {
		return "", fmt.Errorf("libusb_open failed: %d", ret)
	}
	defer C.libusb_close(handle)

	var buf [256]byte
	ret = C.libusb_get_string_descriptor_ascii(handle, C.uint8_t(index), (*C.uchar)(unsafe.Pointer(&buf[0])), 256)
	if ret < 0 {
		return "", fmt.Errorf("libusb_get_string_descriptor_ascii failed: %d", ret)
	}

	return string(buf[:ret]), nil
}

// GetConfigDescriptor retrieves the active configuration descriptor
func (d *Device) GetConfigDescriptor() (*ConfigDescriptor, error) {
	var config *C.struct_libusb_config_descriptor
	ret := C.libusb_get_active_config_descriptor(d.dev, &config)
	if ret < 0 {
		return nil, fmt.Errorf("libusb_get_active_config_descriptor failed: %d", ret)
	}
	defer C.libusb_free_config_descriptor(config)

	return &ConfigDescriptor{
		Length:             uint8(config.bLength),
		Type:               uint8(config.bDescriptorType),
		TotalLength:        uint16(config.wTotalLength),
		NumInterfaces:      uint8(config.bNumInterfaces),
		ConfigurationValue: uint8(config.bConfigurationValue),
		ConfigIndex:        uint8(config.iConfiguration),
		Attributes:         uint8(config.bmAttributes),
		MaxPower:           uint8(config.MaxPower),
	}, nil
}

// GetInterfaceDescriptors retrieves all interface descriptors
func (d *Device) GetInterfaceDescriptors() ([]*InterfaceDescriptor, error) {
	var config *C.struct_libusb_config_descriptor
	ret := C.libusb_get_active_config_descriptor(d.dev, &config)
	if ret < 0 {
		return nil, fmt.Errorf("libusb_get_active_config_descriptor failed: %d", ret)
	}
	defer C.libusb_free_config_descriptor(config)

	var interfaces []*InterfaceDescriptor

	// Simplified approach: return minimal interface info
	// TODO: Implement full interface descriptor parsing
	interfaces = append(interfaces, &InterfaceDescriptor{
		Class:    byte(config.bmAttributes), // Placeholder
		SubClass: 0,
		Protocol: 0,
	})

	return interfaces, nil
}

// Unref unreferences a device
func (d *Device) Unref() {
	if d.dev != nil {
		C.libusb_unref_device(d.dev)
		d.dev = nil
	}
}

// ErrorString returns a string representation of an error code
func ErrorString(code int) string {
	return C.GoString(C.libusb_error_name(C.int(code)))
}