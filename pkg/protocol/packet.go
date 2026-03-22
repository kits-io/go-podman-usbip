// Package protocol implements USB/IP protocol encoding and decoding.
// USB/IP Protocol Version: 0x0111 (v1.1.1)
package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

const (
	// DefaultPort is the default USB/IP port
	DefaultPort = 3240

	// Protocol version
	ProtocolVersion = 0x0111

	// OP_REQ_DEVLIST: Retrieve the list of exported USB devices
	OP_REQ_DEVLIST = 0x8005

	// OP_REP_DEVLIST: Reply with the list of exported USB devices
	OP_REP_DEVLIST = 0x0005

	// OP_REQ_IMPORT: Request to import (attach) a remote USB device
	OP_REQ_IMPORT = 0x8003

	// OP_REP_IMPORT: Reply to import (attach) a remote USB device
	OP_REP_IMPORT = 0x0003

	// USBIP_CMD_SUBMIT: Submit an URB
	USBIP_CMD_SUBMIT = 0x00000001

	// USBIP_RET_SUBMIT: Reply for submitting an URB
	USBIP_RET_SUBMIT = 0x00000003

	// USBIP_CMD_UNLINK: Unlink an URB
	USBIP_CMD_UNLINK = 0x00000002

	// USBIP_RET_UNLINK: Reply for URB unlink
	USBIP_RET_UNLINK = 0x00000004

	// USBIP_DIR_OUT: Direction OUT
	USBIP_DIR_OUT = 0x00000000

	// USBIP_DIR_IN: Direction IN
	USBIP_DIR_IN = 0x00000001
)

// ReqDevListHeader represents OP_REQ_DEVLIST header.
type ReqDevListHeader struct {
	Version uint16
	Command uint16
	Status  uint32
}

// RepDevListHeader represents OP_REP_DEVLIST header.
type RepDevListHeader struct {
	Version uint16
	Command uint16
	Status  uint32
	NDevices uint32
}

// ReqImportHeader represents OP_REQ_IMPORT header.
type ReqImportHeader struct {
	Version uint16
	Command uint16
	Status  uint32
	BusID   [32]byte
}

// RepImportHeader represents OP_REP_IMPORT header.
type RepImportHeader struct {
	Version uint16
	Command uint16
	Status  uint32
}

// USBIPHeaderBasic represents the basic USB/IP header.
type USBIPHeaderBasic struct {
	Command uint32
	SeqNum  uint32
	Devid   uint32
	Dir     uint32
	EP      uint32
}

// USBIPCmdSubmit represents USBIP_CMD_SUBMIT header.
type USBIPCmdSubmit struct {
	Basic           USBIPHeaderBasic
	TransferFlags   uint32
	TransferBufferLength uint32
	StartFrame      uint32
	NumberOfPackets uint32
	Interval        uint32
	Setup           [8]byte
}

// USBIPRetSubmit represents USBIP_RET_SUBMIT header.
type USBIPRetSubmit struct {
	Basic         USBIPHeaderBasic
	Status        uint32
	ActualLength  uint32
	StartFrame    uint32
	NumberOfPackets uint32
	ErrorCount    uint32
	Padding       [8]byte
}

// USBIPCmdUnlink represents USBIP_CMD_UNLINK header.
type USBIPCmdUnlink struct {
	Basic        USBIPHeaderBasic
	UnlinkSeqNum uint32
	Padding      [24]byte
}

// USBIPRetUnlink represents USBIP_RET_UNLINK header.
type USBIPRetUnlink struct {
	Basic   USBIPHeaderBasic
	Status  uint32
	Padding [24]byte
}

// ReadReqDevListHeader reads OP_REQ_DEVLIST header.
func ReadReqDevListHeader(r io.Reader) (*ReqDevListHeader, error) {
	h := &ReqDevListHeader{}
	if err := binary.Read(r, binary.BigEndian, h); err != nil {
		return nil, fmt.Errorf("failed to read REQ_DEVLIST header: %w", err)
	}
	return h, nil
}

// WriteReqDevListHeader writes OP_REQ_DEVLIST header.
func WriteReqDevListHeader(w io.Writer) error {
	h := &ReqDevListHeader{
		Version: ProtocolVersion,
		Command: OP_REQ_DEVLIST,
		Status:  0,
	}
	return binary.Write(w, binary.BigEndian, h)
}

// ReadRepDevListHeader reads OP_REP_DEVLIST header.
func ReadRepDevListHeader(r io.Reader) (*RepDevListHeader, error) {
	h := &RepDevListHeader{}
	if err := binary.Read(r, binary.BigEndian, h); err != nil {
		return nil, fmt.Errorf("failed to read REP_DEVLIST header: %w", err)
	}
	return h, nil
}

// WriteRepDevListHeader writes OP_REP_DEVLIST header.
func WriteRepDevListHeader(w io.Writer, nDevices uint32) error {
	h := &RepDevListHeader{
		Version: ProtocolVersion,
		Command: OP_REP_DEVLIST,
		Status:  0,
		NDevices: nDevices,
	}
	return binary.Write(w, binary.BigEndian, h)
}

// ReadReqImportHeader reads OP_REQ_IMPORT header.
func ReadReqImportHeader(r io.Reader) (*ReqImportHeader, error) {
	h := &ReqImportHeader{}
	if err := binary.Read(r, binary.BigEndian, h); err != nil {
		return nil, fmt.Errorf("failed to read REQ_IMPORT header: %w", err)
	}
	return h, nil
}

// WriteReqImportHeader writes OP_REQ_IMPORT header.
func WriteReqImportHeader(w io.Writer, busID string) error {
	h := &ReqImportHeader{
		Version: ProtocolVersion,
		Command: OP_REQ_IMPORT,
		Status:  0,
	}
	copy(h.BusID[:], busID)
	return binary.Write(w, binary.BigEndian, h)
}

// ReadRepImportHeader reads OP_REP_IMPORT header.
func ReadRepImportHeader(r io.Reader) (*RepImportHeader, error) {
	h := &RepImportHeader{}
	if err := binary.Read(r, binary.BigEndian, h); err != nil {
		return nil, fmt.Errorf("failed to read REP_IMPORT header: %w", err)
	}
	return h, nil
}

// WriteRepImportHeader writes OP_REP_IMPORT header.
func WriteRepImportHeader(w io.Writer, status uint32) error {
	h := &RepImportHeader{
		Version: ProtocolVersion,
		Command: OP_REP_IMPORT,
		Status:  status,
	}
	return binary.Write(w, binary.BigEndian, h)
}

// ReadUSBIPCmdSubmit reads USBIP_CMD_SUBMIT header.
func ReadUSBIPCmdSubmit(r io.Reader) (*USBIPCmdSubmit, error) {
	h := &USBIPCmdSubmit{}
	if err := binary.Read(r, binary.BigEndian, &h.Basic); err != nil {
		return nil, fmt.Errorf("failed to read CMD_SUBMIT basic header: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &h.TransferFlags); err != nil {
		return nil, fmt.Errorf("failed to read CMD_SUBMIT transfer_flags: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &h.TransferBufferLength); err != nil {
		return nil, fmt.Errorf("failed to read CMD_SUBMIT transfer_buffer_length: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &h.StartFrame); err != nil {
		return nil, fmt.Errorf("failed to read CMD_SUBMIT start_frame: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &h.NumberOfPackets); err != nil {
		return nil, fmt.Errorf("failed to read CMD_SUBMIT number_of_packets: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &h.Interval); err != nil {
		return nil, fmt.Errorf("failed to read CMD_SUBMIT interval: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &h.Setup); err != nil {
		return nil, fmt.Errorf("failed to read CMD_SUBMIT setup: %w", err)
	}
	return h, nil
}

// WriteUSBIPCmdSubmit writes USBIP_CMD_SUBMIT header.
func WriteUSBIPCmdSubmit(w io.Writer, h *USBIPCmdSubmit) error {
	if err := binary.Write(w, binary.BigEndian, &h.Basic); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, h.TransferFlags); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, h.TransferBufferLength); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, h.StartFrame); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, h.NumberOfPackets); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, h.Interval); err != nil {
		return err
	}
	return binary.Write(w, binary.BigEndian, h.Setup)
}

// ReadUSBIPRetSubmit reads USBIP_RET_SUBMIT header.
func ReadUSBIPRetSubmit(r io.Reader) (*USBIPRetSubmit, error) {
	h := &USBIPRetSubmit{}
	if err := binary.Read(r, binary.BigEndian, &h.Basic); err != nil {
		return nil, fmt.Errorf("failed to read RET_SUBMIT basic header: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &h.Status); err != nil {
		return nil, fmt.Errorf("failed to read RET_SUBMIT status: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &h.ActualLength); err != nil {
		return nil, fmt.Errorf("failed to read RET_SUBMIT actual_length: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &h.StartFrame); err != nil {
		return nil, fmt.Errorf("failed to read RET_SUBMIT start_frame: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &h.NumberOfPackets); err != nil {
		return nil, fmt.Errorf("failed to read RET_SUBMIT number_of_packets: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &h.ErrorCount); err != nil {
		return nil, fmt.Errorf("failed to read RET_SUBMIT error_count: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &h.Padding); err != nil {
		return nil, fmt.Errorf("failed to read RET_SUBMIT padding: %w", err)
	}
	return h, nil
}

// WriteUSBIPRetSubmit writes USBIP_RET_SUBMIT header.
func WriteUSBIPRetSubmit(w io.Writer, h *USBIPRetSubmit) error {
	if err := binary.Write(w, binary.BigEndian, &h.Basic); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, h.Status); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, h.ActualLength); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, h.StartFrame); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, h.NumberOfPackets); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, h.ErrorCount); err != nil {
		return err
	}
	return binary.Write(w, binary.BigEndian, h.Padding)
}

// ReadNullTerminatedString reads a null-terminated string from a fixed-size array.
func ReadNullTerminatedString(data []byte) string {
	idx := bytes.IndexByte(data, 0)
	if idx == -1 {
		return string(data)
	}
	return string(data[:idx])
}

// WriteNullTerminatedString writes a string to a fixed-size array with null termination.
func WriteNullTerminatedString(data []byte, s string) {
	if len(s) >= len(data) {
		copy(data, s[:len(data)-1])
		data[len(data)-1] = 0
	} else {
		copy(data, s)
		data[len(s)] = 0
	}
}

// FormatBusID formats bus and device numbers into a bus ID string.
func FormatBusID(busNum, devNum uint32) string {
	return fmt.Sprintf("%d-%d", busNum, devNum)
}

// ParseBusID parses a bus ID string into bus and device numbers.
func ParseBusID(busID string) (uint32, uint32, error) {
	parts := strings.Split(busID, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid bus ID format: %s", busID)
	}
	var busNum, devNum uint32
	_, err := fmt.Sscanf(parts[0], "%d", &busNum)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid bus number: %s", parts[0])
	}
	_, err = fmt.Sscanf(parts[1], "%d", &devNum)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid device number: %s", parts[1])
	}
	return busNum, devNum, nil
}

// URBHeader represents the basic USB/IP URB header (usbip_header_basic).
type URBHeader struct {
	Command   uint32
	SeqNum    uint32
	Devid     uint32
	Direction uint32
	Endpoint  uint32
}

// ReadURBHeader reads a URB header from the connection.
func ReadURBHeader(r io.Reader) (*URBHeader, error) {
	var hdr URBHeader
	if err := binary.Read(r, binary.BigEndian, &hdr.Command); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &hdr.SeqNum); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &hdr.Devid); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &hdr.Direction); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &hdr.Endpoint); err != nil {
		return nil, err
	}
	return &hdr, nil
}

// WriteURBHeader writes a URB header to the connection.
func WriteURBHeader(w io.Writer, hdr *URBHeader) error {
	fields := []interface{}{
		hdr.Command,
		hdr.SeqNum,
		hdr.Devid,
		hdr.Direction,
		hdr.Endpoint,
	}
	for _, f := range fields {
		if err := binary.Write(w, binary.BigEndian, f); err != nil {
			return err
		}
	}
	return nil
}

// ReadUint32 reads a uint32 from the reader in big-endian format.
func ReadUint32(r io.Reader) (uint32, error) {
	var v uint32
	err := binary.Read(r, binary.BigEndian, &v)
	return v, err
}