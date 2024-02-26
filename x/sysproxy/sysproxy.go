package sysproxy

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func SafeCloseProxy(done chan bool, sigs chan os.Signal) {
	// Channel to indicate the program can stop
	// Register the channel to receive notifications of the specified signals
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	// Start a goroutine to handle the signals
	// This goroutine executes a blocking receive for signals
	go func() {
		sig := <-sigs
		fmt.Println(sig)
		// Here you can call your cleanup or exit function
		if err := UnsetProxy(); err != nil {
			fmt.Println("Error setting up proxy:", err)
		} else {
			fmt.Println("Proxy unset successful")
		}
		done <- true
	}()
}
