package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pteich/crdlens/internal/config"
	"github.com/pteich/crdlens/internal/k8s"
	"github.com/stretchr/testify/assert"
)

func TestNewModel(t *testing.T) {
	cfg := config.DefaultConfig()
	client := &k8s.Client{}

	m := NewModel(cfg, client)

	assert.Equal(t, CRDListView, m.state)
	assert.Equal(t, cfg, m.config)
	assert.False(t, m.ready)
}

func TestModel_Update_WindowSize(t *testing.T) {
	cfg := config.DefaultConfig()
	client := &k8s.Client{}
	m := NewModel(cfg, client)

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	newModel, _ := m.Update(msg)

	updatedModel := newModel.(Model)
	assert.Equal(t, 100, updatedModel.width)
	assert.Equal(t, 50, updatedModel.height)
	assert.True(t, updatedModel.ready)
	assert.NotNil(t, updatedModel.crdList)
}

func TestModel_Update_Quit(t *testing.T) {
	cfg := config.DefaultConfig()
	client := &k8s.Client{}
	m := NewModel(cfg, client)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	_, cmd := m.Update(msg)

	assert.NotNil(t, cmd)
	// We can't easily check if it's tea.Quit without more complex logic, but we can verify it's not nil
}
