package views

import (
	"context"
	"fmt"

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
	viewport viewport.Model
	client   *k8s.Client
	crd      types.CRDInfo
	loading  bool
	err      error
	spec     *apiextensionsv1.CustomResourceDefinition
	width    int
	height   int
}

// NewCRDSpecModel creates a new CRD spec model
func NewCRDSpecModel(client *k8s.Client, crd types.CRDInfo, width, height int) *CRDSpecModel {
	vp := viewport.New(width, height)

	return &CRDSpecModel{
		viewport: vp,
		client:   client,
		crd:      crd,
		width:    width,
		height:   height,
		loading:  true,
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
		return m, nil

	case ErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height

		if m.spec != nil {
			yamlBytes, err := yaml.Marshal(m.spec)
			if err == nil {
				m.viewport.SetContent(string(yamlBytes))
			}
		}
		return m, nil
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

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Render(fmt.Sprintf("CRD Spec: %s", m.crd.Name))

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		"\n",
		m.viewport.View(),
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
