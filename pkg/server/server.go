// Package server implements the USB/IP server.
package server

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/kits-io/go-podman-usbip/pkg/device"
	"github.com/kits-io/go-podman-usbip/pkg/protocol"
)

// Config holds server configuration.
type Config struct {
	Port int
	Host string
}

// Server represents a USB/IP server.
type Server struct {
	config  Config
	devices []*device.Device
	mu      sync.RWMutex
	running bool
	stopCh  chan struct{}
}

// New creates a new USB/IP server.
func New(config Config) *Server {
	if config.Port == 0 {
		config.Port = protocol.DefaultPort
	}
	return &Server{
		config: config,
		stopCh: make(chan struct{}),
	}
}

// Start starts the USB/IP server.
func (s *Server) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}
	s.running = true
	s.mu.Unlock()

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	fmt.Printf("USB/IP server listening on %s\n", addr)

	go s.acceptLoop(listener)
	return nil
}

// Stop stops the USB/IP server.
func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	close(s.stopCh)
}

// SetDevices sets the available USB devices.
func (s *Server) SetDevices(devices []*device.Device) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.devices = devices
}

// GetDevices returns the available USB devices.
func (s *Server) GetDevices() []*device.Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	devices := make([]*device.Device, len(s.devices))
	copy(devices, s.devices)
	return devices
}

func (s *Server) acceptLoop(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return
			default:
				fmt.Printf("Accept error: %v\n", err)
				continue
			}
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	fmt.Printf("New connection from %s\n", conn.RemoteAddr())

	// Read first header to determine command type
	reqHeader, err := protocol.ReadReqDevListHeader(conn)
	if err != nil {
		fmt.Printf("Failed to read header: %v\n", err)
		return
	}

	fmt.Printf("Received command: %x\n", reqHeader.Command)

	switch reqHeader.Command {
	case protocol.OP_REQ_DEVLIST:
		s.handleDevList(conn)
	case protocol.OP_REQ_IMPORT:
		s.handleImport(conn)
	default:
		fmt.Printf("Unknown command: %x\n", reqHeader.Command)
	}
}

func (s *Server) handleDevList(conn net.Conn) {
	s.mu.RLock()
	devices := s.devices
	s.mu.RUnlock()

	// Write header
	nDevices := uint32(len(devices))
	if err := protocol.WriteRepDevListHeader(conn, nDevices); err != nil {
		fmt.Printf("Failed to write DEVLIST header: %v\n", err)
		return
	}

	// Write each device
	for _, dev := range devices {
		if err := s.writeDeviceInfo(conn, dev); err != nil {
			fmt.Printf("Failed to write device info: %v\n", err)
			return
		}
	}

	fmt.Printf("Sent device list: %d devices\n", nDevices)
}

func (s *Server) handleImport(conn net.Conn) {
	req, err := protocol.ReadReqImportHeader(conn)
	if err != nil {
		fmt.Printf("Failed to read IMPORT request: %v\n", err)
		return
	}

	busID := protocol.ReadNullTerminatedString(req.BusID[:])
	fmt.Printf("Import request for bus ID: %s\n", busID)

	s.mu.RLock()
	var targetDev *device.Device
	for _, dev := range s.devices {
		if dev.BusID == busID {
			targetDev = dev
			break
		}
	}
	s.mu.RUnlock()

	if targetDev == nil {
		fmt.Printf("Device not found: %s\n", busID)
		protocol.WriteRepImportHeader(conn, 1) // Error status
		return
	}

	// Write import response header
	if err := protocol.WriteRepImportHeader(conn, 0); err != nil {
		fmt.Printf("Failed to write IMPORT header: %v\n", err)
		return
	}

	// Write device details
	if err := s.writeImportDeviceDetails(conn, targetDev); err != nil {
		fmt.Printf("Failed to write import device details: %v\n", err)
		return
	}

	fmt.Printf("Device imported: %s\n", targetDev)

	// TODO: Handle URB traffic over this connection
}

func (s *Server) writeDeviceInfo(conn net.Conn, dev *device.Device) error {
	// Path (256 bytes)
	pathBuf := make([]byte, 256)
	protocol.WriteNullTerminatedString(pathBuf, dev.Path)
	if _, err := conn.Write(pathBuf); err != nil {
		return err
	}

	// Bus ID (32 bytes)
	busIDBuf := make([]byte, 32)
	protocol.WriteNullTerminatedString(busIDBuf, dev.BusID)
	if _, err := conn.Write(busIDBuf); err != nil {
		return err
	}

	// Device information
	data := []interface{}{
		dev.BusNum,
		dev.DevNum,
		dev.Speed,
		dev.VendorID,
		dev.ProductID,
		dev.BcdDevice,
		dev.DeviceClass,
		dev.DeviceSub,
		dev.Protocol,
		dev.ConfigValue,
		dev.NumConfigs,
		dev.NumInterfaces,
	}

	for _, v := range data {
		if err := binaryWrite(conn, v); err != nil {
			return err
		}
	}

	// Interfaces
	for _, iface := range dev.Interfaces {
		data = []interface{}{
			iface.Class,
			iface.SubClass,
			iface.Protocol,
			byte(0), // padding
		}
		for _, v := range data {
			if err := binaryWrite(conn, v); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Server) writeImportDeviceDetails(conn net.Conn, dev *device.Device) error {
	// Path (256 bytes)
	pathBuf := make([]byte, 256)
	protocol.WriteNullTerminatedString(pathBuf, dev.Path)
	if _, err := conn.Write(pathBuf); err != nil {
		return err
	}

	// Bus ID (32 bytes)
	busIDBuf := make([]byte, 32)
	protocol.WriteNullTerminatedString(busIDBuf, dev.BusID)
	if _, err := conn.Write(busIDBuf); err != nil {
		return err
	}

	// Device information
	data := []interface{}{
		dev.BusNum,
		dev.DevNum,
		dev.Speed,
		dev.VendorID,
		dev.ProductID,
		dev.BcdDevice,
		dev.DeviceClass,
		dev.DeviceSub,
		dev.Protocol,
		dev.ConfigValue,
		dev.NumConfigs,
		dev.NumInterfaces,
	}

	for _, v := range data {
		if err := binaryWrite(conn, v); err != nil {
			return err
		}
	}

	return nil
}

func binaryWrite(conn net.Conn, v interface{}) error {
	return binaryWriteTo(conn, v)
}

func binaryWriteTo(w io.Writer, v interface{}) error {
	return binary.Write(w, binary.BigEndian, v)
}