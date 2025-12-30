package search

import (
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/pteich/crdlens/internal/types"
)

// MatchResources filters a list of resources using fuzzy search on Name and Namespace
func MatchResources(query string, resources []types.Resource) []types.Resource {
	if query == "" {
		return resources
	}

	var matched []types.Resource
	for _, res := range resources {
		if fuzzy.MatchFold(query, res.Name) || fuzzy.MatchFold(query, res.Namespace) {
			matched = append(matched, res)
		}
	}
	return matched
}

// MatchCRDs filters a list of CRDs using fuzzy search on Name, Kind, and Group
func MatchCRDs(query string, crds []types.CRDInfo) []types.CRDInfo {
	if query == "" {
		return crds
	}

	var matched []types.CRDInfo
	for _, crd := range crds {
		if fuzzy.MatchFold(query, crd.Name) ||
			fuzzy.MatchFold(query, crd.Kind) ||
			fuzzy.MatchFold(query, crd.Group) {
			matched = append(matched, crd)
		}
	}
	return matched
}
