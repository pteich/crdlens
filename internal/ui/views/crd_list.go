package views

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pteich/crdlens/internal/k8s"
	"github.com/pteich/crdlens/internal/types"
)

// CRDItem implements the list.Item interface
type CRDItem struct {
	types.CRDInfo
}

func (i CRDItem) FilterValue() string { return i.Name }
func (i CRDItem) Title() string       { return i.Kind }
func (i CRDItem) Description() string { return fmt.Sprintf("%s (%s)", i.Group, i.Scope) }

// CRDListModel is the model for the CRD list view
type CRDListModel struct {
	list    list.Model
	client  *k8s.Client
	loading bool
	err     error
}

// NewCRDListModel creates a new CRD list model
func NewCRDListModel(client *k8s.Client, width, height int) *CRDListModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), width, height)
	l.Title = "Custom Resource Definitions"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	return &CRDListModel{
		list:   l,
		client: client,
	}
}

// Init initializes the model
func (m *CRDListModel) Init() tea.Cmd {
	return m.FetchCRDs
}

// Update handles messages
func (m *CRDListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case FetchedCRDsMsg:
		m.loading = false
		items := make([]list.Item, len(msg.CRDs))
		for i, crd := range msg.CRDs {
			items[i] = CRDItem{crd}
		}
		return m, m.list.SetItems(items)

	case ErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the model
func (m *CRDListModel) View() string {
	if m.loading {
		return "Loading CRDs..."
	}
	if m.err != nil {
		return fmt.Sprintf("Error fetching CRDs: %v", m.err)
	}
	return m.list.View()
}

// Messages
type FetchedCRDsMsg struct {
	CRDs []types.CRDInfo
}

type ErrorMsg struct {
	Err error
}

// FetchCRDs is a command to fetch CRDs from the cluster
func (m *CRDListModel) FetchCRDs() tea.Msg {
	m.loading = true
	discoverySvc := m.client.Discovery()
	crds, err := discoverySvc.ListCRDs(context.Background())
	if err != nil {
		return ErrorMsg{Err: err}
	}
	return FetchedCRDsMsg{CRDs: crds}
}
