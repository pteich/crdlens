package types

import "time"

// Event represents a Kubernetes event
type Event struct {
	Type          string
	Reason        string
	Message       string
	LastTimestamp time.Time
	Count         int32
}
