// usbip-server is the USB/IP server for macOS.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kits-io/go-podman-usbip/pkg/device"
	"github.com/kits-io/go-podman-usbip/pkg/platform"
	"github.com/kits-io/go-podman-usbip/pkg/server"
)

var (
	Version = "dev"
)

var (
	host = flag.String("host", "0.0.0.0", "Host to listen on")
	port = flag.Int("port", 3240, "Port to listen on")
	list = flag.Bool("list", false, "List available USB devices")
)

func main() {
	flag.Parse()

	if *list {
		listDevices()
		return
	}

	fmt.Printf("usbip-server %s\n", Version)
	fmt.Println("Starting USB/IP server...")

	// Discover USB devices
	devices, err := platform.DiscoverDevices()
	if err != nil {
		fmt.Printf("Warning: Failed to discover devices: %v\n", err)
		fmt.Println("Continuing with no devices...")
		devices = []*device.Device{}
	}

	if len(devices) > 0 {
		fmt.Printf("Found %d USB device(s):\n", len(devices))
		for _, dev := range devices {
			fmt.Printf("  - %s\n", dev)
		}
	}

	// Start server
	srv := server.New(server.Config{
		Host: *host,
		Port: *port,
	})
	srv.SetDevices(devices)

	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")
	srv.Stop()
}

func listDevices() {
	fmt.Printf("usbip-server %s\n", Version)
	fmt.Println("Available USB devices:")

	devices, err := platform.DiscoverDevices()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error discovering devices: %v\n", err)
		os.Exit(1)
	}

	if len(devices) == 0 {
		fmt.Println("No USB devices found")
		return
	}

	for i, dev := range devices {
		fmt.Printf("%3d: %s\n", i+1, dev)
		fmt.Printf("     Path: %s\n", dev.Path)
		fmt.Printf("     Speed: %s\n", device.SpeedString(dev.Speed))
	}
}