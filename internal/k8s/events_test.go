package k8s

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEventService_GetEventsForResource(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	eventsClient := clientset.CoreV1().Events("default")

	// Pre-populate fake client with events
	now := time.Now()
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-event",
			Namespace: "default",
		},
		InvolvedObject: corev1.ObjectReference{
			UID: "test-uid",
		},
		Type:          "Normal",
		Reason:        "Created",
		Message:       "Created resource",
		LastTimestamp: metav1.Time{Time: now},
		Count:         1,
	}

	_, err := eventsClient.Create(context.Background(), event, metav1.CreateOptions{})
	require.NoError(t, err)

	svc := NewEventService(eventsClient)
	events, err := svc.GetEventsForResource(context.Background(), "default", "test-uid")

	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "Normal", events[0].Type)
	assert.Equal(t, "Created", events[0].Reason)
	assert.Equal(t, "Created resource", events[0].Message)
}
