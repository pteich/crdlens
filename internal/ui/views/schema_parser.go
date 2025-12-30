package views

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// ParseSchemaFields recursively extracts fields from a JSON Schema
func ParseSchemaFields(schema *apiextensionsv1.JSONSchemaProps, basePath string, requiredFields map[string]bool) []SchemaField {
	var fields []SchemaField

	if schema == nil {
		return fields
	}

	if schema.Type == "array" && schema.Items != nil && schema.Items.Schema != nil {
		if len(schema.Items.Schema.Properties) > 0 {
			nestedRequired := make(map[string]bool)
			for _, req := range schema.Items.Schema.Required {
				nestedRequired[req] = true
			}
			nestedFields := ParseSchemaFields(schema.Items.Schema, basePath+"[]", nestedRequired)
			fields = append(fields, nestedFields...)
		} else if schema.Items.Schema.Type != "" {
			itemType := string(schema.Items.Schema.Type)
			fields = append(fields, SchemaField{
				FieldPath:   basePath,
				Type:        "array of " + itemType,
				Description: schema.Items.Schema.Description,
				Required:    requiredFields[basePath],
			})
		}
		return fields
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

		if fieldType == "object" && len(propSchema.Properties) > 0 {
			fields = append(fields, SchemaField{
				FieldPath:   fieldPath,
				Type:        fieldType,
				Description: description,
				Required:    required,
			})

			nestedRequired := make(map[string]bool)
			for _, req := range propSchema.Required {
				nestedRequired[req] = true
			}
			nestedFields := ParseSchemaFields(&propSchema, fieldPath, nestedRequired)
			fields = append(fields, nestedFields...)
		} else if fieldType == "array" && propSchema.Items != nil && propSchema.Items.Schema != nil {
			if len(propSchema.Items.Schema.Properties) > 0 {
				fields = append(fields, SchemaField{
					FieldPath:   fieldPath,
					Type:        "array of object",
					Description: description,
					Required:    required,
				})

				nestedRequired := make(map[string]bool)
				for _, req := range propSchema.Items.Schema.Required {
					nestedRequired[req] = true
				}
				nestedFields := ParseSchemaFields(propSchema.Items.Schema, fieldPath+"[]", nestedRequired)
				fields = append(fields, nestedFields...)
			} else if propSchema.Items.Schema.Type != "" {
				itemType := string(propSchema.Items.Schema.Type)
				fields = append(fields, SchemaField{
					FieldPath:   fieldPath,
					Type:        "array of " + itemType,
					Description: description,
					Required:    required,
				})
			}
		} else {
			fields = append(fields, SchemaField{
				FieldPath:   fieldPath,
				Type:        fieldType,
				Description: description,
				Required:    required,
			})
		}
	}

	return fields
}

// ExtractCRDSchemaFields extracts all fields from a CRD's schema
func ExtractCRDSchemaFields(crd *apiextensionsv1.CustomResourceDefinition) []SchemaField {
	if crd == nil || len(crd.Spec.Versions) == 0 {
		return nil
	}

	var fields []SchemaField

	for _, version := range crd.Spec.Versions {
		if !version.Served {
			continue
		}

		if version.Schema != nil && version.Schema.OpenAPIV3Schema != nil {
			requiredFields := make(map[string]bool)
			for _, req := range version.Schema.OpenAPIV3Schema.Required {
				requiredFields[req] = true
			}

			schemaFields := ParseSchemaFields(version.Schema.OpenAPIV3Schema, "", requiredFields)
			fields = append(fields, schemaFields...)
			break
		}
	}

	return fields
}

// SchemaField represents a parsed field for table display
type SchemaField struct {
	FieldPath   string
	Type        string
	Description string
	Required    bool
}

// TableRow returns a table row for this field
func (f SchemaField) TableRow() []string {
	required := "No"
	if f.Required {
		required = "Yes"
	}

	return []string{f.FieldPath, f.Type, required}
}

// TableHeaders returns the table headers for schema fields
func TableHeaders() []string {
	return []string{"Field Path", "Type", "Required"}
}
