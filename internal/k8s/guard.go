// Package k8s queries Kubernetes clusters to build a set of protected image
// digests — any digest referenced by a running or scaled-down workload is
// off-limits regardless of age.
package k8s

import (
	"context"
	"log/slog"
)

// Guard builds a set of image references that are actively used by
// Kubernetes workloads and must never be deleted.
type Guard interface {
	// ProtectedDigests returns the set of digests that are in use.
	// Keys are digest strings (e.g. "sha256:abc123"); values are always true.
	ProtectedDigests(ctx context.Context) (map[string]bool, error)
}

// NoopGuard is a Guard that never protects any digests.
// Use when K8s integration is not configured.
type NoopGuard struct{}

func (NoopGuard) ProtectedDigests(_ context.Context) (map[string]bool, error) {
	return map[string]bool{}, nil
}

// Compile-time interface check.
var _ Guard = NoopGuard{}

// ClusterGuard queries one or more Kubernetes clusters via client-go and
// returns all image digests referenced by Pods and ReplicaSets (including
// scaled-to-zero RS) so the engine can protect them from deletion.
//
// In phase 2 this connects via in-cluster service account or kubeconfig.
// The stub implementation logs a warning and returns an empty set so the
// engine can still run without a live cluster.
type ClusterGuard struct {
	// Kubeconfigs is a list of kubeconfig file paths (one per cluster).
	// If empty, in-cluster config is attempted.
	Kubeconfigs []string
}

// ProtectedDigests queries all configured clusters in parallel and unions
// the result sets.
func (g *ClusterGuard) ProtectedDigests(ctx context.Context) (map[string]bool, error) {
	// TODO(phase2): iterate g.Kubeconfigs, build clientsets with client-go,
	// list Pods and ReplicaSets, extract image references, resolve digests
	// via the registry API.
	slog.WarnContext(ctx, "K8s guard is a stub; no digests will be protected from K8s clusters")
	return map[string]bool{}, nil
}

var _ Guard = (*ClusterGuard)(nil)
