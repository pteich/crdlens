package views

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pteich/crdlens/internal/k8s"
	"github.com/pteich/crdlens/internal/search"
	"github.com/pteich/crdlens/internal/types"
)

// CRListModel is the model for the CR list view
type CRListModel struct {
	table        table.Model
	client       *k8s.Client
	crd          types.CRDInfo
	loading      bool
	err          error
	allResources []types.Resource
	filtered     []types.Resource
	width        int
	height       int
	namespace    string
	textinput    textinput.Model
	filtering    bool
	spinner      spinner.Model
}

// NewCRListModel creates a new CR list model
func NewCRListModel(client *k8s.Client, crd types.CRDInfo, namespace string, width, height int) *CRListModel {
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

	ti := textinput.New()
	ti.Placeholder = "Search resources..."
	ti.Prompt = "/ "

	spn := spinner.New()
	spn.Spinner = spinner.Dot
	spn.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#F780E2"))

	return &CRListModel{
		table:     t,
		client:    client,
		crd:       crd,
		width:     width,
		height:    height,
		namespace: namespace,
		textinput: ti,
		spinner:   spn,
		loading:   true,
	}
}

// Init initializes the model
func (m *CRListModel) Init() tea.Cmd {
	return tea.Batch(m.FetchCRs, m.spinner.Tick)
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
		m.allResources = msg.Resources
		m.filtered = msg.Resources
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

	case tea.KeyMsg:
		if m.filtering {
			switch msg.String() {
			case "esc", "enter":
				m.filtering = false
				m.textinput.Blur()
				return m, nil
			}
		} else {
			switch msg.String() {
			case "/":
				m.filtering = true
				m.textinput.Focus()
				return m, tea.Batch(textinput.Blink)
			}
		}
	}

	if m.filtering {
		var cmd tea.Cmd
		m.textinput, cmd = m.textinput.Update(msg)

		// Filter resources based on query
		m.filtered = search.MatchResources(m.textinput.Value(), m.allResources)
		rows := make([]table.Row, len(m.filtered))
		for i, res := range m.filtered {
			rows[i] = table.Row{
				res.Name,
				res.Namespace,
				res.Age.String(),
			}
		}
		m.table.SetRows(rows)
		return m, cmd
	}

	var sCmd tea.Cmd
	m.spinner, sCmd = m.spinner.Update(msg)

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, tea.Batch(cmd, sCmd)
}

// SelectedResource returns the currently selected resource
func (m *CRListModel) SelectedResource() types.Resource {
	idx := m.table.Cursor()
	if idx >= 0 && idx < len(m.filtered) {
		return m.filtered[idx]
	}
	return types.Resource{}
}

// View renders the model
func (m *CRListModel) View() string {
	if m.loading {
		return fmt.Sprintf("\n %s Loading Custom Resources...", m.spinner.View())
	}
	if m.err != nil {
		return fmt.Sprintf("Error fetching CRs: %v", m.err)
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Render(fmt.Sprintf("Resources: %s", m.crd.Kind))

	view := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"\n",
		m.table.View(),
	)

	if m.filtering {
		view = lipgloss.JoinVertical(lipgloss.Left,
			view,
			"\n",
			m.textinput.View(),
		)
	}

	return view
}

// Refresh re-fetches the resources
func (m *CRListModel) Refresh(namespace string) tea.Cmd {
	m.namespace = namespace
	return m.FetchCRs
}

// IsFiltering returns true if the list is currently filtering
func (m *CRListModel) IsFiltering() bool {
	return m.filtering
}

// FetchedCRsMsg is sent when CRs are successfully fetched
type FetchedCRsMsg struct {
	Resources []types.Resource
}

// FetchCRs is a command to fetch CRs from the cluster
func (m *CRListModel) FetchCRs() tea.Msg {
	dynamicSvc := m.client.Dynamic()
	resources, err := dynamicSvc.ListResources(context.Background(), m.crd.GVR, m.namespace)
	if err != nil {
		return ErrorMsg{Err: err}
	}
	return FetchedCRsMsg{Resources: resources}
}
