package views

import (
	"context"
	"sort"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pteich/crdlens/internal/k8s"
	"github.com/pteich/crdlens/internal/types"
	"sigs.k8s.io/yaml"
)

// CRDetailModel is the model for the CR detail view
type CRDetailModel struct {
	viewport viewport.Model
	table    table.Model
	client   *k8s.Client
	resource types.Resource
	events   []types.Event
	loading  bool
	err      error
	width    int
	height   int
	active   int // 0 for YAML, 1 for Events
}

// NewCRDetailModel creates a new CR detail model
func NewCRDetailModel(client *k8s.Client, resource types.Resource, width, height int) *CRDetailModel {
	vp := viewport.New(width, height/2)

	columns := []table.Column{
		{Title: "Type", Width: 10},
		{Title: "Reason", Width: 20},
		{Title: "Message", Width: 50},
		{Title: "Last Seen", Width: 20},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(false),
		table.WithHeight(height/2-5),
	)

	return &CRDetailModel{
		viewport: vp,
		table:    t,
		client:   client,
		resource: resource,
		width:    width,
		height:   height,
	}
}

// Init initializes the model
func (m *CRDetailModel) Init() tea.Cmd {
	return tea.Batch(
		m.FormatYAML,
		m.FetchEvents,
	)
}

// Update handles messages
func (m *CRDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case FormattedYAMLMsg:
		m.viewport.SetContent(msg.YAML)
		return m, nil

	case FetchedEventsMsg:
		m.events = msg.Events
		// Sort events by last timestamp descending
		sort.Slice(m.events, func(i, j int) bool {
			return m.events[i].LastTimestamp.After(m.events[j].LastTimestamp)
		})

		rows := make([]table.Row, len(m.events))
		for i, event := range m.events {
			rows[i] = table.Row{
				event.Type,
				event.Reason,
				event.Message,
				event.LastTimestamp.Format("2006-01-02 15:04:05"),
			}
		}
		m.table.SetRows(rows)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.active = (m.active + 1) % 2
			if m.active == 1 {
				m.table.Focus()
			} else {
				m.table.Blur()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height / 2
		m.table.SetHeight(m.height/2 - 5)
	}

	if m.active == 0 {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the model
func (m *CRDetailModel) View() string {
	yamlHeader := lipgloss.NewStyle().Bold(true).Render("YAML Configuration")
	if m.active == 0 {
		yamlHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4")).Render("YAML Configuration (Active)")
	}

	eventHeader := lipgloss.NewStyle().Bold(true).Render("Recent Events")
	if m.active == 1 {
		eventHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4")).Render("Recent Events (Active)")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		yamlHeader,
		m.viewport.View(),
		"\n",
		eventHeader,
		m.table.View(),
	)
}

// Messages
type FormattedYAMLMsg struct {
	YAML string
}

type FetchedEventsMsg struct {
	Events []types.Event
}

// FormatYAML is a command to format the resource as YAML
func (m *CRDetailModel) FormatYAML() tea.Msg {
	y, err := yaml.Marshal(m.resource.Raw)
	if err != nil {
		return ErrorMsg{Err: err}
	}
	return FormattedYAMLMsg{YAML: string(y)}
}

// FetchEvents is a command to fetch events for the resource
func (m *CRDetailModel) FetchEvents() tea.Msg {
	eventsSvc := m.client.Events(m.resource.Namespace)
	events, err := eventsSvc.GetEventsForResource(context.Background(), m.resource.Namespace, m.resource.UID)
	if err != nil {
		return ErrorMsg{Err: err}
	}
	return FetchedEventsMsg{Events: events}
}
