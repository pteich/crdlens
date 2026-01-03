package config

import (
	"time"
)

// Config represents the tool's configuration
type Config struct {
	Kubeconfig      string            `yaml:"kubeconfig"`
	Context         string            `yaml:"context"`
	Namespace       string            `yaml:"namespace"`
	AllNamespaces   bool              `yaml:"allNamespaces"`
	RefreshInterval time.Duration     `yaml:"refreshInterval"`
	Theme           ThemeConfig       `yaml:"theme"`
	Keybindings     KeybindingsConfig `yaml:"keybindings"`
	CacheSize       int               `yaml:"cacheSize"`
	DisableCounts   bool              `yaml:"disableCounts"`
}

// ThemeConfig defines the appearance of the TUI
type ThemeConfig struct {
	Primary   string `yaml:"primary"`
	Secondary string `yaml:"secondary"`
	Accent    string `yaml:"accent"`
}

// KeybindingsConfig allows overriding default keybindings
type KeybindingsConfig struct {
	Quit            string `yaml:"quit"`
	Help            string `yaml:"help"`
	Search          string `yaml:"search"`
	Back            string `yaml:"back"`
	ToggleNamespace string `yaml:"toggleNamespace"`
	Refresh         string `yaml:"refresh"`
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		RefreshInterval: 30 * time.Second,
		CacheSize:       1000,
		DisableCounts:   true,
		Theme: ThemeConfig{
			Primary:   "#7D56F4",
			Secondary: "#F780E2",
			Accent:    "#00D9FF",
		},
		Keybindings: KeybindingsConfig{
			Quit:            "q",
			Help:            "?",
			Search:          "/",
			Back:            "esc",
			ToggleNamespace: "n",
			Refresh:         "r",
		},
	}
}
