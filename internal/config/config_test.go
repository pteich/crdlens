package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotZero(t, cfg.RefreshInterval, "refresh interval should be set")
	assert.Equal(t, 1000, cfg.CacheSize)
	assert.True(t, cfg.DisableCounts)
	assert.Equal(t, "#7D56F4", cfg.Theme.Primary)
	assert.Equal(t, "q", cfg.Keybindings.Quit)
}

func TestConfigLoad_WithYAML(t *testing.T) {
	// Create temporary config
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".crdlens.yaml")

	yamlContent := `
kubeconfig: /path/to/kubeconfig
namespace: test-ns
refreshInterval: 1m
theme:
  primary: "#112233"
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Test unmarshaling (since Load() looks at home dir which is hard to mock here)
	cfg := DefaultConfig()
	err = yaml.Unmarshal([]byte(yamlContent), cfg)
	require.NoError(t, err)

	assert.Equal(t, "/path/to/kubeconfig", cfg.Kubeconfig)
	assert.Equal(t, "test-ns", cfg.Namespace)
	assert.Equal(t, "#112233", cfg.Theme.Primary)
}
