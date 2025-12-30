package views

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pteich/crdlens/internal/k8s"
	"github.com/pteich/crdlens/internal/search"
	"github.com/pteich/crdlens/internal/types"
)

// CRDItem implements the list.Item interface
type CRDItem struct {
	types.CRDInfo
}

func (i CRDItem) FilterValue() string { return i.Name }
func (i CRDItem) Title() string       { return i.Kind }
func (i CRDItem) Description() string {
	desc := fmt.Sprintf("%s (%s)", i.Group, i.Scope)
	if i.Count > 0 {
		desc = fmt.Sprintf("%s - %d CR%s", desc, i.Count, pluralize(i.Count))
	}
	return desc
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// CRDListModel is the model for the CRD list view
type CRDListModel struct {
	list      list.Model
	client    *k8s.Client
	loading   bool
	err       error
	allCRDs   []types.CRDInfo
	filtered  []types.CRDInfo
	textinput textinput.Model
	filtering bool
	spinner   spinner.Model
}

// NewCRDListModel creates a new CRD list model
func NewCRDListModel(client *k8s.Client, width, height int) *CRDListModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), width, height)
	l.Title = "Custom Resource Definitions"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(false)

	ti := textinput.New()
	ti.Placeholder = "Search CRDs..."
	ti.Prompt = "/ "

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#F780E2"))

	return &CRDListModel{
		list:      l,
		client:    client,
		textinput: ti,
		spinner:   s,
		loading:   true,
	}
}

// Init initializes the model
func (m *CRDListModel) Init() tea.Cmd {
	return tea.Batch(m.FetchCRDs, m.spinner.Tick)
}

// Update handles messages
func (m *CRDListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case FetchedCRDsMsg:
		m.loading = false
		m.allCRDs = msg.CRDs
		m.filtered = msg.CRDs
		items := make([]list.Item, len(msg.CRDs))
		for i, crd := range msg.CRDs {
			items[i] = CRDItem{crd}
		}
		return m, tea.Batch(m.list.SetItems(items), m.FetchCRDCounts)

	case CRDCountsMsg:
		for i, crd := range m.allCRDs {
			if count, ok := msg.Counts[crd.Name]; ok {
				m.allCRDs[i].Count = count
			}
		}
		for i, crd := range m.filtered {
			if count, ok := msg.Counts[crd.Name]; ok {
				m.filtered[i].Count = count
			}
		}
		items := make([]list.Item, len(m.filtered))
		for i, crd := range m.filtered {
			items[i] = CRDItem{crd}
		}
		return m, m.list.SetItems(items)

	case ErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case tea.KeyMsg:
		if m.filtering {
			switch msg.String() {
			case "esc", "enter":
				m.filtering = false
				m.textinput.Blur()
				return m, nil
			}
			// Handle filtering immediately within KeyMsg case to prevent list interference
			var cmd tea.Cmd
			m.textinput, cmd = m.textinput.Update(msg)

			m.filtered = search.MatchCRDs(m.textinput.Value(), m.allCRDs)
			items := make([]list.Item, len(m.filtered))
			for i, crd := range m.filtered {
				items[i] = CRDItem{crd}
			}
			m.list.SetItems(items)
			return m, cmd
		} else {
			switch msg.String() {
			case "/":
				m.filtering = true
				m.textinput.Focus()
				return m, tea.Batch(textinput.Blink)
			}
		}
	}

	// Only reached when NOT filtering
	var sCmd tea.Cmd
	m.spinner, sCmd = m.spinner.Update(msg)

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, tea.Batch(cmd, sCmd)
}

// View renders the model
func (m *CRDListModel) View() string {
	if m.loading {
		return fmt.Sprintf("\n %s Loading CRDs...", m.spinner.View())
	}
	if m.err != nil {
		return fmt.Sprintf("Error fetching CRDs: %v", m.err)
	}

	view := m.list.View()
	if m.filtering {
		view = lipgloss.JoinVertical(lipgloss.Left,
			view,
			"\n",
			m.textinput.View(),
		)
	}
	return view
}

// SelectedCRD returns the currently selected CRDInfo
func (m *CRDListModel) SelectedCRD() types.CRDInfo {
	if i, ok := m.list.SelectedItem().(CRDItem); ok {
		return i.CRDInfo
	}
	return types.CRDInfo{}
}

// IsFiltering returns true if the list is currently filtering
func (m *CRDListModel) IsFiltering() bool {
	return m.filtering
}

// Refresh re-fetches the CRDs
func (m *CRDListModel) Refresh() tea.Cmd {
	return m.FetchCRDs
}

// Messages
type FetchedCRDsMsg struct {
	CRDs []types.CRDInfo
}

type ErrorMsg struct {
	Err error
}

type CRDCountsMsg struct {
	Counts map[string]int
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

// FetchCRDCounts is a command to fetch counts for all CRDs (async)
func (m *CRDListModel) FetchCRDCounts() tea.Msg {
	dynamicSvc := m.client.Dynamic()
	namespace := ""
	counts := make(map[string]int)

	for _, crd := range m.allCRDs {
		count, err := dynamicSvc.CountResources(context.Background(), crd.GVR, namespace)
		if err != nil {
			continue
		}
		counts[crd.Name] = count
	}

	return CRDCountsMsg{Counts: counts}
}
