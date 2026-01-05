package types

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
		{
			name:     "high generation missing observed generation (legacy/synced assumption)",
			res:      Resource{Generation: 999, ObservedGeneration: 0},
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
		conditions []interface{}
		expected   string
	}{
		{
			name: "ready condition true",
			conditions: []interface{}{
				map[string]interface{}{"type": "Ready", "status": "True"},
			},
			expected: "Ready",
		},
		{
			name: "ready condition false (implies progressing by default in kstatus)",
			conditions: []interface{}{
				map[string]interface{}{"type": "Ready", "status": "False"},
			},
			expected: "Progressing",
		},
		{
			name: "stalled condition (implies failed)",
			conditions: []interface{}{
				map[string]interface{}{
					"type":   "Stalled",
					"status": "True",
				},
			},
			expected: "NotReady",
		},
		{
			name: "progressing condition",
			conditions: []interface{}{
				map[string]interface{}{"type": "Progressing", "status": "True"},
				map[string]interface{}{"type": "Ready", "status": "False"},
			},
			expected: "Progressing", // kstatus prioritizes Progressing
		},
		{
			name:       "no conditions (defaults to ready/current for generic resources)",
			conditions: nil,
			expected:   "Ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Resource{
				Raw: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"status": map[string]interface{}{
							"conditions": tt.conditions,
						},
					},
				},
			}
			result := res.ReadyStatus()
			if result != tt.expected {
				t.Errorf("ReadyStatus() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestResource_ReadyIcon(t *testing.T) {
	tests := []struct {
		name       string
		generation int64
		observed   int64
		conditions []interface{}
		expected   string
	}{
		// Note: kstatus doesn't strictly follow the custom HasObservedGeneration && IsReconciling logic
		// exactly the same way (it computes a holistic status).
		// However, missing ObservedGeneration might yield a different status if spec.generation matches (or not).
		// For simplicity, we test the output of ReadyIcon which now delegates to kstatus.

		{
			name:       "ready shows checkmark",
			generation: 5,
			observed:   5,
			conditions: []interface{}{
				map[string]interface{}{"type": "Ready", "status": "True"},
			},
			expected: "✅",
		},
		{
			name:       "not ready shows hourglass (default for Ready=False)",
			generation: 5,
			observed:   5,
			conditions: []interface{}{
				map[string]interface{}{"type": "Ready", "status": "False"},
			},
			expected: "⏳",
		},
		{
			name:       "unknown shows checkmark (default for no conditions)",
			generation: 5,
			observed:   5,
			conditions: nil,
			expected:   "✅",
		},
		{
			name:       "progressing shows hourglass",
			generation: 6,
			observed:   5,
			conditions: []interface{}{
				map[string]interface{}{"type": "Progressing", "status": "True"},
			},
			expected: "⏳",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Resource{
				Generation:         tt.generation,
				ObservedGeneration: tt.observed,
				Raw: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"metadata": map[string]interface{}{
							"generation": tt.generation,
						},
						"status": map[string]interface{}{
							"observedGeneration": tt.observed,
							"conditions":         tt.conditions,
						},
					},
				},
			}
			result := res.ReadyIcon()
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
