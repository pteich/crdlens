package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
)

type MockDiscovery struct {
	discovery.DiscoveryInterface
	Resources []*metav1.APIResourceList
}

func (m *MockDiscovery) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	return m.Resources, nil
}

func TestDiscoveryService_ListCRDs(t *testing.T) {
	mock := &MockDiscovery{
		Resources: []*metav1.APIResourceList{
			{
				GroupVersion: "cert-manager.io/v1",
				APIResources: []metav1.APIResource{
					{
						Name:       "certificates",
						Namespaced: true,
						Kind:       "Certificate",
					},
				},
			},
			{
				GroupVersion: "v1", // Core resources should be skipped
				APIResources: []metav1.APIResource{
					{
						Name:       "pods",
						Namespaced: true,
						Kind:       "Pod",
					},
				},
			},
		},
	}

	svc := NewDiscoveryService(mock)
	crds, err := svc.ListCRDs(context.Background())

	require.NoError(t, err)
	require.Len(t, crds, 1, "should find exactly one CRD")

	crd := crds[0]
	assert.Equal(t, "Certificate", crd.Kind)
	assert.Equal(t, "cert-manager.io", crd.Group)
	assert.Equal(t, "v1", crd.Version)
	assert.Equal(t, "Namespaced", crd.Scope)
}
