package types

import (
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ReconcileState represents the current reconciliation state of a resource
type ReconcileState int

const (
	ReconcileStateIdle     ReconcileState = iota // Controller has processed latest spec
	ReconcileStateInFlight                       // Controller is processing spec changes
	ReconcileStateStuck                          // Inflight for too long without progress
	ReconcileStateError                          // Controller reported an error
)

// String returns a human-readable representation of the reconcile state
func (s ReconcileState) String() string {
	switch s {
	case ReconcileStateIdle:
		return "Idle"
	case ReconcileStateInFlight:
		return "InFlight"
	case ReconcileStateStuck:
		return "Stuck"
	case ReconcileStateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// Condition represents a Kubernetes-style condition from status.conditions
type Condition struct {
	Type               string
	Status             string // "True", "False", "Unknown"
	Reason             string
	Message            string
	LastTransitionTime time.Time
}

// IsReady returns true if this is a "Ready" condition with status "True"
func (c Condition) IsReady() bool {
	return (c.Type == "Ready" || c.Type == "Available" || c.Type == "Healthy") && c.Status == "True"
}

// Resource represents a single Custom Resource instance
type Resource struct {
	// Basic metadata
	Name      string
	Namespace string
	UID       string
	Kind      string
	GVR       schema.GroupVersionResource
	Age       time.Duration
	CreatedAt time.Time
	Raw       *unstructured.Unstructured

	// Controller-Aware Fields
	Generation         int64       // metadata.generation
	ObservedGeneration int64       // status.observedGeneration (0 if not present)
	Conditions         []Condition // status.conditions[]
	ControllerManager  string      // Primary controller from managedFields
	LastStatusWrite    time.Time   // Last time status was written (from managedFields)
	SpecWriteTime      time.Time   // Last time spec was written (from managedFields)
}

// Lag returns the reconciliation lag
func (r Resource) Lag() time.Duration {
	if r.SpecWriteTime.IsZero() {
		return 0
	}

	if r.IsReconciling() {
		return time.Since(r.SpecWriteTime)
	}

	// If finished, lag is time between spec change and status update
	if !r.LastStatusWrite.IsZero() && r.LastStatusWrite.After(r.SpecWriteTime) {
		return r.LastStatusWrite.Sub(r.SpecWriteTime)
	}

	return 0
}

// Silence returns the time since the last status update
func (r Resource) Silence() time.Duration {
	if r.LastStatusWrite.IsZero() {
		return time.Since(r.CreatedAt)
	}
	return time.Since(r.LastStatusWrite)
}

// Drift returns the difference between generation and observed generation
// A positive value indicates the controller hasn't processed the latest spec
func (r Resource) Drift() int64 {
	return r.Generation - r.ObservedGeneration
}

// IsReconciling returns true if the controller is still processing spec changes
// Only returns true if observedGeneration is present AND there's drift
func (r Resource) IsReconciling() bool {
	// Only consider reconciling if observedGeneration is present (non-zero)
	// and there's actual drift
	return r.ObservedGeneration > 0 && r.Drift() > 0
}

// HasObservedGeneration returns true if the resource has an observedGeneration field
func (r Resource) HasObservedGeneration() bool {
	return r.ObservedGeneration > 0
}

// ReadyStatus returns a summary of the ready state based on conditions
// Returns: "Ready", "NotReady", "Progressing", or "Unknown"
func (r Resource) ReadyStatus() string {
	var ready, notReady, progressing bool

	for _, c := range r.Conditions {
		switch c.Type {
		case "Ready", "Available", "Healthy", "Synced":
			if c.Status == "True" {
				ready = true
			} else if c.Status == "False" {
				notReady = true
			}
		case "Reconciling", "Progressing":
			if c.Status == "True" {
				progressing = true
			}
		}
	}

	if progressing {
		return "Progressing"
	}
	if ready && !notReady {
		return "Ready"
	}
	if notReady {
		return "NotReady"
	}
	return "Unknown"
}

// ReadyIcon returns an icon representing the ready status
func (r Resource) ReadyIcon() string {
	// Only show reconciling spinner if we actually have observedGeneration
	if r.HasObservedGeneration() && r.IsReconciling() {
		return "⏳" // Reconciling/syncing
	}

	status := r.ReadyStatus()
	switch status {
	case "Ready":
		return "✅"
	case "NotReady":
		return "❌"
	case "Progressing":
		return "⏳"
	default:
		return "❔"
	}
}
