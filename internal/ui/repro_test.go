package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pteich/crdlens/internal/config"
	"github.com/pteich/crdlens/internal/k8s"
	"github.com/pteich/crdlens/internal/types"
	"github.com/pteich/crdlens/internal/ui/views"
	"github.com/stretchr/testify/assert"
)

func TestModel_Reproduction(t *testing.T) {
	cfg := &config.Config{}
	client := &k8s.Client{}
	m := NewModel(cfg, client)

	// Test Init
	m.Init()

	// Test Update with WindowSizeMsg
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	newModel, _ := m.Update(msg)
	m = newModel.(Model)

	assert.True(t, m.ready)
	assert.NotNil(t, m.crdList)

	// Test View
	view := m.View()
	assert.Contains(t, view, "Loading CRDs...")

	// Test another Update
	msg2 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newModel, _ = m.Update(msg2)
	m = newModel.(Model)

	// Test state transition (simulated)
	m.state = CRListView
	m.crList = views.NewCRListModel(client, types.CRDInfo{Kind: "Test"}, "default", 100, 50)
	view = m.View()
	assert.Contains(t, view, "Loading Custom Resources...")
}
