package views

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pteich/crdlens/internal/k8s"
)

// NSItem implements the list.Item interface
type NSItem string

func (i NSItem) FilterValue() string { return string(i) }
func (i NSItem) Title() string       { return string(i) }
func (i NSItem) Description() string { return "" }

// NSPickerModel is the model for the namespace picker
type NSPickerModel struct {
	list    list.Model
	client  *k8s.Client
	loading bool
	err     error
	width   int
	height  int
}

// NewNSPickerModel creates a new namespace picker model
func NewNSPickerModel(client *k8s.Client, width, height int) *NSPickerModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), width, height)
	l.Title = "Select Namespace"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	// Customize delegate to remove description padding
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	d.SetHeight(1)
	d.SetSpacing(0)
	l.SetDelegate(d)

	return &NSPickerModel{
		list:   l,
		client: client,
		width:  width,
		height: height,
	}
}

// Init initializes the model
func (m *NSPickerModel) Init() tea.Cmd {
	return m.FetchNamespaces
}

// Update handles messages
func (m *NSPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case FetchedNamespacesMsg:
		m.loading = false
		items := make([]list.Item, len(msg.Namespaces)+1)
		items[0] = NSItem("all-namespaces")
		for i, ns := range msg.Namespaces {
			items[i+1] = NSItem(ns)
		}
		return m, m.list.SetItems(items)

	case ErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if i, ok := m.list.SelectedItem().(NSItem); ok {
				return m, func() tea.Msg {
					return NamespaceSelectedMsg{Namespace: string(i)}
				}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the model as an overlay
func (m *NSPickerModel) View() string {
	if m.loading {
		return "Loading namespaces..."
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}

	// Calculate picker dimensions
	pickerWidth := 40
	if m.width < pickerWidth+4 {
		pickerWidth = m.width - 4
	}
	pickerHeight := 15
	if m.height < pickerHeight+4 {
		pickerHeight = m.height - 4
	}

	m.list.SetSize(pickerWidth, pickerHeight)

	overlay := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(1, 2).
		Width(pickerWidth + 4).
		Height(pickerHeight + 2).
		Render(m.list.View())

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#1a1a1a")),
	)
}

// NamespaceSelectedMsg is sent when a namespace is selected
type NamespaceSelectedMsg struct {
	Namespace string
}

// FetchedNamespacesMsg is sent when namespaces are fetched
type FetchedNamespacesMsg struct {
	Namespaces []string
}

// FetchNamespaces fetches all namespaces from the cluster
func (m *NSPickerModel) FetchNamespaces() tea.Msg {
	m.loading = true
	ns, err := m.client.ListNamespaces(context.Background())
	if err != nil {
		return ErrorMsg{Err: err}
	}
	return FetchedNamespacesMsg{Namespaces: ns}
}
