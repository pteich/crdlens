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
	state     ViewState
	prevState ViewState
	config    *config.Config
	client    *k8s.Client
	width     int
	height    int
	err       error
	ready     bool

	crdList  *views.CRDListModel
	crList   *views.CRListModel
	crDetail *views.CRDetailModel
	crdSpec  *views.CRDSpecModel
	nsPicker *views.NSPickerModel
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

	// Calculate filtering state for all models
	isFiltering := false
	if m.crdList != nil && m.crdList.IsFiltering() {
		isFiltering = true
	} else if m.crList != nil && m.crList.IsFiltering() {
		isFiltering = true
	}

	switch msg := msg.(type) {
	case views.NamespaceSelectedMsg:
		m.state = m.prevState
		if msg.Namespace == "all-namespaces" {
			m.config.AllNamespaces = true
			m.client.Namespace = ""
		} else {
			m.config.AllNamespaces = false
			m.client.Namespace = msg.Namespace
		}
		if m.crList != nil {
			ns := m.client.Namespace
			cmds = append(cmds, m.crList.Refresh(ns))
		}
		if m.crdList != nil {
			cmds = append(cmds, m.crdList.Refresh(msg.Namespace))
		}
		return m, tea.Batch(cmds...)

	case views.SwitchToAllNamespacesMsg:
		m.config.AllNamespaces = true
		m.client.Namespace = ""
		if m.crList != nil {
			cmds = append(cmds, m.crList.Refresh(""))
		}
		if m.crdList != nil {
			cmds = append(cmds, m.crdList.Refresh("all-namespaces"))
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		if !isFiltering && m.state != NSPickerView {
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "?":
				m.showHelp = !m.showHelp
				return m, nil
			case "n":
				m.prevState = m.state
				m.state = NSPickerView
				m.nsPicker = views.NewNSPickerModel(m.client, m.width, m.height)
				return m, m.nsPicker.Init()
			case "r":
				switch m.state {
				case CRDListView:
					if m.crdList != nil {
						ns := m.client.Namespace
						if m.config.AllNamespaces {
							ns = "all-namespaces"
						}
						return m, m.crdList.Refresh(ns)
					}
				case CRListView:
					if m.crList != nil {
						ns := m.client.Namespace
						if m.config.AllNamespaces {
							ns = ""
						}
						return m, m.crList.Refresh(ns)
					}
				}
			case "enter":
				switch m.state {
				case CRDListView:
					if m.crdList != nil && !m.crdList.IsFiltering() {
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
					}
				case CRListView:
					if m.crList != nil {
						selected := m.crList.SelectedResource()
						if selected.Name != "" {
							m.state = CRDetailView
							m.crDetail = views.NewCRDetailModel(m.client, selected, m.width, m.height)
							return m, m.crDetail.Init()
						}
					}
				}
			case "s":
				switch m.state {
				case CRDListView:
					if m.crdList != nil && !m.crdList.IsFiltering() {
						selected := m.crdList.SelectedCRD()
						if selected.Name != "" {
							m.state = CRDSpecView
							m.crdSpec = views.NewCRDSpecModel(m.client, selected, m.width, m.height)
							return m, m.crdSpec.Init()
						}
					}
				}
			case "esc":
				switch m.state {
				case CRListView:
					m.state = CRDListView
					return m, nil
				case CRDetailView:
					// Check if we have navigation history in the detail view
					if m.crDetail != nil && m.crDetail.HasNavigationHistory() {
						newModel, cmd := m.crDetail.Update(msg)
						m.crDetail = newModel.(*views.CRDetailModel)
						return m, cmd
					}
					m.state = CRListView
					return m, nil
				case CRDSpecView:
					// Check if we are showing field details or have navigation history
					if m.crdSpec != nil && (m.crdSpec.IsShowingFieldDetail() || m.crdSpec.HasNavigationHistory()) {
						// Pass the message to the view
						newModel, cmd := m.crdSpec.Update(msg)
						m.crdSpec = newModel.(*views.CRDSpecModel)
						return m, cmd
					}
					m.state = CRDListView
					return m, nil
				}
			}
		} else if m.state == NSPickerView && msg.String() == "esc" {
			m.state = m.prevState
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if m.crdList == nil {
			ns := m.client.Namespace
			if m.config.AllNamespaces {
				ns = "all-namespaces"
			}
			m.crdList = views.NewCRDListModel(m.client, ns, m.width, m.height, m.config.DisableCounts)
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
	}
	m.ready = true

	// KeyMsg should ONLY go to the active view to avoid duplicate handling or interference
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if !isFiltering && m.state != NSPickerView {
			// Global keys are already handled above in the main switch
		}

		switch m.state {
		case CRDListView:
			if m.crdList != nil {
				newModel, cmd := m.crdList.Update(keyMsg)
				m.crdList = newModel.(*views.CRDListModel)
				cmds = append(cmds, cmd)
			}
		case CRListView:
			if m.crList != nil {
				newModel, cmd := m.crList.Update(keyMsg)
				m.crList = newModel.(*views.CRListModel)
				cmds = append(cmds, cmd)
			}
		case CRDetailView:
			if m.crDetail != nil {
				newModel, cmd := m.crDetail.Update(keyMsg)
				m.crDetail = newModel.(*views.CRDetailModel)
				cmds = append(cmds, cmd)
			}
		case CRDSpecView:
			if m.crdSpec != nil {
				newModel, cmd := m.crdSpec.Update(keyMsg)
				m.crdSpec = newModel.(*views.CRDSpecModel)
				cmds = append(cmds, cmd)
			}
		case NSPickerView:
			if m.nsPicker != nil {
				newModel, cmd := m.nsPicker.Update(keyMsg)
				m.nsPicker = newModel.(*views.NSPickerModel)
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
	}

	// Non-KeyMsg (background responses, timer ticks, etc.) should go to ALL models
	// This ensures they stay in sync and don't get stuck in loading states
	if m.crdList != nil {
		newModel, cmd := m.crdList.Update(msg)
		m.crdList = newModel.(*views.CRDListModel)
		cmds = append(cmds, cmd)
	}

	if m.crList != nil {
		newModel, cmd := m.crList.Update(msg)
		m.crList = newModel.(*views.CRListModel)
		cmds = append(cmds, cmd)
	}

	if m.crDetail != nil {
		newModel, cmd := m.crDetail.Update(msg)
		m.crDetail = newModel.(*views.CRDetailModel)
		cmds = append(cmds, cmd)
	}

	if m.crdSpec != nil {
		newModel, cmd := m.crdSpec.Update(msg)
		m.crdSpec = newModel.(*views.CRDSpecModel)
		cmds = append(cmds, cmd)
	}

	if m.nsPicker != nil {
		newModel, cmd := m.nsPicker.Update(msg)
		m.nsPicker = newModel.(*views.NSPickerModel)
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
	case CRDListView, NSPickerView:
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
	case CRDSpecView:
		if m.crdSpec != nil {
			view = m.crdSpec.View()
		} else {
			view = fmt.Sprintf("\n %s Loading CRD Spec...", m.spinner.View())
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

	if m.state == NSPickerView && m.nsPicker != nil {
		view = m.nsPicker.View()
	}

	if m.showHelp {
		return lipgloss.JoinVertical(lipgloss.Left,
			view,
			"\n",
			m.help.View(),
		)
	}

	return AppStyle.Render(view)
}
