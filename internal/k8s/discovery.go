package k8s

import (
	"context"
	"fmt"

	"github.com/pteich/crdlens/internal/types"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DiscoveryService handles finding CRDs and their GVR info
type DiscoveryService struct {
	client clientset.Interface
}

// NewDiscoveryService creates a new DiscoveryService
func NewDiscoveryService(client clientset.Interface) *DiscoveryService {
	return &DiscoveryService{
		client: client,
	}
}

// ListCRDs finds all CRDs in the cluster
func (s *DiscoveryService) ListCRDs(ctx context.Context) ([]types.CRDInfo, error) {
	crdList, err := s.client.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list crds: %w", err)
	}

	var crds []types.CRDInfo
	for _, crd := range crdList.Items {
		// Find the served version that is marked as storage
		var version string
		for _, v := range crd.Spec.Versions {
			if v.Served {
				version = v.Name
				if v.Storage {
					break
				}
			}
		}

		if version == "" {
			continue
		}

		scope := "Namespaced"
		if crd.Spec.Scope == "Cluster" {
			scope = "Cluster"
		}

		crds = append(crds, types.CRDInfo{
			Name:    crd.Name,
			Group:   crd.Spec.Group,
			Version: version,
			Kind:    crd.Spec.Names.Kind,
			Scope:   scope,
			GVR: schema.GroupVersionResource{
				Group:    crd.Spec.Group,
				Version:  version,
				Resource: crd.Spec.Names.Plural,
			},
		})
	}

	return crds, nil
}
