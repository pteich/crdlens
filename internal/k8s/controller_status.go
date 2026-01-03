package k8s

import (
	"time"

	"github.com/pteich/crdlens/internal/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ExtractObservedGeneration extracts status.observedGeneration from an unstructured object
// Returns 0 if the field is not present
func ExtractObservedGeneration(obj *unstructured.Unstructured) int64 {
	status, found, err := unstructured.NestedMap(obj.Object, "status")
	if err != nil || !found {
		return 0
	}

	observedGen, found, err := unstructured.NestedInt64(status, "observedGeneration")
	if err != nil || !found {
		return 0
	}

	return observedGen
}

// ExtractConditions extracts status.conditions from an unstructured object
// Handles the common Kubernetes condition format used by most controllers
func ExtractConditions(obj *unstructured.Unstructured) []types.Condition {
	conditionsRaw, found, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil || !found {
		return nil
	}

	var conditions []types.Condition
	for _, c := range conditionsRaw {
		condMap, ok := c.(map[string]interface{})
		if !ok {
			continue
		}

		condition := types.Condition{
			Type:    getStringField(condMap, "type"),
			Status:  getStringField(condMap, "status"),
			Reason:  getStringField(condMap, "reason"),
			Message: getStringField(condMap, "message"),
		}

		// Parse lastTransitionTime
		if timeStr := getStringField(condMap, "lastTransitionTime"); timeStr != "" {
			if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
				condition.LastTransitionTime = t
			}
		}

		conditions = append(conditions, condition)
	}

	return conditions
}

// ExtractControllerInfo extracts controller manager name, last status write time
// and last spec write time from the object's managedFields metadata
func ExtractControllerInfo(managedFields []metav1.ManagedFieldsEntry) (manager string, lastStatusWrite, lastSpecWrite time.Time) {
	for _, mf := range managedFields {
		// Skip entries without timestamps
		if mf.Time == nil {
			continue
		}

		// Check if this entry manages status fields
		if mf.Subresource == "status" || containsStatusFields(mf) {
			if mf.Time.Time.After(lastStatusWrite) {
				lastStatusWrite = mf.Time.Time
				manager = mf.Manager
			}
		}

		// Check if this entry manages spec fields
		if containsSpecFields(mf) {
			if mf.Time.Time.After(lastSpecWrite) {
				lastSpecWrite = mf.Time.Time
			}
		}
	}

	return manager, lastStatusWrite, lastSpecWrite
}

// containsSpecFields checks if a managedFieldsEntry contains spec-related fields
func containsSpecFields(mf metav1.ManagedFieldsEntry) bool {
	if mf.FieldsV1 == nil || mf.Subresource != "" {
		return false
	}

	// In Kubernetes, spec is almost always under "f:spec"
	// For some built-in resources it might be different, but for CRs it's standard.
	// We check if "f:spec" is present in the raw JSON of FieldsV1
	return contains(string(mf.FieldsV1.Raw), "f:spec")
}

// containsStatusFields checks if a managedFieldsEntry contains status-related fields
func containsStatusFields(mf metav1.ManagedFieldsEntry) bool {
	if mf.FieldsV1 == nil {
		return false
	}

	// For status writers, they typically own "f:status"
	if contains(string(mf.FieldsV1.Raw), "f:status") {
		return true
	}

	// Fallback to manager name heuristic
	controllerPatterns := []string{
		"controller",
		"operator",
		"reconciler",
		"argocd",
		"crossplane",
		"flux",
		"helm",
		"kustomize",
		"cert-manager",
	}

	managerLower := toLower(mf.Manager)
	for _, pattern := range controllerPatterns {
		if contains(managerLower, pattern) {
			return true
		}
	}

	return false
}

// ShortenManagerName shortens a controller manager name for display
// e.g., "crossplane-kubernetes.crossplane.io" -> "crossplane-k8s"
// e.g., "argocd-application-controller" -> "argocd"
func ShortenManagerName(manager string) string {
	if manager == "" {
		return "-"
	}

	// Common shortening patterns
	shortenPatterns := map[string]string{
		"argocd-application-controller":       "argocd",
		"crossplane-kubernetes.crossplane.io": "crossplane",
		"helm-controller":                     "helm",
		"kustomize-controller":                "kustomize",
		"source-controller":                   "flux-src",
		"cert-manager-controller":             "cert-mgr",
		"cert-manager":                        "cert-mgr",
	}

	if short, ok := shortenPatterns[manager]; ok {
		return short
	}

	// Fallback: truncate to 15 chars
	if len(manager) > 15 {
		return manager[:12] + "..."
	}

	return manager
}

// getStringField safely extracts a string field from a map
func getStringField(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// toLower converts string to lowercase (simple implementation to avoid import)
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

// contains checks if s contains substr (case-sensitive)
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
