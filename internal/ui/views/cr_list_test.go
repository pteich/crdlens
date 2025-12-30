package views

import (
	"testing"
	"time"

	"github.com/pteich/crdlens/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestCRListModel_Update_FetchedCRs(t *testing.T) {
	m := NewCRListModel(nil, types.CRDInfo{Kind: "TestKind"}, "default", 100, 100)

	resources := []types.Resource{
		{
			Name:      "test-1",
			Namespace: "default",
			Age:       10 * time.Minute,
		},
	}

	msg := FetchedCRsMsg{Resources: resources}
	_, cmd := m.Update(msg)

	assert.Nil(t, cmd)
	assert.False(t, m.loading)
	assert.Equal(t, 1, len(m.table.Rows()))
	assert.Equal(t, "test-1", m.table.Rows()[0][0])
	assert.Equal(t, "default", m.table.Rows()[0][1])
}
