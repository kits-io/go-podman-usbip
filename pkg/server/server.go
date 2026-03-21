// Package server implements the USB/IP server.
package server

import (
	"fmt"
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

	// TODO: implement protocol handling
	header, err := protocol.ReadHeader(conn)
	if err != nil {
		fmt.Printf("Failed to read header: %v\n", err)
		return
	}

	fmt.Printf("Received command: %x\n", header.Command)
}
