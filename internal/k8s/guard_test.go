package k8s_test

import (
	"context"
	"testing"

	"github.com/senet/gar-cleanup/internal/k8s"
)

func TestNoopGuard_ProtectedDigests(t *testing.T) {
	g := k8s.NoopGuard{}
	digests, err := g.ProtectedDigests(context.Background())
	if err != nil {
		t.Fatalf("NoopGuard.ProtectedDigests() returned error: %v", err)
	}
	if len(digests) != 0 {
		t.Errorf("expected empty map, got %v", digests)
	}
}

func TestNoopGuard_ImplementsGuardInterface(t *testing.T) {
	var _ k8s.Guard = k8s.NoopGuard{}
}

func TestClusterGuard_ProtectedDigests_Stub(t *testing.T) {
	g := &k8s.ClusterGuard{
		Kubeconfigs: []string{"/path/to/kubeconfig"},
	}
	digests, err := g.ProtectedDigests(context.Background())
	if err != nil {
		t.Fatalf("ClusterGuard.ProtectedDigests() returned error: %v", err)
	}
	if len(digests) != 0 {
		t.Errorf("stub should return empty map, got %v", digests)
	}
}

func TestClusterGuard_EmptyKubeconfigs(t *testing.T) {
	g := &k8s.ClusterGuard{}
	digests, err := g.ProtectedDigests(context.Background())
	if err != nil {
		t.Fatalf("ClusterGuard.ProtectedDigests() returned error: %v", err)
	}
	if digests == nil {
		t.Error("expected non-nil map")
	}
}

func TestClusterGuard_ImplementsGuardInterface(t *testing.T) {
	var _ k8s.Guard = (*k8s.ClusterGuard)(nil)
}
