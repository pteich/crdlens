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

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// CRDListModel is the model for the CRD list view
type CRDListModel struct {
	table        table.Model
	client       *k8s.Client
	loading      bool
	err          error
	allCRDs      []types.CRDInfo
	filtered     []types.CRDInfo
	textinput    textinput.Model
	filtering    bool
	spinner      spinner.Model
	width        int
	height       int
	countsLoaded bool
	tickCount    int
}

// NewCRDListModel creates a new CRD list model
func NewCRDListModel(client *k8s.Client, width, height int) *CRDListModel {
	columns := []table.Column{
		{Title: "Name", Width: 40},
		{Title: "API Group", Width: 40},
		{Title: "CR Count", Width: 10},
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
	ti.Placeholder = "Search CRDs..."
	ti.Prompt = "/ "

	spn := spinner.New()
	spn.Spinner = spinner.Dot
	spn.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#F780E2"))

	return &CRDListModel{
		table:     t,
		client:    client,
		textinput: ti,
		spinner:   spn,
		loading:   true,
		width:     width,
		height:    height,
	}
}

// Init initializes the model
func (m *CRDListModel) Init() tea.Cmd {
	return tea.Batch(m.FetchCRDs, m.spinner.Tick)
}

var asciiSpinner = []string{"|", "/", "-", "\\"}

func (m *CRDListModel) renderRows() {
	rows := make([]table.Row, len(m.filtered))
	frame := m.tickCount % 4
	spinnerChar := asciiSpinner[frame]

	for i, crd := range m.filtered {
		countStr := spinnerChar
		if m.countsLoaded {
			countStr = fmt.Sprintf("%d", crd.Count)
		}
		rows[i] = table.Row{
			crd.Kind,
			crd.Group,
			countStr,
		}
	}
	m.table.SetRows(rows)
}

// Update handles messages
func (m *CRDListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case FetchedCRDsMsg:
		m.loading = false
		m.countsLoaded = false
		m.allCRDs = msg.CRDs
		m.filtered = msg.CRDs
		m.renderRows()
		return m, m.FetchCRDCounts

	case CRDCountsMsg:
		for i, crd := range m.allCRDs {
			if count, ok := msg.Counts[crd.Name]; ok {
				m.allCRDs[i].Count = count
			}
		}
		for i, crd := range m.filtered {
			if count, ok := msg.Counts[crd.Name]; ok {
				m.filtered[i].Count = count
			}
		}
		m.countsLoaded = true
		m.renderRows()
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
			// Handle filtering immediately within KeyMsg case to prevent list interference
			var cmd tea.Cmd
			m.textinput, cmd = m.textinput.Update(msg)

			m.filtered = search.MatchCRDs(m.textinput.Value(), m.allCRDs)
			m.renderRows()
			return m, cmd
		} else {
			switch msg.String() {
			case "/":
				m.filtering = true
				m.textinput.Focus()
				return m, tea.Batch(textinput.Blink)
			}
		}
	}

	// Only reached when NOT filtering
	var sCmd tea.Cmd
	m.spinner, sCmd = m.spinner.Update(msg)

	if _, ok := msg.(spinner.TickMsg); ok && !m.countsLoaded {
		m.tickCount++
		m.renderRows()
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, tea.Batch(cmd, sCmd)
}

// View renders the model
func (m *CRDListModel) View() string {
	if m.loading {
		return fmt.Sprintf("\n %s Loading CRDs...", m.spinner.View())
	}
	if m.err != nil {
		return fmt.Sprintf("Error fetching CRDs: %v", m.err)
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Render("Custom Resource Definitions")

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

// SelectedCRD returns the currently selected CRDInfo
func (m *CRDListModel) SelectedCRD() types.CRDInfo {
	idx := m.table.Cursor()
	if idx >= 0 && idx < len(m.filtered) {
		return m.filtered[idx]
	}
	return types.CRDInfo{}
}

// IsFiltering returns true if the list is currently filtering
func (m *CRDListModel) IsFiltering() bool {
	return m.filtering
}

// Refresh re-fetches the CRDs
func (m *CRDListModel) Refresh() tea.Cmd {
	return m.FetchCRDs
}

// Messages
type FetchedCRDsMsg struct {
	CRDs []types.CRDInfo
}

type ErrorMsg struct {
	Err error
}

type CRDCountsMsg struct {
	Counts map[string]int
}

// FetchCRDs is a command to fetch CRDs from the cluster
func (m *CRDListModel) FetchCRDs() tea.Msg {
	m.loading = true
	discoverySvc := m.client.Discovery()
	crds, err := discoverySvc.ListCRDs(context.Background())
	if err != nil {
		return ErrorMsg{Err: err}
	}
	return FetchedCRDsMsg{CRDs: crds}
}

// FetchCRDCounts is a command to fetch counts for all CRDs (async)
func (m *CRDListModel) FetchCRDCounts() tea.Msg {
	dynamicSvc := m.client.Dynamic()
	namespace := ""
	counts := make(map[string]int)

	for _, crd := range m.allCRDs {
		count, err := dynamicSvc.CountResources(context.Background(), crd.GVR, namespace)
		if err != nil {
			continue
		}
		counts[crd.Name] = count
	}

	return CRDCountsMsg{Counts: counts}
}
