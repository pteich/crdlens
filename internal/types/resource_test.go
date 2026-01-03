package types

import (
	"testing"
)

func TestResource_Drift(t *testing.T) {
	tests := []struct {
		name     string
		res      Resource
		expected int64
	}{
		{
			name:     "no drift",
			res:      Resource{Generation: 5, ObservedGeneration: 5},
			expected: 0,
		},
		{
			name:     "positive drift",
			res:      Resource{Generation: 7, ObservedGeneration: 5},
			expected: 2,
		},
		{
			name:     "zero generation",
			res:      Resource{Generation: 0, ObservedGeneration: 0},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.res.Drift()
			if result != tt.expected {
				t.Errorf("Drift() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestResource_IsReconciling(t *testing.T) {
	tests := []struct {
		name     string
		res      Resource
		expected bool
	}{
		{
			name:     "not reconciling (idle)",
			res:      Resource{Generation: 5, ObservedGeneration: 5},
			expected: false,
		},
		{
			name:     "reconciling (drift > 0 with observedGeneration)",
			res:      Resource{Generation: 6, ObservedGeneration: 5},
			expected: true,
		},
		{
			name:     "not reconciling (no observedGeneration)",
			res:      Resource{Generation: 5, ObservedGeneration: 0},
			expected: false, // No observedGeneration means we can't tell
		},
		{
			name:     "not reconciling (both zero)",
			res:      Resource{Generation: 0, ObservedGeneration: 0},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.res.IsReconciling()
			if result != tt.expected {
				t.Errorf("IsReconciling() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestResource_HasObservedGeneration(t *testing.T) {
	tests := []struct {
		name     string
		res      Resource
		expected bool
	}{
		{
			name:     "has observedGeneration",
			res:      Resource{ObservedGeneration: 5},
			expected: true,
		},
		{
			name:     "no observedGeneration",
			res:      Resource{ObservedGeneration: 0},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.res.HasObservedGeneration()
			if result != tt.expected {
				t.Errorf("HasObservedGeneration() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestResource_ReadyStatus(t *testing.T) {
	tests := []struct {
		name       string
		conditions []Condition
		expected   string
	}{
		{
			name: "ready condition true",
			conditions: []Condition{
				{Type: "Ready", Status: "True"},
			},
			expected: "Ready",
		},
		{
			name: "ready condition false",
			conditions: []Condition{
				{Type: "Ready", Status: "False"},
			},
			expected: "NotReady",
		},
		{
			name: "progressing condition",
			conditions: []Condition{
				{Type: "Progressing", Status: "True"},
				{Type: "Ready", Status: "False"},
			},
			expected: "Progressing",
		},
		{
			name: "available condition true",
			conditions: []Condition{
				{Type: "Available", Status: "True"},
			},
			expected: "Ready",
		},
		{
			name:       "no conditions",
			conditions: nil,
			expected:   "Unknown",
		},
		{
			name: "synced true",
			conditions: []Condition{
				{Type: "Synced", Status: "True"},
			},
			expected: "Ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Resource{Conditions: tt.conditions}
			result := res.ReadyStatus()
			if result != tt.expected {
				t.Errorf("ReadyStatus() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestResource_ReadyIcon(t *testing.T) {
	tests := []struct {
		name     string
		res      Resource
		expected string
	}{
		{
			name: "reconciling shows spinner (with observedGeneration)",
			res: Resource{
				Generation:         6,
				ObservedGeneration: 5,
			},
			expected: "⏳",
		},
		{
			name: "no observedGeneration shows status based icon",
			res: Resource{
				Generation:         6,
				ObservedGeneration: 0, // No observedGeneration
				Conditions:         []Condition{{Type: "Ready", Status: "True"}},
			},
			expected: "✅", // Should show ready, not spinner
		},
		{
			name: "ready shows checkmark",
			res: Resource{
				Generation:         5,
				ObservedGeneration: 5,
				Conditions:         []Condition{{Type: "Ready", Status: "True"}},
			},
			expected: "✅",
		},
		{
			name: "not ready shows cross",
			res: Resource{
				Generation:         5,
				ObservedGeneration: 5,
				Conditions:         []Condition{{Type: "Ready", Status: "False"}},
			},
			expected: "❌",
		},
		{
			name: "unknown shows question mark",
			res: Resource{
				Generation:         5,
				ObservedGeneration: 5,
				Conditions:         nil,
			},
			expected: "❔",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.res.ReadyIcon()
			if result != tt.expected {
				t.Errorf("ReadyIcon() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestCondition_IsReady(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		expected  bool
	}{
		{
			name:      "Ready=True",
			condition: Condition{Type: "Ready", Status: "True"},
			expected:  true,
		},
		{
			name:      "Available=True",
			condition: Condition{Type: "Available", Status: "True"},
			expected:  true,
		},
		{
			name:      "Healthy=True",
			condition: Condition{Type: "Healthy", Status: "True"},
			expected:  true,
		},
		{
			name:      "Ready=False",
			condition: Condition{Type: "Ready", Status: "False"},
			expected:  false,
		},
		{
			name:      "Other type",
			condition: Condition{Type: "Synced", Status: "True"},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.condition.IsReady()
			if result != tt.expected {
				t.Errorf("IsReady() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestReconcileState_String(t *testing.T) {
	tests := []struct {
		state    ReconcileState
		expected string
	}{
		{ReconcileStateIdle, "Idle"},
		{ReconcileStateInFlight, "InFlight"},
		{ReconcileStateStuck, "Stuck"},
		{ReconcileStateError, "Error"},
		{ReconcileState(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if result := tt.state.String(); result != tt.expected {
				t.Errorf("String() = %s, want %s", result, tt.expected)
			}
		})
	}
}
