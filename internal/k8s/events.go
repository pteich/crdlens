package k8s

import (
	"context"
	"fmt"

	"github.com/pteich/crdlens/internal/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// EventService handles Kubernetes event operations
type EventService struct {
	client v1.EventInterface
}

// NewEventService creates a new EventService
func NewEventService(client v1.EventInterface) *EventService {
	return &EventService{
		client: client,
	}
}

// GetEventsForResource fetches events for a specific resource
func (s *EventService) GetEventsForResource(ctx context.Context, namespace, uid string) ([]types.Event, error) {
	// Filter events by involvedObject.uid
	list, err := s.client.List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.uid=%s", uid),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	var events []types.Event
	for _, item := range list.Items {
		events = append(events, types.Event{
			Type:          item.Type,
			Reason:        item.Reason,
			Message:       item.Message,
			LastTimestamp: item.LastTimestamp.Time,
			Count:         item.Count,
		})
	}

	return events, nil
}
