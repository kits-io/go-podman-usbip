// usbip-client is the USB/IP client for Linux containers.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kits-io/go-podman-usbip/pkg/client"
	"github.com/kits-io/go-podman-usbip/pkg/device"
)

var (
	Version = "dev"
)

var (
	serverHost = flag.String("server", "host.docker.internal", "USB/IP server host")
	serverPort = flag.Int("port", 3240, "USB/IP server port")
	listCmd    = flag.Bool("list", false, "List available devices from server")
	attachCmd  = flag.Bool("attach", false, "Attach a USB device")
	busID      = flag.String("bus-id", "", "Bus ID of device to attach (e.g., 1-2)")
)

func main() {
	flag.Parse()

	if *listCmd {
		listDevices()
		return
	}

	if *attachCmd {
		if *busID == "" {
			fmt.Fprintf(os.Stderr, "Error: --bus-id is required for --attach\n")
			os.Exit(1)
		}
		attachDevice()
		return
	}

	fmt.Printf("usbip-client %s\n", Version)
	fmt.Println("Use --list to list devices or --attach to attach a device")
	flag.Usage()
}

func listDevices() {
	fmt.Printf("usbip-client %s\n", Version)
	fmt.Printf("Connecting to %s:%d...\n", *serverHost, *serverPort)

	cli := client.New(client.Config{
		ServerHost: *serverHost,
		ServerPort: *serverPort,
	})

	if err := cli.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server: %v\n", err)
		os.Exit(1)
	}
	defer cli.Close()

	devices, err := cli.ListDevices()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing devices: %v\n", err)
		os.Exit(1)
	}

	if len(devices) == 0 {
		fmt.Println("No devices available from server")
		return
	}

	fmt.Printf("\nAvailable devices:\n")
	for i, dev := range devices {
		fmt.Printf("%3d: %s\n", i+1, dev)
		fmt.Printf("     Bus ID: %s\n", dev.BusID)
		fmt.Printf("     Path: %s\n", dev.Path)
		fmt.Printf("     Speed: %s\n", device.SpeedString(dev.Speed))
	}
}

func attachDevice() {
	fmt.Printf("usbip-client %s\n", Version)
	fmt.Printf("Attaching device %s from %s:%d...\n", *busID, *serverHost, *serverPort)

	cli := client.New(client.Config{
		ServerHost: *serverHost,
		ServerPort: *serverPort,
	})

	if err := cli.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server: %v\n", err)
		os.Exit(1)
	}
	defer cli.Close()

	dev, err := cli.ImportDevice(*busID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error importing device: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nDevice attached successfully:\n")
	fmt.Printf("  %s\n", dev)
	fmt.Printf("  Bus ID: %s\n", dev.BusID)
	fmt.Printf("  Vendor/Product: %04x:%04x\n", dev.VendorID, dev.ProductID)

	fmt.Println("\nNote: USB traffic handling not yet implemented.")
	fmt.Println("The connection is kept alive but URB traffic is not processed.")
}