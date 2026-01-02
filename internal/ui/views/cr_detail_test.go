package views

import (
	"testing"

	"github.com/pteich/crdlens/internal/types"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestCRDetailModel_Update_FormattedYAML(t *testing.T) {
	m := NewCRDetailModel(nil, types.Resource{}, 100, 100)

	msg := FormattedYAMLMsg{YAML: "key: value"}
	_, cmd := m.Update(msg)

	assert.Nil(t, cmd)
	assert.Contains(t, m.viewport.View(), "key: value") // Viewport returns content in view
}

func TestCRDetailModel_Update_FetchedEvents(t *testing.T) {
	m := NewCRDetailModel(nil, types.Resource{}, 100, 100)

	events := []types.Event{
		{
			Type:    "Normal",
			Reason:  "Created",
			Message: "Created resource",
		},
	}

	msg := FetchedEventsMsg{Events: events}
	_, cmd := m.Update(msg)

	assert.Nil(t, cmd)
	assert.Equal(t, 1, len(m.eventTable.Rows()))
	assert.Equal(t, "Normal", m.eventTable.Rows()[0][0])
}

func TestCRDetailModel_Update_ParsedFields(t *testing.T) {
	m := NewCRDetailModel(nil, types.Resource{}, 100, 100)

	fields := []ValueField{
		{Name: "foo", Value: "bar"},
	}

	msg := ParsedFieldsMsg{Fields: fields}
	_, cmd := m.Update(msg)

	assert.Nil(t, cmd)
	assert.Equal(t, 1, len(m.fieldTable.Rows()))
	assert.Equal(t, "foo", m.fieldTable.Rows()[0][0])
}

func TestCRDetailModel_FormatYAML(t *testing.T) {
	res := types.Resource{
		Raw: &unstructured.Unstructured{
			Object: map[string]interface{}{
				"kind": "Test",
			},
		},
	}
	m := NewCRDetailModel(nil, res, 100, 100)

	msg := m.FormatYAML()
	formattedMsg, ok := msg.(FormattedYAMLMsg)

	assert.True(t, ok)
	assert.Contains(t, formattedMsg.YAML, "kind: Test")
}
