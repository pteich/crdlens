package views

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"sigs.k8s.io/yaml"

	"github.com/pteich/crdlens/internal/k8s"
	"github.com/pteich/crdlens/internal/types"
)

type DetailViewMode int

const (
	DetailViewYAML DetailViewMode = iota
	DetailViewFields
	DetailViewEvents
	DetailViewReconcile
)

func (m DetailViewMode) String() string {
	switch m {
	case DetailViewYAML:
		return "YAML"
	case DetailViewFields:
		return "Fields"
	case DetailViewEvents:
		return "Events"
	case DetailViewReconcile:
		return "Reconcile Status"
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

	reconcileTable table.Model
	reconcileFocus int // 0: Conditions, 1: Status

	// Status fields data
	statusTable         table.Model
	rootStatusFields    []ValueField
	currentStatusFields []ValueField
	statusNavStack      []ValueNavState

	// Fields data
	rootFields    []ValueField
	currentFields []ValueField
	navStack      []NavState
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

	// Status Table
	st := table.New(
		table.WithColumns(fieldColumns),
		table.WithFocused(false),
		table.WithHeight(height/2),
	)
	st.SetStyles(s)

	m := &CRDetailModel{
		viewport:    vp,
		eventTable:  et,
		fieldTable:  ft,
		statusTable: st,
		client:      client,
		resource:    resource,
		width:       width,
		height:      height,
		activeView:  DetailViewReconcile,
		currentPath: resource.Name,
	}
	m.initReconcileTable()
	return m
}

func (m *CRDetailModel) initReconcileTable() {
	columns := []table.Column{
		{Title: "Type", Width: 20},
		{Title: "Status", Width: 10},
		{Title: "Reason", Width: 20},
		{Title: "Age", Width: 15},
		{Title: "Message", Width: 40},
	}
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10), // Fixed height for conditions
	)
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).BorderBottom(true).Bold(false)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(false)
	t.SetStyles(s)
	m.reconcileTable = t
	m.updateReconcileTableRows()
}

func (m *CRDetailModel) updateReconcileTableRows() {
	rows := make([]table.Row, len(m.resource.Conditions))
	for i, cond := range m.resource.Conditions {
		age := "-"
		if !cond.LastTransitionTime.IsZero() {
			age = time.Since(cond.LastTransitionTime).Round(time.Second).String()
		}
		rows[i] = table.Row{
			cond.Type,
			cond.Status,
			cond.Reason,
			age,
			cond.Message,
		}
	}
	m.reconcileTable.SetRows(rows)
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
	if m.activeView == DetailViewFields {
		return len(m.valueNavStack) > 0
	}
	if m.activeView == DetailViewReconcile && m.reconcileFocus == 1 {
		return len(m.statusNavStack) > 0
	}
	return false
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

		m.rootStatusFields = msg.StatusFields
		m.currentStatusFields = m.rootStatusFields
		m.updateStatusTableRows()

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.activeView = (m.activeView + 1) % 4
			if m.activeView == DetailViewReconcile && m.reconcileTable.Rows() == nil {
				m.initReconcileTable()
			}
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
			} else if m.activeView == DetailViewReconcile && m.reconcileFocus == 1 && len(m.statusNavStack) > 0 {
				lastState := m.statusNavStack[len(m.statusNavStack)-1]
				m.statusNavStack = m.statusNavStack[:len(m.statusNavStack)-1]

				m.currentStatusFields = lastState.Fields
				m.updateStatusTableRows()
				m.statusTable.SetCursor(lastState.Cursor)
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
			} else if m.activeView == DetailViewReconcile && m.reconcileFocus == 1 {
				idx := m.statusTable.Cursor()
				if idx >= 0 && idx < len(m.currentStatusFields) {
					selected := m.currentStatusFields[idx]
					if len(selected.Children) > 0 {
						// Drill down status
						m.statusNavStack = append(m.statusNavStack, ValueNavState{
							Fields: m.currentStatusFields,
							Cursor: m.statusTable.Cursor(),
						})
						m.currentStatusFields = selected.Children
						m.statusTable.SetCursor(0)
						m.updateStatusTableRows()
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

		// Split view resizing
		condHeight := 10
		statusHeight := msg.Height - condHeight - 12 // Reserve space for headers/summary
		if statusHeight < 5 {
			statusHeight = 5
		}
		m.reconcileTable.SetHeight(condHeight)
		m.statusTable.SetHeight(statusHeight)
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
	case DetailViewReconcile:
		// Focus management logic
		switch m.reconcileFocus {
		case 0: // Conditions table focused
			if k, ok := msg.(tea.KeyMsg); ok && k.String() == "down" {
				if m.reconcileTable.Cursor() == len(m.reconcileTable.Rows())-1 {
					m.reconcileFocus = 1
					m.reconcileTable.Blur()
					m.statusTable.Focus()
					return m, nil
				}
			}
			var cmd tea.Cmd
			m.reconcileTable, cmd = m.reconcileTable.Update(msg)
			cmds = append(cmds, cmd)

		case 1: // Status table focused
			if k, ok := msg.(tea.KeyMsg); ok && k.String() == "up" {
				if m.statusTable.Cursor() == 0 && len(m.statusNavStack) == 0 {
					m.reconcileFocus = 0
					m.statusTable.Blur()
					m.reconcileTable.Focus()
					return m, nil
				}
			}
			var cmd tea.Cmd
			m.statusTable, cmd = m.statusTable.Update(msg)
			cmds = append(cmds, cmd)
		}
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

func (m *CRDetailModel) updateStatusTableRows() {
	rows := make([]table.Row, len(m.currentStatusFields))
	for i, field := range m.currentStatusFields {
		rows[i] = field.TableRow()
	}
	m.statusTable.SetRows(rows)
}

// View renders the model
func (m *CRDetailModel) View() string {
	titleText := fmt.Sprintf("CR: %s", m.resource.Name)
	if m.activeView == DetailViewFields && len(m.valueNavStack) > 0 {
		titleText = fmt.Sprintf("CR: %s", m.currentPath)
	} else if m.activeView == DetailViewReconcile && m.reconcileFocus == 1 && len(m.statusNavStack) > 0 {
		// Show current path for status drill down too
		if len(m.statusNavStack) > 0 {
			// We don't track title path for status stack yet, but we can show generic drill down info or improve ValueNavState
			// For now let's just show "CR: Status" or keep generic
		}
	}

	helpText := fmt.Sprintf("[Tab: View (%s)] [Esc: Back] [Enter: Drill Down]", m.activeView.String())
	if m.activeView == DetailViewReconcile {
		helpText += " [↑/↓: Switch]"
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Render(fmt.Sprintf("%s  %s", titleText, helpText))

	var content string
	switch m.activeView {
	case DetailViewYAML:
		content = m.viewport.View()
	case DetailViewEvents:
		content = m.eventTable.View()
	case DetailViewFields:
		content = m.fieldTable.View()
	case DetailViewReconcile:
		content = m.renderReconcileView()
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"\n",
		content,
	)
}

func (m *CRDetailModel) renderReconcileView() string {
	res := m.resource

	// Summary Info
	driftText := fmt.Sprintf("Drift: %d (Gen: %d / Obs: %d)", res.Drift(), res.Generation, res.ObservedGeneration)
	lagText := fmt.Sprintf("Lag: %v", res.Lag().Round(time.Second))
	silenceText := fmt.Sprintf("Silence: %v", res.Silence().Round(time.Second))
	controllerText := fmt.Sprintf("Controller: %s", res.ControllerManager)

	summaryStyle := lipgloss.NewStyle().Margin(0, 0, 1, 0)
	infoLine := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(driftText),
		"  ",
		lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render(lagText),
		"  ",
		lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Render(silenceText),
		"  ",
		lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(controllerText),
	)

	fieldsStyle := lipgloss.NewStyle().Margin(1, 0, 0, 0)
	fieldsText := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Additional Status Fields:")
	
	// Add focus indication
	statusTableStyle := lipgloss.NewStyle()
	if m.reconcileFocus == 1 {
		statusTableStyle = statusTableStyle.Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("62")) // Highlight
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		summaryStyle.Render(infoLine),
		m.reconcileTable.View(),
		fieldsStyle.Render(fieldsText),
		statusTableStyle.Render(m.statusTable.View()),
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
	Fields       []ValueField
	StatusFields []ValueField
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
		return ParsedFieldsMsg{Fields: []ValueField{}, StatusFields: []ValueField{}}
	}

	// Parse all fields
	fields := ParseValueFields(m.resource.Raw.Object, "")

	// Parse status fields, excluding conditions
	var statusFields []ValueField
	if status, ok := m.resource.Raw.Object["status"].(map[string]interface{}); ok {
		statusCopy := make(map[string]interface{})
		for k, v := range status {
			if k != "conditions" {
				statusCopy[k] = v
			}
		}
		statusFields = ParseValueFields(statusCopy, "status")
	}

	return ParsedFieldsMsg{Fields: fields, StatusFields: statusFields}
}
