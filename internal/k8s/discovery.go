package k8s

import (
	"context"
	"fmt"
	"strings"

	"github.com/pteich/crdlens/internal/types"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

// DiscoveryService handles finding CRDs and their GVR info
type DiscoveryService struct {
	client discovery.DiscoveryInterface
}

// NewDiscoveryService creates a new DiscoveryService
func NewDiscoveryService(client discovery.DiscoveryInterface) *DiscoveryService {
	return &DiscoveryService{
		client: client,
	}
}

// ListCRDs finds all CRDs in the cluster
func (s *DiscoveryService) ListCRDs(ctx context.Context) ([]types.CRDInfo, error) {
	// We use ServerPreferredResources to find the latest version of all resources
	resourceLists, err := s.client.ServerPreferredResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return nil, fmt.Errorf("failed to discover resources: %w", err)
	}

	var crds []types.CRDInfo
	for _, rl := range resourceLists {
		for _, r := range rl.APIResources {
			// Skip if it's not a CRD (standard resources don't have a '.' in the group or aren't custom)
			// Actually, a better way is to check the GroupVersion
			gv, err := schema.ParseGroupVersion(rl.GroupVersion)
			if err != nil {
				continue
			}

			// We are looking for resources that belong to a group (not core 'v1')
			// and where the Kind matches what we'd expect for a CRD
			// (this is a heuristic, but usually CRDs have a group like 'cert-manager.io')
			if gv.Group == "" || !strings.Contains(gv.Group, ".") {
				continue
			}

			scope := "Namespaced"
			if !r.Namespaced {
				scope = "Cluster"
			}

			crds = append(crds, types.CRDInfo{
				Name:    r.Name + "." + gv.Group,
				Group:   gv.Group,
				Version: gv.Version,
				Kind:    r.Kind,
				Scope:   scope,
				GVR: schema.GroupVersionResource{
					Group:    gv.Group,
					Version:  gv.Version,
					Resource: r.Name,
				},
			})
		}
	}

	return crds, nil
}
