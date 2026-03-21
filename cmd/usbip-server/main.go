// usbip-server is the USB/IP server for macOS.
package main

import (
	"fmt"
	"os"
)

var Version = "dev"

func main() {
	fmt.Printf("usbip-server %s\n", Version)
	fmt.Println("USB/IP server for macOS")
	fmt.Println("TODO: implement server")
	os.Exit(0)
}
