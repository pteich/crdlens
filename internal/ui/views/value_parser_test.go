package views

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseValueFields_Simple(t *testing.T) {
	data := map[string]interface{}{
		"name":   "test-resource",
		"age":    10,
		"active": true,
	}

	fields := ParseValueFields(data, "")
	require.Len(t, fields, 3)

	assert.Equal(t, "active", fields[0].Name)
	assert.Equal(t, "true", fields[0].Value)

	assert.Equal(t, "age", fields[1].Name)
	assert.Equal(t, "10", fields[1].Value)

	assert.Equal(t, "name", fields[2].Name)
	assert.Equal(t, "test-resource", fields[2].Value)
}

func TestParseValueFields_Nested(t *testing.T) {
	data := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "my-pod",
			"labels": map[string]interface{}{
				"app": "nginx",
			},
		},
	}

	fields := ParseValueFields(data, "")
	require.Len(t, fields, 1)

	metadata := fields[0]
	assert.Equal(t, "metadata", metadata.Name)
	assert.Equal(t, "map", metadata.Type)
	require.Len(t, metadata.Children, 2)

	labels := metadata.Children[0]
	assert.Equal(t, "labels", labels.Name)
	assert.Equal(t, "metadata.labels", labels.Key)

	name := metadata.Children[1]
	assert.Equal(t, "name", name.Name)
	assert.Equal(t, "my-pod", name.Value)
}

func TestParseValueFields_List(t *testing.T) {
	data := map[string]interface{}{
		"items": []interface{}{
			"a",
			map[string]interface{}{"b": 2},
		},
	}

	fields := ParseValueFields(data, "")
	require.Len(t, fields, 1)

	items := fields[0]
	assert.Equal(t, "items", items.Name)
	assert.Equal(t, "list", items.Type)
	require.Len(t, items.Children, 2)

	item0 := items.Children[0]
	assert.Equal(t, "[0]", item0.Name)
	assert.Equal(t, "a", item0.Value)

	item1 := items.Children[1]
	assert.Equal(t, "[1]", item1.Name)
	assert.Equal(t, "object", item1.Type)
	require.Len(t, item1.Children, 1)
	assert.Equal(t, "b", item1.Children[0].Name)
	assert.Equal(t, "2", item1.Children[0].Value)
}
