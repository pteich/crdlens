package views

import (
	"context"
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pteich/crdlens/internal/k8s"
	"github.com/pteich/crdlens/internal/search"
	"github.com/pteich/crdlens/internal/types"
)

// SortMode defines how resources are sorted
type SortMode int

const (
	SortByName SortMode = iota
	SortByDrift
	SortByCreated
	SortByStatus
)

// String returns a human-readable sort mode name
func (s SortMode) String() string {
	switch s {
	case SortByName:
		return "Name"
	case SortByDrift:
		return "Drift"
	case SortByCreated:
		return "Created"
	case SortByStatus:
		return "Status"
	default:
		return "Unknown"
	}
}

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

	// New: Sorting and pagination
	sortMode      SortMode
	sortAsc       bool
	showSortMenu  bool
	continueToken string
	hasMorePages  bool
	totalShown    int

	// Empty namespace dialog
	showDialog        bool
	dialogSelectedYes bool
}

// NewCRListModel creates a new CR list model
func NewCRListModel(client *k8s.Client, crd types.CRDInfo, namespace string, width, height int) *CRListModel {
	columns := []table.Column{
		{Title: "R", Width: 2},        // Ready icon
		{Title: "Status", Width: 8},   // Ready status
		{Title: "Name", Width: 40},    // Resource name (wider)
		{Title: "NS", Width: 20},      // Namespace
		{Title: "Drift", Width: 6},    // Generation drift
		{Title: "Ctrl", Width: 15},    // Controller manager (wider)
		{Title: "Created", Width: 16}, // Creation date
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
		sortMode:  SortByName,
		sortAsc:   true,
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
		m.allResources = msg.Resources
		m.continueToken = msg.ContinueToken
		m.hasMorePages = msg.ContinueToken != ""
		m.totalShown = len(msg.Resources)
		m.filtered = m.allResources
		m.sortResources()
		m.updateTableRows()

		// Show dialog if no resources found and not in all-namespaces mode
		if len(m.filtered) == 0 && m.namespace != "" && m.namespace != "all-namespaces" {
			m.showDialog = true
		}
		return m, nil

	case FetchedMoreCRsMsg:
		m.loading = false
		m.allResources = append(m.allResources, msg.Resources...)
		m.continueToken = msg.ContinueToken
		m.hasMorePages = msg.ContinueToken != ""
		m.totalShown = len(m.allResources)
		m.filtered = search.MatchResources(m.textinput.Value(), m.allResources)
		m.sortResources()
		m.updateTableRows()
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
		// Handle empty namespace dialog
		if m.showDialog {
			switch msg.String() {
			case "left", "right", "tab":
				m.dialogSelectedYes = !m.dialogSelectedYes
				return m, nil
			case "enter":
				if m.dialogSelectedYes {
					m.showDialog = false
					return m, func() tea.Msg {
						return SwitchToAllNamespacesMsg{}
					}
				}
				m.showDialog = false
				return m, nil
			case "esc":
				m.showDialog = false
				return m, nil
			}
			return m, nil
		}

		// Handle sort menu
		if m.showSortMenu {
			switch msg.String() {
			case "1":
				m.sortMode = SortByDrift
				m.sortAsc = false // Highest drift first
				m.showSortMenu = false
				m.sortResources()
				m.updateTableRows()
				return m, nil
			case "2":
				m.sortMode = SortByCreated
				m.sortAsc = false // Newest first
				m.showSortMenu = false
				m.sortResources()
				m.updateTableRows()
				return m, nil
			case "3":
				m.sortMode = SortByStatus
				m.sortAsc = true
				m.showSortMenu = false
				m.sortResources()
				m.updateTableRows()
				return m, nil
			case "4":
				m.sortMode = SortByName
				m.sortAsc = true
				m.showSortMenu = false
				m.sortResources()
				m.updateTableRows()
				return m, nil
			case "esc":
				m.showSortMenu = false
				return m, nil
			}
		}

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
			case "s":
				m.showSortMenu = !m.showSortMenu
				return m, nil
			}
		}
	}

	if m.filtering {
		var cmd tea.Cmd
		m.textinput, cmd = m.textinput.Update(msg)

		// Filter resources based on query
		m.filtered = search.MatchResources(m.textinput.Value(), m.allResources)
		m.sortResources()
		m.updateTableRows()
		return m, cmd
	}

	var sCmd tea.Cmd
	m.spinner, sCmd = m.spinner.Update(msg)

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)

	// Lazy load more CRs when scrolling near the end
	if m.hasMorePages && !m.loading && !m.filtering {
		cursor := m.table.Cursor()
		rows := m.table.Rows()
		threshold := 10 // Start loading when within 10 rows of the end
		if cursor >= len(rows)-threshold {
			m.loading = true
			cmd = tea.Batch(cmd, m.FetchMoreCRs)
		}
	}

	return m, tea.Batch(cmd, sCmd)
}

// sortResources sorts the filtered resources based on current sort mode
func (m *CRListModel) sortResources() {
	sort.Slice(m.filtered, func(i, j int) bool {
		var less bool
		switch m.sortMode {
		case SortByDrift:
			less = m.filtered[i].Drift() < m.filtered[j].Drift()
		case SortByCreated:
			less = m.filtered[i].CreatedAt.Before(m.filtered[j].CreatedAt)
		case SortByStatus:
			less = m.filtered[i].ReadyStatus() < m.filtered[j].ReadyStatus()
		default: // SortByName
			less = m.filtered[i].Name < m.filtered[j].Name
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
}

// updateTableRows updates the table with current filtered/sorted resources
func (m *CRListModel) updateTableRows() {
	rows := make([]table.Row, len(m.filtered))
	for i, res := range m.filtered {
		rows[i] = m.resourceToRow(res)
	}
	m.table.SetRows(rows)
}

// resourceToRow converts a Resource to a table row with controller-aware columns
func (m *CRListModel) resourceToRow(res types.Resource) table.Row {
	// Format drift
	drift := "-"
	if res.Generation > 0 {
		d := res.Drift()
		if d > 0 {
			drift = fmt.Sprintf("+%d", d)
		} else {
			drift = "0"
		}
	}

	// Shorten namespace
	ns := res.Namespace
	if len(ns) > 15 {
		ns = ns[:12] + "..."
	}

	// Shorten controller manager
	ctrl := k8s.ShortenManagerName(res.ControllerManager)

	// Format created date
	created := res.CreatedAt.Format("2006-01-02 15:04")

	return table.Row{
		res.ReadyIcon(),
		res.ReadyStatus(),
		res.Name,
		ns,
		drift,
		ctrl,
		created,
	}
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
	if m.loading && len(m.allResources) == 0 {
		return fmt.Sprintf("\n %s Loading Custom Resources...", m.spinner.View())
	}
	if m.err != nil {
		return fmt.Sprintf("Error fetching CRs: %v", m.err)
	}

	// Title with count and sort info
	countInfo := fmt.Sprintf("%d", len(m.filtered))
	if m.hasMorePages {
		countInfo += "+"
	}

	loadingIndicator := ""
	if m.loading {
		loadingIndicator = fmt.Sprintf(" %s", m.spinner.View())
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Render(fmt.Sprintf("%s (%s) [Sort: %s]%s", m.crd.Kind, countInfo, m.sortMode.String(), loadingIndicator))

	view := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"\n",
		m.table.View(),
	)

	// Show sort menu if active
	if m.showSortMenu {
		sortMenu := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Render("Sort by:\n1) Drift ↓\n2) Created ↓\n3) Status\n4) Name\n[Esc] Cancel")
		view = lipgloss.JoinVertical(lipgloss.Left, view, "\n", sortMenu)
	}

	// Show empty namespace dialog if active
	if m.showDialog {
		yesStyle := lipgloss.NewStyle()
		noStyle := lipgloss.NewStyle()
		if m.dialogSelectedYes {
			yesStyle = yesStyle.
				Foreground(lipgloss.Color("229")).
				Background(lipgloss.Color("57"))
		} else {
			noStyle = noStyle.
				Foreground(lipgloss.Color("229")).
				Background(lipgloss.Color("57"))
		}

		dialog := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2).
			Width(50).
			Render(
				"No CRs found in current namespace.\n\n" +
					"Switch to all-namespaces?\n\n" +
					"  " + yesStyle.Render("[ Yes ]") + "  " + noStyle.Render("[ No ]") +
					"\n\n[←/→] Select  [Enter] Confirm  [Esc] Cancel",
			)

		view = lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			dialog,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("#1a1a1a")),
		)
	}

	if m.filtering {
		view = lipgloss.JoinVertical(lipgloss.Left,
			view,
			"\n",
			m.textinput.View(),
		)
	}

	// Footer with keybindings
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("[/] Search  [s] Sort  [Enter] Details  [Esc] Back")
	view = lipgloss.JoinVertical(lipgloss.Left, view, "\n", footer)

	return view
}

// Refresh re-fetches the resources
func (m *CRListModel) Refresh(namespace string) tea.Cmd {
	m.namespace = namespace
	m.continueToken = "" // Reset pagination
	return m.FetchCRs
}

// IsFiltering returns true if the list is currently filtering
func (m *CRListModel) IsFiltering() bool {
	return m.filtering
}

// FetchedCRsMsg is sent when CRs are successfully fetched
type FetchedCRsMsg struct {
	Resources     []types.Resource
	ContinueToken string
}

// FetchedMoreCRsMsg is sent when additional CRs are fetched (pagination)
type FetchedMoreCRsMsg struct {
	Resources     []types.Resource
	ContinueToken string
}

// FetchCRs is a command to fetch CRs from the cluster
func (m *CRListModel) FetchCRs() tea.Msg {
	dynamicSvc := m.client.Dynamic()
	result, err := dynamicSvc.ListResourcesPaginated(context.Background(), m.crd.GVR, m.namespace, k8s.ListResourcesOptions{
		Limit: 100,
	})
	if err != nil {
		return ErrorMsg{Err: err}
	}
	return FetchedCRsMsg{
		Resources:     result.Resources,
		ContinueToken: result.ContinueToken,
	}
}

// FetchMoreCRs fetches the next page of resources
func (m *CRListModel) FetchMoreCRs() tea.Msg {
	if m.continueToken == "" {
		return nil
	}

	dynamicSvc := m.client.Dynamic()
	result, err := dynamicSvc.ListResourcesPaginated(context.Background(), m.crd.GVR, m.namespace, k8s.ListResourcesOptions{
		Limit:    100,
		Continue: m.continueToken,
	})
	if err != nil {
		return ErrorMsg{Err: err}
	}
	return FetchedMoreCRsMsg{
		Resources:     result.Resources,
		ContinueToken: result.ContinueToken,
	}
}

// SwitchToAllNamespacesMsg is sent when user wants to switch to all namespaces
type SwitchToAllNamespacesMsg struct{}
