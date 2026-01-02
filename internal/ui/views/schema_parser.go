package views

import (
	"sort"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// SchemaField represents a parsed field for table display
type SchemaField struct {
	Name        string
	FieldPath   string
	Type        string
	Description string
	Required    bool
	Children    []SchemaField
}

// ParseSchemaFields recursively extracts fields from a JSON Schema
func ParseSchemaFields(schema *apiextensionsv1.JSONSchemaProps, basePath string, requiredFields map[string]bool) []SchemaField {
	var fields []SchemaField

	if schema == nil {
		return fields
	}

	// Handle array types that have a schema for items
	if schema.Type == "array" && schema.Items != nil && schema.Items.Schema != nil {
		// If items are objects with properties, we treat it as a nested structure
		if len(schema.Items.Schema.Properties) > 0 {

			// Recurse for children
			nestedRequired := make(map[string]bool)
			for _, req := range schema.Items.Schema.Required {
				nestedRequired[req] = true
			}
			// When parsing children of an array, the path for children should logically be valid.
			// e.g. spec.containers -> spec.containers[].name
			// For the visual hierarchy, we just pass the children.
			children := ParseSchemaFields(schema.Items.Schema, basePath+"[]", nestedRequired)

			return children
		}
		// arrays of primitives are handled as a single field in the parent usually.
	}

	for propName, propSchema := range schema.Properties {
		fieldPath := propName
		if basePath != "" {
			fieldPath = basePath + "." + propName
		}

		fieldType := "unknown"
		if propSchema.Type != "" {
			fieldType = string(propSchema.Type)
		} else if len(propSchema.AnyOf) > 0 {
			fieldType = "anyOf"
		} else if len(propSchema.AllOf) > 0 {
			fieldType = "allOf"
		} else if len(propSchema.OneOf) > 0 {
			fieldType = "oneOf"
		}

		required := requiredFields[propName]
		description := propSchema.Description

		field := SchemaField{
			Name:        propName,
			FieldPath:   fieldPath,
			Type:        fieldType,
			Description: description,
			Required:    required,
		}

		// Check for nested children
		if fieldType == "object" && len(propSchema.Properties) > 0 {
			nestedRequired := make(map[string]bool)
			for _, req := range propSchema.Required {
				nestedRequired[req] = true
			}
			field.Children = ParseSchemaFields(&propSchema, fieldPath, nestedRequired)
		} else if fieldType == "array" && propSchema.Items != nil && propSchema.Items.Schema != nil {
			if len(propSchema.Items.Schema.Properties) > 0 {
				field.Type = "array of object"
				nestedRequired := make(map[string]bool)
				for _, req := range propSchema.Items.Schema.Required {
					nestedRequired[req] = true
				}
				field.Children = ParseSchemaFields(propSchema.Items.Schema, fieldPath+"[]", nestedRequired)
			} else if propSchema.Items.Schema.Type != "" {
				itemType := string(propSchema.Items.Schema.Type)
				field.Type = "array of " + itemType
			}
		}

		fields = append(fields, field)
	}

	// Sort fields by name for consistent display
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})

	return fields
}

// FlattenSchemaFields converts a hierarchical list of fields into a flat list
func FlattenSchemaFields(fields []SchemaField) []SchemaField {
	var flat []SchemaField
	for _, field := range fields {
		// Create a copy without children for the flat list
		item := field
		flat = append(flat, item)

		if len(field.Children) > 0 {
			nested := FlattenSchemaFields(field.Children)
			flat = append(flat, nested...)
		}
	}
	return flat
}

// ExtractCRDSchemaFields extracts all fields from a CRD's schema
func ExtractCRDSchemaFields(crd *apiextensionsv1.CustomResourceDefinition) []SchemaField {
	if crd == nil || len(crd.Spec.Versions) == 0 {
		return nil
	}

	for _, version := range crd.Spec.Versions {
		if !version.Served {
			continue
		}

		if version.Schema != nil && version.Schema.OpenAPIV3Schema != nil {
			requiredFields := make(map[string]bool)
			for _, req := range version.Schema.OpenAPIV3Schema.Required {
				requiredFields[req] = true
			}

			// We treat the top level schema as the source of fields.
			// The top level is usually an object.
			return ParseSchemaFields(version.Schema.OpenAPIV3Schema, "", requiredFields)
		}
	}

	return nil
}

// TableRow returns a table row for this field
func (f SchemaField) TableRow(isFlat bool) []string {
	required := "No"
	if f.Required {
		required = "Yes"
	}

	name := f.Name
	if !isFlat {
		// Add indicator if it has children
		if len(f.Children) > 0 {
			name = name + " ▸"
		}
	} else {
		name = f.FieldPath
	}

	return []string{name, f.Type, required}
}

// TableHeaders returns the table headers for schema fields
func TableHeaders() []string {
	return []string{"Field", "Type", "Required"}
}
