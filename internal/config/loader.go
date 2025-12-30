package config

import (
	"flag"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load reads configuration from file and overrides with CLI flags
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// 1. Try to load from ~/.crdlens.yaml
	home, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(home, ".crdlens.yaml")
		if _, err := os.Stat(configPath); err == nil {
			data, err := os.ReadFile(configPath)
			if err == nil {
				_ = yaml.Unmarshal(data, cfg)
			}
		}
	}

	// 2. Parse CLI flags
	kubeconfig := flag.String("kubeconfig", "", "path to the kubeconfig file")
	context := flag.String("context", "", "the name of the kubeconfig context to use")
	namespace := flag.String("namespace", "", "the namespace to use")
	allNamespaces := flag.Bool("all-namespaces", cfg.AllNamespaces, "list resources in all namespaces")

	flag.Parse()

	if *kubeconfig != "" {
		cfg.Kubeconfig = *kubeconfig
	}
	if *context != "" {
		cfg.Context = *context
	}
	if *namespace != "" {
		cfg.Namespace = *namespace
		cfg.AllNamespaces = false // If namespace is specified, disable all-namespaces
	}
	if *allNamespaces {
		cfg.AllNamespaces = true
	}

	return cfg, nil
}
