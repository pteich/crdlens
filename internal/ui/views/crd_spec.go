package views

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pteich/crdlens/internal/k8s"
	"github.com/pteich/crdlens/internal/types"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// CRDSpecModel is the model for the CRD spec view
type CRDSpecModel struct {
	viewport        viewport.Model
	table           table.Model
	client          *k8s.Client
	crd             types.CRDInfo
	loading         bool
	err             error
	spec            *apiextensionsv1.CustomResourceDefinition
	fields          []SchemaField
	showTable       bool
	showFieldDetail bool
	selectedField   *SchemaField
	width           int
	height          int
}

// NewCRDSpecModel creates a new CRD spec model
func NewCRDSpecModel(client *k8s.Client, crd types.CRDInfo, width, height int) *CRDSpecModel {
	vp := viewport.New(width, height-8)

	columns := []table.Column{
		{Title: "Field Path", Width: 40},
		{Title: "Type", Width: 20},
		{Title: "Required", Width: 12},
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

	return &CRDSpecModel{
		viewport:  vp,
		table:     t,
		client:    client,
		crd:       crd,
		width:     width,
		height:    height,
		loading:   true,
		showTable: true,
	}
}

// Init initializes the model
func (m *CRDSpecModel) Init() tea.Cmd {
	return m.FetchCRDSpec
}

// Update handles messages
func (m *CRDSpecModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case FetchedCRDSpecMsg:
		m.loading = false
		m.spec = msg.Spec

		yamlBytes, err := yaml.Marshal(msg.Spec)
		if err != nil {
			m.err = err
			return m, nil
		}

		m.viewport.SetContent(string(yamlBytes))

		m.fields = ExtractCRDSchemaFields(msg.Spec)
		rows := make([]table.Row, len(m.fields))
		for i, field := range m.fields {
			rows[i] = field.TableRow()
		}
		m.table.SetRows(rows)

		return m, nil

	case ErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 8
		m.table.SetHeight(msg.Height - 10)

		if m.spec != nil {
			yamlBytes, err := yaml.Marshal(m.spec)
			if err == nil {
				m.viewport.SetContent(string(yamlBytes))
			}
		}
		return m, nil

	case tea.KeyMsg:
		if m.showFieldDetail && (msg.String() == "esc" || msg.String() == "enter") {
			m.showFieldDetail = false
			m.selectedField = nil
			return m, nil
		}

		if !m.showFieldDetail {
			if msg.String() == "tab" {
				m.showTable = !m.showTable
				if m.showTable && len(m.fields) > 0 {
					m.table.GotoTop()
				}
				return m, nil
			}

			if m.showTable && msg.String() == "enter" {
				idx := m.table.Cursor()
				if idx >= 0 && idx < len(m.fields) {
					m.selectedField = &m.fields[idx]
					m.showFieldDetail = true
				}
				return m, nil
			}
		}
	}

	if m.showFieldDetail {
		return m, nil
	}

	if m.showTable {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the model
func (m *CRDSpecModel) View() string {
	if m.loading {
		return "\nLoading CRD spec..."
	}
	if m.err != nil {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Render(fmt.Sprintf("Error fetching CRD spec: %v", m.err))
	}

	viewMode := "YAML"
	if m.showTable {
		viewMode = "Table"
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Render(fmt.Sprintf("CRD Spec: %s  [Tab: View Mode (%s)] [Enter: Field Details] [Esc: Close]", m.crd.Name, viewMode))

	var baseView string
	if m.showTable {
		baseView = lipgloss.JoinVertical(lipgloss.Left,
			title,
			"\n",
			m.table.View(),
		)
	} else {
		baseView = lipgloss.JoinVertical(lipgloss.Left,
			title,
			"\n",
			m.viewport.View(),
		)
	}

	if m.showFieldDetail && m.selectedField != nil {
		return m.renderFieldDetailOverlay(baseView)
	}

	return baseView
}

// IsShowingFieldDetail returns whether the field detail overlay is currently shown
func (m *CRDSpecModel) IsShowingFieldDetail() bool {
	return m.showFieldDetail
}

func (m *CRDSpecModel) renderFieldDetailOverlay(baseView string) string {
	overlayWidth := 60
	// Ensure overlay doesn't exceed screen width
	if m.width < overlayWidth+4 {
		overlayWidth = m.width - 4
	}

	required := "No"
	if m.selectedField.Required {
		required = "Yes"
	}

	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")). // Primary color
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F780E2")). // Secondary color
		Width(12)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF"))

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#BBBBBB")).
		MarginTop(1)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		MarginTop(1).
		Align(lipgloss.Right)

	// Content Construction
	content := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Field Details"),
		lipgloss.JoinHorizontal(lipgloss.Left, labelStyle.Render("Path:"), valueStyle.Render(m.selectedField.FieldPath)),
		lipgloss.JoinHorizontal(lipgloss.Left, labelStyle.Render("Type:"), valueStyle.Render(m.selectedField.Type)),
		lipgloss.JoinHorizontal(lipgloss.Left, labelStyle.Render("Required:"), valueStyle.Render(required)),
		descStyle.Render(m.selectedField.Description),
		helpStyle.Render("esc or enter to close"),
	)

	// Overlay Box
	overlay := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(1, 2).
		Width(overlayWidth).
		Render(content)

	// Center the overlay
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#1a1a1a")), // Darken background slightly if possible, or just standard
	)
}

// Messages
type FetchedCRDSpecMsg struct {
	Spec *apiextensionsv1.CustomResourceDefinition
}

// FetchCRDSpec is a command to fetch the CRD spec from the cluster
func (m *CRDSpecModel) FetchCRDSpec() tea.Msg {
	config, err := apiextensionsclientset.NewForConfig(m.client.Config)
	if err != nil {
		return ErrorMsg{Err: err}
	}

	spec, err := config.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), m.crd.Name, metav1.GetOptions{})
	if err != nil {
		return ErrorMsg{Err: err}
	}

	return FetchedCRDSpecMsg{Spec: spec}
}
