package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	chroma "github.com/amikos-tech/chroma-go-local"
)

func main() {
	// Initialize the library - CHROMA_LIB_PATH must be set
	if err := chroma.Init(""); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize Chroma: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Chroma shim version: %s\n", chroma.Version())

	// Start server with builder pattern
	server, err := chroma.NewServer(
		chroma.WithPort(8000),
		chroma.WithListenAddress("127.0.0.1"),
		chroma.WithPersistPath("./chroma_data"),
		chroma.WithAllowReset(true),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Server started at %s\n", server.URL())
	fmt.Println("Press Ctrl+C to stop...")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nStopping server...")
	if err := server.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Error stopping server: %v\n", err)
	}
	fmt.Println("Server stopped.")
}
