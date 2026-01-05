package views

import (
	"testing"
	"time"

	"github.com/pteich/crdlens/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestCRListModel_Update_FetchedCRs(t *testing.T) {
	m := NewCRListModel(nil, types.CRDInfo{Kind: "TestKind"}, "default", 100, 100)

	now := time.Now()
	resources := []types.Resource{
		{
			Name:      "test-1",
			Namespace: "default",
			Age:       10 * time.Minute,
			CreatedAt: now,
		},
	}

	msg := FetchedCRsMsg{Resources: resources}
	_, cmd := m.Update(msg)

	assert.Nil(t, cmd)
	assert.False(t, m.loading)
	assert.Equal(t, 1, len(m.table.Rows()))
	// New column structure: [0]=R, [1]=Status, [2]=Name, [3]=NS, [4]=Drift, [5]=Ctrl, [6]=Created
	assert.Equal(t, "❔", m.table.Rows()[0][0])       // Ready icon (unknown - no conditions)
	assert.Equal(t, "Unknown", m.table.Rows()[0][1]) // Status text (new column)
	assert.Equal(t, "test-1", m.table.Rows()[0][2])  // Name
	assert.Equal(t, "default", m.table.Rows()[0][3]) // Namespace
}
