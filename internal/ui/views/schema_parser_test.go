package views

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseSchemaFields_SimpleObject(t *testing.T) {
	schema := &apiextensionsv1.JSONSchemaProps{
		Type: "object",
		Properties: map[string]apiextensionsv1.JSONSchemaProps{
			"replicas": {
				Type:        "integer",
				Description: "Number of replicas",
			},
			"enabled": {
				Type:        "boolean",
				Description: "Whether the feature is enabled",
			},
		},
		Required: []string{"replicas"},
	}

	required := map[string]bool{"replicas": true}
	fields := ParseSchemaFields(schema, "", required)

	require.Len(t, fields, 2)

	replicasField := findField(t, fields, "replicas")
	assert.Equal(t, "integer", replicasField.Type)
	assert.Equal(t, "Number of replicas", replicasField.Description)
	assert.True(t, replicasField.Required)

	enabledField := findField(t, fields, "enabled")
	assert.Equal(t, "boolean", enabledField.Type)
	assert.Equal(t, "Whether the feature is enabled", enabledField.Description)
	assert.False(t, enabledField.Required)
}

func TestParseSchemaFields_NestedObject(t *testing.T) {
	schema := &apiextensionsv1.JSONSchemaProps{
		Type: "object",
		Properties: map[string]apiextensionsv1.JSONSchemaProps{
			"spec": {
				Type: "object",
				Properties: map[string]apiextensionsv1.JSONSchemaProps{
					"replicas": {
						Type:        "integer",
						Description: "Number of replicas",
					},
				},
				Required: []string{"replicas"},
			},
		},
	}

	required := map[string]bool{}
	fields := ParseSchemaFields(schema, "", required)

	require.Len(t, fields, 2)

	specField := findField(t, fields, "spec")
	assert.Equal(t, "object", specField.Type)

	replicasField := findField(t, fields, "spec.replicas")
	assert.Equal(t, "integer", replicasField.Type)
	assert.Equal(t, "Number of replicas", replicasField.Description)
	assert.True(t, replicasField.Required)
}

func TestParseSchemaFields_Array(t *testing.T) {
	schema := &apiextensionsv1.JSONSchemaProps{
		Type: "object",
		Properties: map[string]apiextensionsv1.JSONSchemaProps{
			"items": {
				Type: "array",
				Items: &apiextensionsv1.JSONSchemaPropsOrArray{
					Schema: &apiextensionsv1.JSONSchemaProps{
						Type:        "string",
						Description: "Item in the list",
					},
				},
				Description: "List of items",
			},
		},
	}

	fields := ParseSchemaFields(schema, "", map[string]bool{})

	require.Len(t, fields, 1)
	assert.Equal(t, "items", fields[0].FieldPath)
	assert.Equal(t, "array of string", fields[0].Type)
	assert.Equal(t, "List of items", fields[0].Description)
}

func TestExtractCRDSchemaFields_NoVersions(t *testing.T) {
	crd := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test.example.com",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{},
		},
	}

	fields := ExtractCRDSchemaFields(crd)
	assert.Nil(t, fields)
}

func TestExtractCRDSchemaFields_WithVersions(t *testing.T) {
	crd := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test.example.com",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1",
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]apiextensionsv1.JSONSchemaProps{
								"spec": {
									Type:        "object",
									Description: "Specification",
								},
							},
						},
					},
				},
			},
		},
	}

	fields := ExtractCRDSchemaFields(crd)
	require.Len(t, fields, 1)
	assert.Equal(t, "spec", fields[0].FieldPath)
	assert.Equal(t, "Specification", fields[0].Description)
}

func TestExtractCRDSchemaFields_NonServedVersion(t *testing.T) {
	crd := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test.example.com",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1alpha1",
					Served:  false,
					Storage: false,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]apiextensionsv1.JSONSchemaProps{
								"spec": {
									Type:        "object",
									Description: "Spec",
								},
							},
						},
					},
				},
			},
		},
	}

	fields := ExtractCRDSchemaFields(crd)
	assert.Nil(t, fields)
}

func findField(t *testing.T, fields []SchemaField, path string) SchemaField {
	t.Helper()
	for _, field := range fields {
		if field.FieldPath == path {
			return field
		}
	}
	t.Fatalf("field with path %s not found", path)
	return SchemaField{}
}
