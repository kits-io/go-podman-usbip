// Package client implements the USB/IP client.
package client

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/kits-io/go-podman-usbip/pkg/device"
	"github.com/kits-io/go-podman-usbip/pkg/protocol"
)

// Config holds client configuration.
type Config struct {
	ServerHost string
	ServerPort int
}

// Client represents a USB/IP client.
type Client struct {
	config Config
	conn   net.Conn
}

// New creates a new USB/IP client.
func New(config Config) *Client {
	if config.ServerPort == 0 {
		config.ServerPort = protocol.DefaultPort
	}
	return &Client{config: config}
}

// Connect connects to the USB/IP server.
func (c *Client) Connect() error {
	addr := fmt.Sprintf("%s:%d", c.config.ServerHost, c.config.ServerPort)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}
	c.conn = conn
	return nil
}

// Close closes the connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ListDevices lists available devices from the server.
func (c *Client) ListDevices() ([]*device.Device, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	// Send OP_REQ_DEVLIST
	if err := protocol.WriteReqDevListHeader(c.conn); err != nil {
		return nil, fmt.Errorf("failed to send DEVLIST request: %w", err)
	}

	// Read OP_REP_DEVLIST header
	repHeader, err := protocol.ReadRepDevListHeader(c.conn)
	if err != nil {
		return nil, fmt.Errorf("failed to read DEVLIST response: %w", err)
	}

	if repHeader.Status != 0 {
		return nil, fmt.Errorf("server returned error status: %d", repHeader.Status)
	}

	// Read devices
	devices := make([]*device.Device, repHeader.NDevices)
	for i := uint32(0); i < repHeader.NDevices; i++ {
		dev, err := c.readDeviceInfo()
		if err != nil {
			return nil, fmt.Errorf("failed to read device %d: %w", i, err)
		}
		devices[i] = dev
	}

	return devices, nil
}

// ImportDevice imports (attaches) a USB device by bus ID.
func (c *Client) ImportDevice(busID string) (*device.Device, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	// Send OP_REQ_IMPORT
	if err := protocol.WriteReqImportHeader(c.conn, busID); err != nil {
		return nil, fmt.Errorf("failed to send IMPORT request: %w", err)
	}

	// Read OP_REP_IMPORT header
	repHeader, err := protocol.ReadRepImportHeader(c.conn)
	if err != nil {
		return nil, fmt.Errorf("failed to read IMPORT response: %w", err)
	}

	if repHeader.Status != 0 {
		return nil, fmt.Errorf("server returned error status: %d", repHeader.Status)
	}

	// Read device details
	dev, err := c.readImportDeviceDetails()
	if err != nil {
		return nil, fmt.Errorf("failed to read import device details: %w", err)
	}

	return dev, nil
}

func (c *Client) readDeviceInfo() (*device.Device, error) {
	dev := &device.Device{}

	// Read path (256 bytes)
	pathBuf := make([]byte, 256)
	if _, err := io.ReadFull(c.conn, pathBuf); err != nil {
		return nil, err
	}
	dev.Path = protocol.ReadNullTerminatedString(pathBuf)

	// Read bus ID (32 bytes)
	busIDBuf := make([]byte, 32)
	if _, err := io.ReadFull(c.conn, busIDBuf); err != nil {
		return nil, err
	}
	dev.BusID = protocol.ReadNullTerminatedString(busIDBuf)

	// Read device information
	if err := binary.Read(c.conn, binary.BigEndian, &dev.BusNum); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.DevNum); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.Speed); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.VendorID); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.ProductID); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.BcdDevice); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.DeviceClass); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.DeviceSub); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.Protocol); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.ConfigValue); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.NumConfigs); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.NumInterfaces); err != nil {
		return nil, err
	}

	// Read interfaces
	dev.Interfaces = make([]device.Interface, dev.NumInterfaces)
	for i := byte(0); i < dev.NumInterfaces; i++ {
		if err := binary.Read(c.conn, binary.BigEndian, &dev.Interfaces[i].Class); err != nil {
			return nil, err
		}
		if err := binary.Read(c.conn, binary.BigEndian, &dev.Interfaces[i].SubClass); err != nil {
			return nil, err
		}
		if err := binary.Read(c.conn, binary.BigEndian, &dev.Interfaces[i].Protocol); err != nil {
			return nil, err
		}
		var padding byte
		if err := binary.Read(c.conn, binary.BigEndian, &padding); err != nil {
			return nil, err
		}
	}

	return dev, nil
}

func (c *Client) readImportDeviceDetails() (*device.Device, error) {
	dev := &device.Device{}

	// Read path (256 bytes)
	pathBuf := make([]byte, 256)
	if _, err := io.ReadFull(c.conn, pathBuf); err != nil {
		return nil, err
	}
	dev.Path = protocol.ReadNullTerminatedString(pathBuf)

	// Read bus ID (32 bytes)
	busIDBuf := make([]byte, 32)
	if _, err := io.ReadFull(c.conn, busIDBuf); err != nil {
		return nil, err
	}
	dev.BusID = protocol.ReadNullTerminatedString(busIDBuf)

	// Read device information
	if err := binary.Read(c.conn, binary.BigEndian, &dev.BusNum); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.DevNum); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.Speed); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.VendorID); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.ProductID); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.BcdDevice); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.DeviceClass); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.DeviceSub); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.Protocol); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.ConfigValue); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.NumConfigs); err != nil {
		return nil, err
	}
	if err := binary.Read(c.conn, binary.BigEndian, &dev.NumInterfaces); err != nil {
		return nil, err
	}

	return dev, nil
}