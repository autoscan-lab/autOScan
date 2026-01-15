package app

import (
	"github.com/felipetrejos/felituive/internal/tui"
)

// Run initializes and starts the application
func Run() error {
	// TODO: Initialize config
	// TODO: Initialize adapters based on config
	// TODO: Wire up services

	return tui.Start()
}
