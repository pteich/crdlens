package views

import (
	"context"
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pteich/crdlens/internal/k8s"
	"github.com/pteich/crdlens/internal/types"
	"sigs.k8s.io/yaml"
)

type DetailViewMode int

const (
	DetailViewYAML DetailViewMode = iota
	DetailViewFields
	DetailViewEvents
)

func (m DetailViewMode) String() string {
	switch m {
	case DetailViewYAML:
		return "YAML"
	case DetailViewFields:
		return "Fields"
	case DetailViewEvents:
		return "Events"
	default:
		return "Unknown"
	}
}

// CRDetailModel is the model for the CR detail view
type CRDetailModel struct {
	viewport   viewport.Model
	eventTable table.Model
	fieldTable table.Model
	client     *k8s.Client
	resource   types.Resource
	events     []types.Event
	loading    bool
	err        error
	width      int
	height     int
	activeView DetailViewMode

	// Fields data
	rootFields    []ValueField
	currentFields []ValueField
	navStack      []NavState // Reusing NavState from crd_spec.go? No, let's define locally or reuse if possible.
	// NavState in crd_spec.go uses SchemaField, we need ValueField.
	// Let's define ValueNavState to avoid coupling or dependency issues if packages assume different things (though they are same package).
	// Actually they are in 'views' package so we can share if logical.
	// But ValueField != SchemaField.
	valueNavStack []ValueNavState
	currentPath   string
}

// ValueNavState represents a state in the value navigation stack
type ValueNavState struct {
	Fields    []ValueField
	Cursor    int
	TitlePath string
}

// NewCRDetailModel creates a new CR detail model
func NewCRDetailModel(client *k8s.Client, resource types.Resource, width, height int) *CRDetailModel {
	vp := viewport.New(width, height-8) // Reserve space for header/footer

	// Event Table
	eventColumns := []table.Column{
		{Title: "Type", Width: 10},
		{Title: "Reason", Width: 20},
		{Title: "Message", Width: 50},
		{Title: "Last Seen", Width: 20},
	}
	et := table.New(
		table.WithColumns(eventColumns),
		table.WithFocused(true),
		table.WithHeight(height-10),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).BorderBottom(true).Bold(false)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(false)
	et.SetStyles(s)

	// Field Table
	fieldColumns := []table.Column{
		{Title: "Field", Width: 30},
		{Title: "Value", Width: 40},
		{Title: "Type", Width: 10},
	}
	ft := table.New(
		table.WithColumns(fieldColumns),
		table.WithFocused(true),
		table.WithHeight(height-10),
	)
	ft.SetStyles(s)

	return &CRDetailModel{
		viewport:    vp,
		eventTable:  et,
		fieldTable:  ft,
		client:      client,
		resource:    resource,
		width:       width,
		height:      height,
		activeView:  DetailViewFields,
		currentPath: resource.Name,
	}
}

// Init initializes the model
func (m *CRDetailModel) Init() tea.Cmd {
	return tea.Batch(
		m.FormatYAML,
		m.FetchEvents,
		m.ParseFields,
	)
}

// HasNavigationHistory returns whether there is navigation history to go back to
func (m *CRDetailModel) HasNavigationHistory() bool {
	return m.activeView == DetailViewFields && len(m.valueNavStack) > 0
}

// Update handles messages
func (m *CRDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case FormattedYAMLMsg:
		m.viewport.SetContent(msg.YAML)

	case FetchedEventsMsg:
		m.events = msg.Events
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
		m.eventTable.SetRows(rows)

	case ParsedFieldsMsg:
		m.rootFields = msg.Fields
		m.currentFields = m.rootFields
		m.updateFieldTableRows()

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.activeView = (m.activeView + 1) % 3
			return m, nil

		case "esc", "backspace":
			if m.activeView == DetailViewFields && len(m.valueNavStack) > 0 {
				lastState := m.valueNavStack[len(m.valueNavStack)-1]
				m.valueNavStack = m.valueNavStack[:len(m.valueNavStack)-1]

				m.currentFields = lastState.Fields
				m.currentPath = lastState.TitlePath
				m.updateFieldTableRows()
				m.fieldTable.SetCursor(lastState.Cursor)
				return m, nil
			}
			// If no history, let parent handle it (return to list)

		case "enter":
			if m.activeView == DetailViewFields {
				idx := m.fieldTable.Cursor()
				if idx >= 0 && idx < len(m.currentFields) {
					selected := m.currentFields[idx]
					if len(selected.Children) > 0 {
						// Drill down
						m.valueNavStack = append(m.valueNavStack, ValueNavState{
							Fields:    m.currentFields,
							Cursor:    m.fieldTable.Cursor(),
							TitlePath: m.currentPath,
						})
						m.currentFields = selected.Children
						m.currentPath = selected.Key
						m.fieldTable.SetCursor(0)
						m.updateFieldTableRows()
					}
				}
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 8
		m.eventTable.SetHeight(msg.Height - 10)
		m.fieldTable.SetHeight(msg.Height - 10)
	}

	// Update active view component
	switch m.activeView {
	case DetailViewYAML:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	case DetailViewEvents:
		var cmd tea.Cmd
		m.eventTable, cmd = m.eventTable.Update(msg)
		cmds = append(cmds, cmd)
	case DetailViewFields:
		var cmd tea.Cmd
		m.fieldTable, cmd = m.fieldTable.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *CRDetailModel) updateFieldTableRows() {
	rows := make([]table.Row, len(m.currentFields))
	for i, field := range m.currentFields {
		rows[i] = field.TableRow()
	}
	m.fieldTable.SetRows(rows)
}

// View renders the model
func (m *CRDetailModel) View() string {
	titleText := fmt.Sprintf("CR: %s", m.resource.Name)
	if m.activeView == DetailViewFields && len(m.valueNavStack) > 0 {
		titleText = fmt.Sprintf("CR: %s", m.currentPath)
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Render(fmt.Sprintf("%s  [Tab: View (%s)] [Esc: Back] [Enter: Drill Down]", titleText, m.activeView.String()))

	var content string
	switch m.activeView {
	case DetailViewYAML:
		content = m.viewport.View()
	case DetailViewEvents:
		content = m.eventTable.View()
	case DetailViewFields:
		content = m.fieldTable.View()
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"\n",
		content,
	)
}

// Messages
type FormattedYAMLMsg struct {
	YAML string
}

type FetchedEventsMsg struct {
	Events []types.Event
}

type ParsedFieldsMsg struct {
	Fields []ValueField
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

// ParseFields parses the resource content into fields
func (m *CRDetailModel) ParseFields() tea.Msg {
	if m.resource.Raw == nil {
		return ParsedFieldsMsg{Fields: []ValueField{}}
	}
	fields := ParseValueFields(m.resource.Raw.Object, "")
	return ParsedFieldsMsg{Fields: fields}
}
