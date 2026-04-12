package engine_test

import (
	"context"
	"testing"
	"time"

	"github.com/senet/gar-cleanup/internal/engine"
	"github.com/senet/gar-cleanup/internal/policy"
	"github.com/senet/gar-cleanup/internal/registry"
)

func mustPolicy(t *testing.T, pol *policy.Policy) *policy.Policy {
	t.Helper()
	if err := pol.Validate(); err != nil {
		t.Fatalf("policy.Validate: %v", err)
	}
	return pol
}

func TestRun_DryRun_NoRepos(t *testing.T) {
	pol := mustPolicy(t, &policy.Policy{
		KeepCount:      5,
		MaxAge:         "30d",
		DeleteUntagged: true,
		DryRun:         true,
	})

	fake := &registry.FakeClient{
		Repos:  nil,
		Images: map[string][]registry.Image{},
	}

	cfg := engine.Config{
		Project:  "test-project",
		Location: "us-central1",
		DryRun:   true,
	}

	eng, err := engine.New(cfg, pol, engine.WithRegistry(fake))
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}

	if err := eng.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Deleted) != 0 {
		t.Errorf("expected 0 deletions in dry-run, got %d", len(fake.Deleted))
	}
}

func TestRun_DeletesOldUntagged(t *testing.T) {
	pol := mustPolicy(t, &policy.Policy{
		KeepCount:      2,
		MaxAge:         "30d",
		ProtectTags:    []string{`^v\d+\.\d+\.\d+$`},
		DeleteUntagged: true,
	})

	now := time.Now()
	old := now.Add(-40 * 24 * time.Hour)

	repo := "us-central1-docker.pkg.dev/proj/repo/app"
	fake := &registry.FakeClient{
		Repos: []string{repo},
		Images: map[string][]registry.Image{
			repo: {
				{Digest: "sha256:aaa", Tags: []string{}, PushedAt: old},
				{Digest: "sha256:bbb", Tags: []string{"latest"}, PushedAt: now},
				{Digest: "sha256:ccc", Tags: []string{"v1.2.3"}, PushedAt: old},
			},
		},
	}

	cfg := engine.Config{
		Project:  "proj",
		Location: "us-central1",
		DryRun:   false,
	}

	eng, err := engine.New(cfg, pol, engine.WithRegistry(fake))
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}

	if err := eng.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// sha256:aaa is untagged → deleted
	if len(fake.Deleted) != 1 || fake.Deleted[0] != "sha256:aaa" {
		t.Errorf("expected [sha256:aaa] deleted, got %v", fake.Deleted)
	}
}

func TestRun_ProtectsK8sDigests(t *testing.T) {
	pol := mustPolicy(t, &policy.Policy{
		KeepCount:      1,
		MaxAge:         "1d",
		DeleteUntagged: true,
	})

	old := time.Now().Add(-5 * 24 * time.Hour)
	repo := "us-central1-docker.pkg.dev/proj/repo/app"
	fake := &registry.FakeClient{
		Repos: []string{repo},
		Images: map[string][]registry.Image{
			repo: {
				{Digest: "sha256:k8s", Tags: []string{}, PushedAt: old},
			},
		},
	}

	// K8s guard protects sha256:k8s
	guard := &fakeGuard{digests: map[string]bool{"sha256:k8s": true}}

	cfg := engine.Config{
		Project:  "proj",
		Location: "us-central1",
		DryRun:   false,
	}

	eng, err := engine.New(cfg, pol, engine.WithRegistry(fake), engine.WithGuard(guard))
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}

	if err := eng.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Deleted) != 0 {
		t.Errorf("expected 0 deletions (K8s protected), got %v", fake.Deleted)
	}
}

type fakeGuard struct {
	digests map[string]bool
}

func (f *fakeGuard) ProtectedDigests(_ context.Context) (map[string]bool, error) {
	return f.digests, nil
}
