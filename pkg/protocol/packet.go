// Package protocol implements USB/IP protocol encoding and decoding.
package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	// USB/IP port
	DefaultPort = 3240

	// Command codes
	CmdDevList = 0x8005
	CmdImport  = 0x8003

	// Response codes
	RepDevList = 0x0005
	RepImport  = 0x0003
)

// Header represents USB/IP command header.
type Header struct {
	Command uint32
	Status  uint32
}

// DeviceInfo represents USB device information.
type DeviceInfo struct {
	Path         [256]byte
	BusID        [32]byte
	BusNum       uint32
	DevNum       uint32
	Speed        uint32
	VendorID     uint16
	ProductID    uint16
	DeviceClass  byte
	DeviceSub    byte
	Protocol     byte
	ConfigCount  byte
}

// ReadHeader reads a USB/IP header from the connection.
func ReadHeader(r io.Reader) (*Header, error) {
	h := &Header{}
	if err := binary.Read(r, binary.BigEndian, h); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}
	return h, nil
}

// WriteHeader writes a USB/IP header to the connection.
func WriteHeader(w io.Writer, h *Header) error {
	if err := binary.Write(w, binary.BigEndian, h); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}
	return nil
}
