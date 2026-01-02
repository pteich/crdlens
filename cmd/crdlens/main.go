package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pteich/crdlens/internal/config"
	"github.com/pteich/crdlens/internal/k8s"
	"github.com/pteich/crdlens/internal/ui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	client, err := k8s.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	m := ui.NewModel(cfg, client)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
