package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/pteich/crdlens/internal/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// DynamicService handles CR instance operations
type DynamicService struct {
	client dynamic.Interface
}

// NewDynamicService creates a new DynamicService
func NewDynamicService(client dynamic.Interface) *DynamicService {
	return &DynamicService{
		client: client,
	}
}

// ListResources lists instance of a CRD
func (s *DynamicService) ListResources(ctx context.Context, gvr schema.GroupVersionResource, namespace string) ([]types.Resource, error) {
	res, err := s.client.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list %s: %w", gvr.Resource, err)
	}

	var resources []types.Resource
	for _, item := range res.Items {
		creationTimestamp := item.GetCreationTimestamp()
		age := time.Since(creationTimestamp.Time)

		resources = append(resources, types.Resource{
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
			UID:       string(item.GetUID()),
			Kind:      item.GetKind(),
			GVR:       gvr,
			Age:       age,
			Raw:       &item,
		})
	}

	return resources, nil
}

// GetResource gets a specific CR instance
func (s *DynamicService) GetResource(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) (*types.Resource, error) {
	item, err := s.client.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	creationTimestamp := item.GetCreationTimestamp()
	age := time.Since(creationTimestamp.Time)

	return &types.Resource{
		Name:      item.GetName(),
		Namespace: item.GetNamespace(),
		UID:       string(item.GetUID()),
		Kind:      item.GetKind(),
		GVR:       gvr,
		Age:       age,
		Raw:       item,
	}, nil
}
