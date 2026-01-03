package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pteich/crdlens/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestCRDListModel_Filtering(t *testing.T) {
	m := NewCRDListModel(nil, "", 80, 24, false)

	crds := []types.CRDInfo{
		{Name: "certs", Kind: "Certificate"},
		{Name: "pods", Kind: "Pod"},
	}

	m.Update(FetchedCRDsMsg{CRDs: crds})

	assert.Len(t, m.filtered, 2)

	// Start filtering
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.True(t, m.filtering)

	// Type "c"
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	assert.Len(t, m.filtered, 1)
	assert.Equal(t, "Certificate", m.filtered[0].Kind)

	// Exit filtering
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.False(t, m.filtering)
}
