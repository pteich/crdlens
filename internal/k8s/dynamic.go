package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/pteich/crdlens/internal/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	// DefaultPageSize is the default number of resources to fetch per page
	DefaultPageSize = 100
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

// ListResourcesOptions configures pagination for listing resources
type ListResourcesOptions struct {
	Limit    int64  // Number of resources per page (0 = use default)
	Continue string // Continuation token for pagination
}

// ListResourcesResult contains the result of a paginated list operation
type ListResourcesResult struct {
	Resources      []types.Resource
	ContinueToken  string // Token for next page (empty if no more pages)
	RemainingCount *int64 // Approximate remaining items (may be nil)
	TotalCount     int    // Total fetched so far (including this page)
}

// ListResources lists instances of a CRD with optional pagination
func (s *DynamicService) ListResources(ctx context.Context, gvr schema.GroupVersionResource, namespace string) ([]types.Resource, error) {
	result, err := s.ListResourcesPaginated(ctx, gvr, namespace, ListResourcesOptions{})
	if err != nil {
		return nil, err
	}
	return result.Resources, nil
}

// ListResourcesPaginated lists instances of a CRD with pagination support
func (s *DynamicService) ListResourcesPaginated(ctx context.Context, gvr schema.GroupVersionResource, namespace string, opts ListResourcesOptions) (*ListResourcesResult, error) {
	limit := opts.Limit
	if limit == 0 {
		limit = DefaultPageSize
	}

	listOpts := metav1.ListOptions{
		Limit:    limit,
		Continue: opts.Continue,
	}

	res, err := s.client.Resource(gvr).Namespace(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list %s: %w", gvr.Resource, err)
	}

	resources := make([]types.Resource, 0, len(res.Items))
	for _, item := range res.Items {
		resources = append(resources, s.itemToResource(item, gvr))
	}

	return &ListResourcesResult{
		Resources:      resources,
		ContinueToken:  res.GetContinue(),
		RemainingCount: res.GetRemainingItemCount(),
		TotalCount:     len(resources),
	}, nil
}

// ListAllResources fetches all resources across all pages
// Use with caution for large result sets
func (s *DynamicService) ListAllResources(ctx context.Context, gvr schema.GroupVersionResource, namespace string, pageSize int64) ([]types.Resource, error) {
	if pageSize == 0 {
		pageSize = DefaultPageSize
	}

	var allResources []types.Resource
	continueToken := ""

	for {
		result, err := s.ListResourcesPaginated(ctx, gvr, namespace, ListResourcesOptions{
			Limit:    pageSize,
			Continue: continueToken,
		})
		if err != nil {
			return nil, err
		}

		allResources = append(allResources, result.Resources...)

		if result.ContinueToken == "" {
			break
		}
		continueToken = result.ContinueToken
	}

	return allResources, nil
}

// GetResource gets a specific CR instance
func (s *DynamicService) GetResource(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) (*types.Resource, error) {
	item, err := s.client.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	resource := s.itemToResource(*item, gvr)
	return &resource, nil
}

// CountResources counts the number of CR instances for a given GVR
func (s *DynamicService) CountResources(ctx context.Context, gvr schema.GroupVersionResource, namespace string) (int, error) {
	// Use limit=1 to minimize data transfer, just get the count
	res, err := s.client.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{
		Limit: 1,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count %s: %w", gvr.Resource, err)
	}

	// If there's a remaining count, add 1 (for the item we fetched)
	if remaining := res.GetRemainingItemCount(); remaining != nil {
		return int(*remaining) + len(res.Items), nil
	}

	// Fallback: no pagination info, need to count all
	allRes, err := s.client.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to count %s: %w", gvr.Resource, err)
	}
	return len(allRes.Items), nil
}

// itemToResource converts an unstructured item to a Resource with controller-aware fields
func (s *DynamicService) itemToResource(item unstructured.Unstructured, gvr schema.GroupVersionResource) types.Resource {
	creationTimestamp := item.GetCreationTimestamp()
	age := time.Since(creationTimestamp.Time)

	// Extract controller-aware information
	observedGen := ExtractObservedGeneration(&item)
	conditions := ExtractConditions(&item)
	controllerManager, lastStatusWrite := ExtractControllerInfo(item.GetManagedFields())

	return types.Resource{
		// Basic metadata
		Name:      item.GetName(),
		Namespace: item.GetNamespace(),
		UID:       string(item.GetUID()),
		Kind:      item.GetKind(),
		GVR:       gvr,
		Age:       age,
		CreatedAt: creationTimestamp.Time,
		Raw:       &item,

		// Controller-Aware Fields
		Generation:         item.GetGeneration(),
		ObservedGeneration: observedGen,
		Conditions:         conditions,
		ControllerManager:  controllerManager,
		LastStatusWrite:    lastStatusWrite,
	}
}
