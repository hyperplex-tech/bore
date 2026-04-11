package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/hyperplex-tech/bore/internal/cli"
	"github.com/hyperplex-tech/bore/internal/config"
	"github.com/hyperplex-tech/bore/internal/tui"
)

func main() {
	socketPath := os.Getenv("BORE_SOCKET")
	if socketPath == "" {
		socketPath = config.SocketPath()
	}

	clients, err := cli.Dial(socketPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot connect to bore daemon: %v\n", err)
		fmt.Fprintf(os.Stderr, "Is the daemon running? Start it with: bored\n")
		os.Exit(1)
	}
	defer clients.Close()

	model := tui.NewModel(clients)

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
