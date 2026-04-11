package main

import (
	"fmt"
	"os"

	"github.com/hyperplex-tech/bore/internal/daemon"
)

func main() {
	opts := daemon.Options{
		ConfigPath: os.Getenv("BORE_CONFIG"),
		SocketPath: os.Getenv("BORE_SOCKET"),
		LogLevel:   os.Getenv("BORE_LOG_LEVEL"),
	}

	if opts.LogLevel == "" {
		opts.LogLevel = "info"
	}

	d, err := daemon.New(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := d.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
