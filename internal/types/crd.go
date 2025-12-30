package types

import "k8s.io/apimachinery/pkg/runtime/schema"

// CRDInfo contains metadata about a discovered CRD
type CRDInfo struct {
	Name    string
	Group   string
	Version string
	Kind    string
	Scope   string // Namespaced or Cluster
	GVR     schema.GroupVersionResource
	Count   int // Number of instances (optional/cached)
}
