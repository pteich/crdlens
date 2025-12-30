package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pteich/crdlens/internal/config"
	"github.com/pteich/crdlens/internal/k8s"
	"github.com/pteich/crdlens/internal/ui/views"
)

// Model is the root model for the CRDLens TUI
type Model struct {
	state  ViewState
	config *config.Config
	client *k8s.Client
	width  int
	height int
	err    error
	ready  bool

	crdList *views.CRDListModel
}

// NewModel creates a new root model
func NewModel(cfg *config.Config, client *k8s.Client) Model {
	return Model{
		state:  CRDListView,
		config: cfg,
		client: client,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if m.crdList == nil {
			m.crdList = views.NewCRDListModel(m.client, m.width, m.height)
			cmds = append(cmds, m.crdList.Init())
		} else {
			// Update size of child models
			// We'll need to add a SetSize method to CRDListModel or handle it in Update
		}

		m.ready = true
	}

	if m.crdList != nil && m.state == CRDListView {
		newModel, cmd := m.crdList.Update(msg)
		m.crdList = newModel.(*views.CRDListModel)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the model
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}

	var view string
	switch m.state {
	case CRDListView:
		if m.crdList != nil {
			view = m.crdList.View()
		} else {
			view = "Loading CRD List..."
		}
	default:
		view = "Unknown View"
	}

	return AppStyle.Render(view)
}
