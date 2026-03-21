// Package client implements the USB/IP client.
package client

import (
	"fmt"
	"net"

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
func (c *Client) ListDevices() error {
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	// Send OP_REQ_DEVLIST
	header := &protocol.Header{
		Command: protocol.CmdDevList,
	}
	if err := protocol.WriteHeader(c.conn, header); err != nil {
		return err
	}

	// TODO: read response
	return nil
}

// AttachDevice attaches a USB device by bus ID.
func (c *Client) AttachDevice(busID string) error {
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	// TODO: implement device attachment
	return fmt.Errorf("not implemented")
}
