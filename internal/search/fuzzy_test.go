package search

import (
	"testing"

	"github.com/pteich/crdlens/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestMatchResources(t *testing.T) {
	resources := []types.Resource{
		{Name: "nginx-deployment", Namespace: "default"},
		{Name: "redis-master", Namespace: "redis"},
		{Name: "postgres", Namespace: "db"},
	}

	// Exact match
	matched := MatchResources("nginx", resources)
	assert.Len(t, matched, 1)
	assert.Equal(t, "nginx-deployment", matched[0].Name)

	// Fuzzy match
	matched = MatchResources("ndep", resources)
	assert.Len(t, matched, 1)
	assert.Equal(t, "nginx-deployment", matched[0].Name)

	// Namespace match
	matched = MatchResources("redis", resources)
	assert.Len(t, matched, 1)
	assert.Equal(t, "redis-master", matched[0].Name)

	// Empty query
	matched = MatchResources("", resources)
	assert.Len(t, matched, 3)

	// No match
	matched = MatchResources("nonexistent", resources)
	assert.Len(t, matched, 0)
}

func TestMatchCRDs(t *testing.T) {
	crds := []types.CRDInfo{
		{Name: "certificates.cert-manager.io", Kind: "Certificate", Group: "cert-manager.io"},
		{Name: "prometheuses.monitoring.coreos.com", Kind: "Prometheus", Group: "monitoring.coreos.com"},
	}

	// Match by name
	matched := MatchCRDs("cert", crds)
	assert.Len(t, matched, 1)
	assert.Equal(t, "Certificate", matched[0].Kind)

	// Match by kind
	matched = MatchCRDs("Prom", crds)
	assert.Len(t, matched, 1)
	assert.Equal(t, "Prometheus", matched[0].Kind)

	// Match by group
	matched = MatchCRDs("monitoring", crds)
	assert.Len(t, matched, 1)
	assert.Equal(t, "Prometheus", matched[0].Kind)

	// Empty query
	matched = MatchCRDs("", crds)
	assert.Len(t, matched, 2)
}
