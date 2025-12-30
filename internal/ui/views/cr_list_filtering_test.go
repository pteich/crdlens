package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pteich/crdlens/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestCRListModel_Filtering(t *testing.T) {
	m := NewCRListModel(nil, types.CRDInfo{Kind: "TestKind"}, 100, 100)

	resources := []types.Resource{
		{Name: "abc", Namespace: "default"},
		{Name: "def", Namespace: "kube-system"},
	}

	m.Update(FetchedCRsMsg{Resources: resources})

	assert.Len(t, m.filtered, 2)

	// Start filtering
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.True(t, m.filtering)

	// Type "a"
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.Len(t, m.filtered, 1)
	assert.Equal(t, "abc", m.filtered[0].Name)

	// Exit filtering
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.False(t, m.filtering)
}
