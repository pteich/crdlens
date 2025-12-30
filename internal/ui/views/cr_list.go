package views

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pteich/crdlens/internal/k8s"
	"github.com/pteich/crdlens/internal/types"
)

// CRListModel is the model for the CR list view
type CRListModel struct {
	table   table.Model
	client  *k8s.Client
	crd     types.CRDInfo
	loading bool
	err     error
	width   int
	height  int
}

// NewCRListModel creates a new CR list model
func NewCRListModel(client *k8s.Client, crd types.CRDInfo, width, height int) *CRListModel {
	columns := []table.Column{
		{Title: "Name", Width: 40},
		{Title: "Namespace", Width: 30},
		{Title: "Age", Width: 15},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(height-10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return &CRListModel{
		table:  t,
		client: client,
		crd:    crd,
		width:  width,
		height: height,
	}
}

// Init initializes the model
func (m *CRListModel) Init() tea.Cmd {
	return m.FetchCRs
}

// Update handles messages
func (m *CRListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case FetchedCRsMsg:
		m.loading = false
		rows := make([]table.Row, len(msg.Resources))
		for i, res := range msg.Resources {
			rows[i] = table.Row{
				res.Name,
				res.Namespace,
				res.Age.String(),
			}
		}
		m.table.SetRows(rows)
		return m, nil

	case ErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetHeight(m.height - 10)
		return m, nil
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the model
func (m *CRListModel) View() string {
	if m.loading {
		return "Loading Custom Resources..."
	}
	if m.err != nil {
		return fmt.Sprintf("Error fetching CRs: %v", m.err)
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Render(fmt.Sprintf("Resources: %s", m.crd.Kind))

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		"\n",
		m.table.View(),
	)
}

// FetchedCRsMsg is sent when CRs are successfully fetched
type FetchedCRsMsg struct {
	Resources []types.Resource
}

// FetchCRs is a command to fetch CRs from the cluster
func (m *CRListModel) FetchCRs() tea.Msg {
	m.loading = true
	dynamicSvc := m.client.Dynamic()
	// Use all namespaces if the CRD is cluster scoped, or if we want to filter
	// For now let's assume all namespaces for listing if it's cluster scoped
	ns := ""
	resources, err := dynamicSvc.ListResources(context.Background(), m.crd.GVR, ns)
	if err != nil {
		return ErrorMsg{Err: err}
	}
	return FetchedCRsMsg{Resources: resources}
}
