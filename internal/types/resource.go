package types

import (
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Resource represents a single Custom Resource instance
type Resource struct {
	Name      string
	Namespace string
	UID       string
	Kind      string
	GVR       schema.GroupVersionResource
	Age       time.Duration
	Raw       *unstructured.Unstructured
}
