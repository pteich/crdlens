package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	crdList  *views.CRDListModel
	crList   *views.CRListModel
	crDetail *views.CRDetailModel
	help     *views.HelpModel
	showHelp bool
	spinner  spinner.Model
}

// NewModel creates a new root model
func NewModel(cfg *config.Config, client *k8s.Client) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	return Model{
		state:   CRDListView,
		config:  cfg,
		client:  client,
		help:    views.NewHelpModel(),
		spinner: s,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	var spinnerCmd tea.Cmd
	m.spinner, spinnerCmd = m.spinner.Update(msg)
	cmds = append(cmds, spinnerCmd)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Skip global key handling when any view is in filtering mode
		isFiltering := false
		if m.state == CRDListView && m.crdList != nil && m.crdList.IsFiltering() {
			isFiltering = true
		} else if m.state == CRListView && m.crList != nil && m.crList.IsFiltering() {
			isFiltering = true
		}

		if !isFiltering {
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "?":
				m.showHelp = !m.showHelp
				return m, nil
			case "n":
				m.config.AllNamespaces = !m.config.AllNamespaces
				ns := m.client.Namespace
				if m.config.AllNamespaces {
					ns = ""
				}
				if m.state == CRListView && m.crList != nil {
					return m, m.crList.Refresh(ns)
				}
				return m, nil
			case "r":
				if m.state == CRDListView && m.crdList != nil {
					return m, m.crdList.Refresh()
				} else if m.state == CRListView && m.crList != nil {
					ns := m.client.Namespace
					if m.config.AllNamespaces {
						ns = ""
					}
					return m, m.crList.Refresh(ns)
				}
			case "enter":
				if m.state == CRDListView && m.crdList != nil && !m.crdList.IsFiltering() {
					selected := m.crdList.SelectedCRD()
					if selected.Name != "" {
						m.state = CRListView
						ns := m.client.Namespace
						if m.config.AllNamespaces {
							ns = ""
						}
						m.crList = views.NewCRListModel(m.client, selected, ns, m.width, m.height)
						return m, m.crList.Init()
					}
				} else if m.state == CRListView && m.crList != nil {
					selected := m.crList.SelectedResource()
					if selected.Name != "" {
						m.state = CRDetailView
						m.crDetail = views.NewCRDetailModel(m.client, selected, m.width, m.height)
						return m, m.crDetail.Init()
					}
				}
			case "esc":
				if m.state == CRListView {
					m.state = CRDListView
					return m, nil
				} else if m.state == CRDetailView {
					m.state = CRListView
					return m, nil
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if m.crdList == nil {
			m.crdList = views.NewCRDListModel(m.client, m.width, m.height)
			cmds = append(cmds, m.crdList.Init())
		} else {
			if m.crdList != nil {
				// We'll need to add a SetSize method to CRDListModel or handle it in Update
				// For now bubbles list handles WindowSizeMsg if passed
			}
			if m.crList != nil {
				// Handle resize for CR list
			}
			if m.crDetail != nil {
				// Handle resize for CR detail
			}
		}

		m.ready = true
	}

	if m.crdList != nil && m.state == CRDListView {
		newModel, cmd := m.crdList.Update(msg)
		m.crdList = newModel.(*views.CRDListModel)
		cmds = append(cmds, cmd)
	}

	if m.crList != nil && m.state == CRListView {
		newModel, cmd := m.crList.Update(msg)
		m.crList = newModel.(*views.CRListModel)
		cmds = append(cmds, cmd)
	}

	if m.crDetail != nil && m.state == CRDetailView {
		newModel, cmd := m.crDetail.Update(msg)
		m.crDetail = newModel.(*views.CRDetailModel)
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
	case CRListView:
		if m.crList != nil {
			view = m.crList.View()
		} else {
			view = "Loading CR List..."
		}
	case CRDetailView:
		if m.crDetail != nil {
			view = m.crDetail.View()
		} else {
			view = fmt.Sprintf("\n %s Loading CR Detail...", m.spinner.View())
		}
	default:
		view = "Unknown View"
	}

	nsText := m.client.Namespace
	if m.config.AllNamespaces {
		nsText = "all-namespaces"
	}

	statusBar := lipgloss.JoinHorizontal(lipgloss.Top,
		StatusBarMainStyle.Render(fmt.Sprintf("Context: %s", m.client.Context)),
		StatusBarExtraStyle.Render(fmt.Sprintf("Namespace: %s", nsText)),
	)

	view = lipgloss.JoinVertical(lipgloss.Left,
		view,
		"\n",
		statusBar,
	)

	if m.showHelp {
		return lipgloss.JoinVertical(lipgloss.Left,
			view,
			"\n",
			m.help.View(),
		)
	}

	return AppStyle.Render(view)
}
