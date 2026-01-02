package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDiscoveryService_ListCRDs(t *testing.T) {
	crd := &v1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "certificates.cert-manager.io",
		},
		Spec: v1.CustomResourceDefinitionSpec{
			Group: "cert-manager.io",
			Names: v1.CustomResourceDefinitionNames{
				Kind:   "Certificate",
				Plural: "certificates",
			},
			Scope: v1.NamespaceScoped,
			Versions: []v1.CustomResourceDefinitionVersion{
				{
					Name:    "v1",
					Served:  true,
					Storage: true,
				},
			},
		},
	}

	client := fake.NewSimpleClientset(crd)
	svc := NewDiscoveryService(client)
	crds, err := svc.ListCRDs(context.Background())

	require.NoError(t, err)
	require.Len(t, crds, 1, "should find exactly one CRD")

	result := crds[0]
	assert.Equal(t, "Certificate", result.Kind)
	assert.Equal(t, "cert-manager.io", result.Group)
	assert.Equal(t, "v1", result.Version)
	assert.Equal(t, "Namespaced", result.Scope)
	assert.Equal(t, "certificates", result.GVR.Resource)
}
