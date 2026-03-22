#!/bin/bash
# Test script for USB/IP client

echo "USB/IP Client Test Script"
echo "========================"

# Check if we're running in Linux
if [ "$(uname)" != "Linux" ]; then
    echo "Error: This script must run on Linux (inside a container or VM)"
    exit 1
fi

# Check if usbip-client binary exists
CLIENT_BIN="./usbip-client-linux-amd64"
if [ ! -f "$CLIENT_BIN" ]; then
    echo "Error: usbip-client binary not found"
    exit 1
fi

# Check if vhci_hcd module is available
echo "Checking for vhci_hcd kernel module..."
if [ -d "/sys/module/vhci_hcd" ]; then
    echo "✓ vhci_hcd module is loaded"
elif [ -f "/lib/modules/$(uname -r)/kernel/drivers/usb/usbip/vhci_hcd.ko" ]; then
    echo "✓ vhci_hcd module is available but not loaded"
    echo "  Run: sudo modprobe vhci_hcd"
else
    echo "✗ vhci_hcd module not found"
    echo "  USB/IP client cannot work without vhci_hcd"
    echo ""
    echo "To add vhci_hcd support, you need to:"
    echo "1. Build a custom Fedora CoreOS image with the module"
    echo "2. Or use a different VM with full kernel access"
fi

echo ""
echo "Usage:"
echo "  $CLIENT_BIN --list --server <server-host>"
echo "  $CLIENT_BIN --attach --bus-id <bus-id> --server <server-host>"