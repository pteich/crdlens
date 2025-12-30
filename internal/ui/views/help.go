package views

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// keyMap defines the keybindings for the application help
type keyMap struct {
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	Enter     key.Binding
	Esc       key.Binding
	Help      key.Binding
	Quit      key.Binding
	Filter    key.Binding
	Namespace key.Binding
	Refresh   key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings to be shown in the expanded help view
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},          // first column
		{k.Enter, k.Esc, k.Filter},               // second column
		{k.Namespace, k.Refresh, k.Help, k.Quit}, // third column
	}
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "move left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "move right"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Esc: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
	Namespace: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "toggle namespace"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
}

// HelpModel is the model for the help view
type HelpModel struct {
	help  help.Model
	keys  keyMap
	width int
}

// NewHelpModel creates a new help model
func NewHelpModel() *HelpModel {
	return &HelpModel{
		help: help.New(),
		keys: keys,
	}
}

// Init initializes the model
func (m *HelpModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *HelpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return m, nil
}

// View renders the help view
func (m *HelpModel) View() string {
	helpView := m.help.FullHelpView(m.keys.FullHelp())

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render(helpView)
}
