// Package server implements the USB/IP server.
package server

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/kits-io/go-podman-usbip/pkg/device"
	"github.com/kits-io/go-podman-usbip/pkg/platform"
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
	fmt.Printf("New connection from %s\n", conn.RemoteAddr())

	// Read first header to determine command type
	reqHeader, err := protocol.ReadReqDevListHeader(conn)
	if err != nil {
		fmt.Printf("Failed to read header: %v\n", err)
		conn.Close()
		return
	}

	fmt.Printf("Received command: %x\n", reqHeader.Command)

	switch reqHeader.Command {
	case protocol.OP_REQ_DEVLIST:
		s.handleDevList(conn)
		conn.Close()
	case protocol.OP_REQ_IMPORT:
		// Don't close connection for IMPORT - it needs to stay open for URB traffic
		s.handleImport(conn)
	default:
		fmt.Printf("Unknown command: %x\n", reqHeader.Command)
		conn.Close()
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
	// The common header (version, command, status) was already read by handleConnection.
	// Now we only need to read the BusID (32 bytes).
	busIDBuf := make([]byte, 32)
	if _, err := io.ReadFull(conn, busIDBuf); err != nil {
		fmt.Printf("Failed to read BusID: %v\n", err)
		return
	}

	busID := protocol.ReadNullTerminatedString(busIDBuf)
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
		conn.Close()
		return
	}

	fmt.Printf("Device imported: %s\n", targetDev)

	// Handle URB traffic over this connection
	s.handleURBTraffic(conn, targetDev)
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

// handleURBTraffic handles USB/IP URB traffic after device import.
func (s *Server) handleURBTraffic(conn net.Conn, dev *device.Device) {
	defer conn.Close()
	defer fmt.Printf("URB traffic connection closed for device %s\n", dev.BusID)

	fmt.Printf("Starting URB traffic handling for device %s\n", dev.BusID)

	// Get the USB device handle from platform
	usbDev, err := platform.OpenDevice(dev)
	if err != nil {
		fmt.Printf("Failed to open USB device: %v\n", err)
		return
	}
	defer usbDev.Close()

	fmt.Printf("USB device opened successfully: %s\n", dev.BusID)

	// Main URB processing loop
	for {
		// Read URB header
		cmd, err := protocol.ReadURBHeader(conn)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Failed to read URB header: %v\n", err)
			}
			return
		}

		fmt.Printf("Received URB command: 0x%x, seqnum: %d\n", cmd.Command, cmd.SeqNum)

		switch cmd.Command {
		case protocol.USBIP_CMD_SUBMIT:
			s.handleCmdSubmit(conn, cmd, usbDev)
		case protocol.USBIP_CMD_UNLINK:
			s.handleCmdUnlink(conn, cmd)
		default:
			fmt.Printf("Unknown URB command: 0x%x\n", cmd.Command)
			return
		}
	}
}

// handleCmdSubmit handles USBIP_CMD_SUBMIT requests.
func (s *Server) handleCmdSubmit(conn net.Conn, cmd *protocol.URBHeader, usbDev platform.USBDeviceHandle) {
	// Read CMD_SUBMIT specific fields
	transferFlags, err := protocol.ReadUint32(conn)
	if err != nil {
		fmt.Printf("Failed to read transfer_flags: %v\n", err)
		return
	}

	transferBufferLength, err := protocol.ReadUint32(conn)
	if err != nil {
		fmt.Printf("Failed to read transfer_buffer_length: %v\n", err)
		return
	}

	startFrame, err := protocol.ReadUint32(conn)
	if err != nil {
		fmt.Printf("Failed to read start_frame: %v\n", err)
		return
	}

	numberOfPackets, err := protocol.ReadUint32(conn)
	if err != nil {
		fmt.Printf("Failed to read number_of_packets: %v\n", err)
		return
	}

	interval, err := protocol.ReadUint32(conn)
	if err != nil {
		fmt.Printf("Failed to read interval: %v\n", err)
		return
	}
	_ = interval // interval is used for interrupt/isochronous transfers

	// Read setup (8 bytes)
	setup := make([]byte, 8)
	if _, err := io.ReadFull(conn, setup); err != nil {
		fmt.Printf("Failed to read setup: %v\n", err)
		return
	}

	// Read transfer buffer if direction is OUT
	var transferBuffer []byte
	direction := cmd.Direction
	if direction == protocol.USBIP_DIR_OUT && transferBufferLength > 0 {
		transferBuffer = make([]byte, transferBufferLength)
		if _, err := io.ReadFull(conn, transferBuffer); err != nil {
			fmt.Printf("Failed to read transfer buffer: %v\n", err)
			return
		}
	}

	fmt.Printf("CMD_SUBMIT: ep=%d, dir=%d, flags=0x%x, len=%d\n",
		cmd.Endpoint, direction, transferFlags, transferBufferLength)

	// Execute the USB transfer
	var actualLength uint32
	var status int32

	if direction == protocol.USBIP_DIR_OUT {
		// OUT transfer
		actualLength, status = usbDev.SubmitOutTransfer(
			uint8(cmd.Endpoint),
			transferBuffer,
			timeoutFromFlags(transferFlags),
		)
	} else {
		// IN transfer
		transferBuffer = make([]byte, transferBufferLength)
		actualLength, status = usbDev.SubmitInTransfer(
			uint8(cmd.Endpoint),
			transferBuffer,
			timeoutFromFlags(transferFlags),
		)
	}

	// Send RET_SUBMIT response
	ret := &protocol.URBHeader{
		Command: protocol.USBIP_RET_SUBMIT,
		SeqNum:  cmd.SeqNum,
		Devid:   cmd.Devid,
		Direction: cmd.Direction,
		Endpoint: cmd.Endpoint,
	}

	if err := protocol.WriteURBHeader(conn, ret); err != nil {
		fmt.Printf("Failed to write RET_SUBMIT header: %v\n", err)
		return
	}

	// Write RET_SUBMIT specific fields
	fields := []interface{}{
		status,
		actualLength,
		startFrame,
		numberOfPackets,
		int(0), // error_count
		make([]byte, 8), // padding
	}
	for _, f := range fields {
		if err := binaryWrite(conn, f); err != nil {
			fmt.Printf("Failed to write RET_SUBMIT field: %v\n", err)
			return
		}
	}

	// Write transfer buffer if direction is IN
	if direction == protocol.USBIP_DIR_IN && actualLength > 0 {
		if _, err := conn.Write(transferBuffer[:actualLength]); err != nil {
			fmt.Printf("Failed to write transfer buffer: %v\n", err)
			return
		}
	}

	fmt.Printf("RET_SUBMIT: seqnum=%d, status=%d, actual=%d\n", cmd.SeqNum, status, actualLength)
}

// handleCmdUnlink handles USBIP_CMD_UNLINK requests.
func (s *Server) handleCmdUnlink(conn net.Conn, cmd *protocol.URBHeader) {
	// Read unlink_seqnum
	unlinkSeqNum, err := protocol.ReadUint32(conn)
	if err != nil {
		fmt.Printf("Failed to read unlink_seqnum: %v\n", err)
		return
	}

	// Skip padding (24 bytes)
	padding := make([]byte, 24)
	if _, err := io.ReadFull(conn, padding); err != nil {
		fmt.Printf("Failed to read padding: %v\n", err)
		return
	}

	fmt.Printf("CMD_UNLINK: seqnum=%d, unlink_seqnum=%d\n", cmd.SeqNum, unlinkSeqNum)

	// Send RET_UNLINK response
	ret := &protocol.URBHeader{
		Command: protocol.USBIP_RET_UNLINK,
		SeqNum:  cmd.SeqNum,
	}

	if err := protocol.WriteURBHeader(conn, ret); err != nil {
		fmt.Printf("Failed to write RET_UNLINK header: %v\n", err)
		return
	}

	// Write status (0 = success, -ECONNRESET = cancelled)
	status := int32(0) // Simplified: assume unlink succeeded
	if err := binaryWrite(conn, status); err != nil {
		fmt.Printf("Failed to write RET_UNLINK status: %v\n", err)
		return
	}

	// Write padding (24 bytes)
	if err := binaryWrite(conn, make([]byte, 24)); err != nil {
		fmt.Printf("Failed to write RET_UNLINK padding: %v\n", err)
		return
	}

	fmt.Printf("RET_UNLINK: seqnum=%d, status=%d\n", cmd.SeqNum, status)
}

// timeoutFromFlags extracts timeout from URB transfer flags.
func timeoutFromFlags(flags uint32) int {
	// Simplified: return a default timeout
	// In real implementation, would parse URB_NO_FSBR, URB_ZERO_PACKET, etc.
	return 5000 // 5 seconds default
}