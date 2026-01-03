package k8s

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestExtractObservedGeneration(t *testing.T) {
	tests := []struct {
		name     string
		obj      *unstructured.Unstructured
		expected int64
	}{
		{
			name: "with observedGeneration",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"status": map[string]interface{}{
						"observedGeneration": int64(5),
					},
				},
			},
			expected: 5,
		},
		{
			name: "without observedGeneration",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"status": map[string]interface{}{},
				},
			},
			expected: 0,
		},
		{
			name: "without status",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractObservedGeneration(tt.obj)
			if result != tt.expected {
				t.Errorf("ExtractObservedGeneration() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestExtractConditions(t *testing.T) {
	tests := []struct {
		name          string
		obj           *unstructured.Unstructured
		expectedCount int
		checkFirst    bool
		firstType     string
		firstStatus   string
	}{
		{
			name: "with conditions",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"type":               "Ready",
								"status":             "True",
								"reason":             "AllGood",
								"message":            "Resource is ready",
								"lastTransitionTime": "2024-01-01T12:00:00Z",
							},
							map[string]interface{}{
								"type":   "Synced",
								"status": "True",
							},
						},
					},
				},
			},
			expectedCount: 2,
			checkFirst:    true,
			firstType:     "Ready",
			firstStatus:   "True",
		},
		{
			name: "without conditions",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"status": map[string]interface{}{},
				},
			},
			expectedCount: 0,
		},
		{
			name: "without status",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{},
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractConditions(tt.obj)
			if len(result) != tt.expectedCount {
				t.Errorf("ExtractConditions() returned %d conditions, want %d", len(result), tt.expectedCount)
			}
			if tt.checkFirst && len(result) > 0 {
				if result[0].Type != tt.firstType {
					t.Errorf("First condition type = %s, want %s", result[0].Type, tt.firstType)
				}
				if result[0].Status != tt.firstStatus {
					t.Errorf("First condition status = %s, want %s", result[0].Status, tt.firstStatus)
				}
			}
		})
	}
}

func TestExtractControllerInfo(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)

	tests := []struct {
		name            string
		managedFields   []metav1.ManagedFieldsEntry
		expectedManager string
		expectTime      bool
	}{
		{
			name: "with status subresource",
			managedFields: []metav1.ManagedFieldsEntry{
				{
					Manager:     "argocd-application-controller",
					Subresource: "status",
					Time:        &metav1.Time{Time: now},
				},
				{
					Manager: "kubectl",
					Time:    &metav1.Time{Time: earlier},
				},
			},
			expectedManager: "argocd-application-controller",
			expectTime:      true,
		},
		{
			name: "with controller pattern in manager name and status subresource",
			managedFields: []metav1.ManagedFieldsEntry{
				{
					Manager:     "crossplane-controller",
					Subresource: "status",
					Time:        &metav1.Time{Time: now},
				},
			},
			expectedManager: "crossplane-controller",
			expectTime:      true,
		},
		{
			name:            "empty managed fields",
			managedFields:   []metav1.ManagedFieldsEntry{},
			expectedManager: "",
			expectTime:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, lastWrite := ExtractControllerInfo(tt.managedFields)
			if manager != tt.expectedManager {
				t.Errorf("ExtractControllerInfo() manager = %s, want %s", manager, tt.expectedManager)
			}
			if tt.expectTime && lastWrite.IsZero() {
				t.Error("ExtractControllerInfo() returned zero time, expected non-zero")
			}
		})
	}
}

func TestShortenManagerName(t *testing.T) {
	tests := []struct {
		name     string
		manager  string
		expected string
	}{
		{
			name:     "argocd application controller",
			manager:  "argocd-application-controller",
			expected: "argocd",
		},
		{
			name:     "crossplane kubernetes",
			manager:  "crossplane-kubernetes.crossplane.io",
			expected: "crossplane",
		},
		{
			name:     "helm controller",
			manager:  "helm-controller",
			expected: "helm",
		},
		{
			name:     "empty manager",
			manager:  "",
			expected: "-",
		},
		{
			name:     "short manager",
			manager:  "kubectl",
			expected: "kubectl",
		},
		{
			name:     "long manager gets truncated",
			manager:  "very-long-controller-manager-name-here",
			expected: "very-long-co...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShortenManagerName(tt.manager)
			if result != tt.expected {
				t.Errorf("ShortenManagerName(%s) = %s, want %s", tt.manager, result, tt.expected)
			}
		})
	}
}
