package views

import (
	"fmt"
	"sort"
	"strconv"
)

// ValueField represents a field in the resource data
type ValueField struct {
	Name     string
	Key      string
	Value    string
	Type     string
	Children []ValueField
}

// TableRow returns a table row for this field
func (f ValueField) TableRow() []string {
	name := f.Name
	if len(f.Children) > 0 {
		name = name + " ▸"
	}
	return []string{name, f.Value, f.Type}
}

// ParseValueFields recursively parses a map into a slice of ValueField
func ParseValueFields(data map[string]interface{}, activePrefix string) []ValueField {
	var fields []ValueField

	for key, val := range data {
		fullKey := key
		if activePrefix != "" {
			fullKey = activePrefix + "." + key
		}

		field := ValueField{
			Name: key,
			Key:  fullKey,
		}

		switch v := val.(type) {
		case map[string]interface{}:
			field.Type = "map"
			field.Children = ParseValueFields(v, fullKey)
		case []interface{}:
			field.Type = "list"
			// Handle list of objects or primitives
			// For simplicity in the list view, we might want to show children as index items
			field.Children = parseListFields(v, fullKey)
			field.Value = fmt.Sprintf("[%d items]", len(v))
		case string:
			field.Type = "string"
			field.Value = v
		case int, int32, int64, float64:
			field.Type = "number"
			field.Value = fmt.Sprintf("%v", v)
		case bool:
			field.Type = "bool"
			field.Value = strconv.FormatBool(v)
		case nil:
			field.Type = "null"
			field.Value = "null"
		default:
			field.Type = "unknown"
			field.Value = fmt.Sprintf("%v", v)
		}

		fields = append(fields, field)
	}

	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})

	return fields
}

func parseListFields(list []interface{}, parentKey string) []ValueField {
	var fields []ValueField
	for i, item := range list {
		key := fmt.Sprintf("[%d]", i)
		fullKey := parentKey + key

		field := ValueField{
			Name: key,
			Key:  fullKey,
		}

		switch v := item.(type) {
		case map[string]interface{}:
			field.Type = "object"
			field.Children = ParseValueFields(v, fullKey)
		case []interface{}:
			field.Type = "list"
			field.Children = parseListFields(v, fullKey)
			field.Value = fmt.Sprintf("[%d items]", len(v))
		default:
			field.Type = "value"
			field.Value = fmt.Sprintf("%v", v)
		}

		fields = append(fields, field)
	}
	return fields
}
