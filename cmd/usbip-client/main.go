// usbip-client is the USB/IP client for Linux containers.
package main

import (
	"fmt"
	"os"
)

var Version = "dev"

func main() {
	fmt.Printf("usbip-client %s\n", Version)
	fmt.Println("USB/IP client for Linux")
	fmt.Println("TODO: implement client")
	os.Exit(0)
}
