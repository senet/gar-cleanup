package engine_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/senet/gar-cleanup/internal/audit"
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

// --- Additional tests for improved coverage ---

func TestRun_KeepCountBoundary(t *testing.T) {
	pol := mustPolicy(t, &policy.Policy{
		KeepCount:      2,
		MaxAge:         "30d",
		DeleteUntagged: false,
	})

	now := time.Now()
	repo := "us-central1-docker.pkg.dev/proj/repo/app"
	fake := &registry.FakeClient{
		Repos: []string{repo},
		Images: map[string][]registry.Image{
			repo: {
				{Digest: "sha256:newest", Tags: []string{"v3"}, PushedAt: now},
				{Digest: "sha256:mid", Tags: []string{"v2"}, PushedAt: now.Add(-10 * 24 * time.Hour)},
				{Digest: "sha256:old", Tags: []string{"v1"}, PushedAt: now.Add(-40 * 24 * time.Hour)},
			},
		},
	}

	cfg := engine.Config{Project: "proj", Location: "us-central1", DryRun: false}
	eng, err := engine.New(cfg, pol, engine.WithRegistry(fake))
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}

	if err := eng.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// sha256:newest (rank 0) and sha256:mid (rank 1) are within keep_count=2
	// sha256:old (rank 2) exceeds keep_count AND max_age → deleted
	if len(fake.Deleted) != 1 || fake.Deleted[0] != "sha256:old" {
		t.Errorf("expected [sha256:old] deleted, got %v", fake.Deleted)
	}
	// sha256:old has tag "v1" so it should be untagged first
	if len(fake.Untagged) != 1 || fake.Untagged[0] != "sha256:old:v1" {
		t.Errorf("expected [sha256:old:v1] untagged, got %v", fake.Untagged)
	}
}

func TestRun_MaxAgeOnly(t *testing.T) {
	pol := mustPolicy(t, &policy.Policy{
		KeepCount:      0,
		MaxAge:         "10d",
		DeleteUntagged: false,
	})

	now := time.Now()
	repo := "repo"
	fake := &registry.FakeClient{
		Repos: []string{repo},
		Images: map[string][]registry.Image{
			repo: {
				{Digest: "sha256:new", Tags: []string{"latest"}, PushedAt: now},
				{Digest: "sha256:old", Tags: []string{"old"}, PushedAt: now.Add(-20 * 24 * time.Hour)},
			},
		},
	}

	cfg := engine.Config{Project: "proj", Location: "us-central1", DryRun: false}
	eng, err := engine.New(cfg, pol, engine.WithRegistry(fake))
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}

	if err := eng.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Deleted) != 1 || fake.Deleted[0] != "sha256:old" {
		t.Errorf("expected [sha256:old] deleted by max_age, got %v", fake.Deleted)
	}
}

func TestRun_MultipleRepositories(t *testing.T) {
	pol := mustPolicy(t, &policy.Policy{
		KeepCount:      1,
		MaxAge:         "5d",
		DeleteUntagged: true,
	})

	now := time.Now()
	old := now.Add(-10 * 24 * time.Hour)
	repo1 := "repo1"
	repo2 := "repo2"
	fake := &registry.FakeClient{
		Repos: []string{repo1, repo2},
		Images: map[string][]registry.Image{
			repo1: {
				{Digest: "sha256:r1new", Tags: []string{"latest"}, PushedAt: now},
				{Digest: "sha256:r1old", Tags: nil, PushedAt: old},
			},
			repo2: {
				{Digest: "sha256:r2new", Tags: []string{"latest"}, PushedAt: now},
				{Digest: "sha256:r2old", Tags: nil, PushedAt: old},
			},
		},
	}

	cfg := engine.Config{Project: "proj", Location: "us-central1", DryRun: false}
	eng, err := engine.New(cfg, pol, engine.WithRegistry(fake))
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}

	if err := eng.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Deleted) != 2 {
		t.Errorf("expected 2 deletions from 2 repos, got %d: %v", len(fake.Deleted), fake.Deleted)
	}
}

func TestRun_ProtectedTagPreventsMaxAgeDeletion(t *testing.T) {
	pol := mustPolicy(t, &policy.Policy{
		KeepCount:      0,
		MaxAge:         "1d",
		ProtectTags:    []string{`^v\d+\.\d+\.\d+$`},
		DeleteUntagged: false,
	})

	old := time.Now().Add(-30 * 24 * time.Hour)
	repo := "repo"
	fake := &registry.FakeClient{
		Repos: []string{repo},
		Images: map[string][]registry.Image{
			repo: {
				{Digest: "sha256:protected", Tags: []string{"v1.0.0"}, PushedAt: old},
			},
		},
	}

	cfg := engine.Config{Project: "proj", Location: "us-central1", DryRun: false}
	eng, err := engine.New(cfg, pol, engine.WithRegistry(fake))
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}

	if err := eng.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Deleted) != 0 {
		t.Errorf("protected tag should prevent deletion, got %v", fake.Deleted)
	}
}

func TestRun_WithAuditor(t *testing.T) {
	pol := mustPolicy(t, &policy.Policy{
		KeepCount:      1,
		MaxAge:         "1d",
		DeleteUntagged: true,
	})

	old := time.Now().Add(-5 * 24 * time.Hour)
	repo := "repo"
	fake := &registry.FakeClient{
		Repos: []string{repo},
		Images: map[string][]registry.Image{
			repo: {
				{Digest: "sha256:new", Tags: []string{"latest"}, PushedAt: time.Now()},
				{Digest: "sha256:old", Tags: nil, PushedAt: old, SizeBytes: 1024},
			},
		},
	}

	auditor := &fakeAuditor{}
	cfg := engine.Config{Project: "proj", Location: "us-central1", DryRun: false}
	eng, err := engine.New(cfg, pol,
		engine.WithRegistry(fake),
		engine.WithAuditor(auditor),
	)
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}

	if err := eng.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(auditor.events) == 0 {
		t.Fatal("expected at least one audit event")
	}
	found := false
	for _, e := range auditor.events {
		if e.Digest == "sha256:old" && e.Reason == "untagged" {
			found = true
			if e.SizeBytes != 1024 {
				t.Errorf("audit event SizeBytes = %d, want 1024", e.SizeBytes)
			}
		}
	}
	if !found {
		t.Errorf("expected audit event for sha256:old, got events: %+v", auditor.events)
	}
}

func TestRun_K8sProtectedImageAuditEvent(t *testing.T) {
	pol := mustPolicy(t, &policy.Policy{
		KeepCount:      0,
		MaxAge:         "1d",
		DeleteUntagged: true,
	})

	old := time.Now().Add(-5 * 24 * time.Hour)
	repo := "repo"
	fake := &registry.FakeClient{
		Repos: []string{repo},
		Images: map[string][]registry.Image{
			repo: {
				{Digest: "sha256:k8s", Tags: []string{"v1"}, PushedAt: old, SizeBytes: 2048},
			},
		},
	}

	guard := &fakeGuard{digests: map[string]bool{"sha256:k8s": true}}
	auditor := &fakeAuditor{}

	cfg := engine.Config{Project: "proj", Location: "us-central1", DryRun: false}
	eng, err := engine.New(cfg, pol,
		engine.WithRegistry(fake),
		engine.WithGuard(guard),
		engine.WithAuditor(auditor),
	)
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}

	if err := eng.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Deleted) != 0 {
		t.Errorf("k8s-protected image should not be deleted, got %v", fake.Deleted)
	}

	// Should have an audit event for k8s-protected image
	if len(auditor.events) != 1 {
		t.Fatalf("expected 1 audit event for k8s protection, got %d", len(auditor.events))
	}
	e := auditor.events[0]
	if !e.K8sProtected {
		t.Error("expected K8sProtected=true in audit event")
	}
	if e.Reason != "k8s_in_use" {
		t.Errorf("expected reason=k8s_in_use, got %s", e.Reason)
	}
}

func TestRun_LiveMode(t *testing.T) {
	pol := mustPolicy(t, &policy.Policy{
		KeepCount:      1,
		MaxAge:         "5d",
		DeleteUntagged: true,
	})

	now := time.Now()
	old := now.Add(-10 * 24 * time.Hour)
	repo := "repo"
	fake := &registry.FakeClient{
		Repos: []string{repo},
		Images: map[string][]registry.Image{
			repo: {
				{Digest: "sha256:new", Tags: []string{"latest"}, PushedAt: now},
				{Digest: "sha256:old-tagged", Tags: []string{"old-tag"}, PushedAt: old},
			},
		},
	}

	cfg := engine.Config{Project: "proj", Location: "us-central1", DryRun: false}
	eng, err := engine.New(cfg, pol, engine.WithRegistry(fake))
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}

	if err := eng.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// sha256:old-tagged exceeds keep_count and max_age, should be untagged then deleted
	if len(fake.Untagged) != 1 || fake.Untagged[0] != "sha256:old-tagged:old-tag" {
		t.Errorf("expected [sha256:old-tagged:old-tag] untagged, got %v", fake.Untagged)
	}
	if len(fake.Deleted) != 1 || fake.Deleted[0] != "sha256:old-tagged" {
		t.Errorf("expected [sha256:old-tagged] deleted, got %v", fake.Deleted)
	}
}

func TestRun_GuardError(t *testing.T) {
	pol := mustPolicy(t, &policy.Policy{
		KeepCount: 5,
		MaxAge:    "30d",
	})

	fake := &registry.FakeClient{Repos: []string{"repo"}}
	guard := &errGuard{err: fmt.Errorf("k8s connection failed")}

	cfg := engine.Config{Project: "proj", Location: "us-central1", DryRun: true}
	eng, err := engine.New(cfg, pol, engine.WithRegistry(fake), engine.WithGuard(guard))
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}

	err = eng.Run(context.Background())
	if err == nil {
		t.Fatal("expected error from guard failure")
	}
}

func TestNew_NoRegistry_EmptyProject(t *testing.T) {
	pol := mustPolicy(t, &policy.Policy{
		KeepCount: 5,
		MaxAge:    "30d",
	})

	cfg := engine.Config{Project: "", Location: "us-central1", DryRun: true}
	_, err := engine.New(cfg, pol)
	if err == nil {
		t.Fatal("expected error when creating GARClient with empty project")
	}
}

func TestRun_DryRun_NoDeletionsRecorded(t *testing.T) {
	pol := mustPolicy(t, &policy.Policy{
		KeepCount:      0,
		MaxAge:         "1d",
		DeleteUntagged: true,
	})

	old := time.Now().Add(-5 * 24 * time.Hour)
	repo := "repo"
	fake := &registry.FakeClient{
		Repos: []string{repo},
		Images: map[string][]registry.Image{
			repo: {
				{Digest: "sha256:old", Tags: nil, PushedAt: old},
			},
		},
	}

	cfg := engine.Config{Project: "proj", Location: "us-central1", DryRun: true}
	eng, err := engine.New(cfg, pol, engine.WithRegistry(fake))
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}

	if err := eng.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Dry-run: FakeClient doesn't record deletions
	if len(fake.Deleted) != 0 {
		t.Errorf("expected 0 recorded deletions in dry-run, got %v", fake.Deleted)
	}
}

func TestRun_RetainedWhenWithinKeepAndAge(t *testing.T) {
	pol := mustPolicy(t, &policy.Policy{
		KeepCount:      10,
		MaxAge:         "30d",
		DeleteUntagged: false,
	})

	now := time.Now()
	repo := "repo"
	fake := &registry.FakeClient{
		Repos: []string{repo},
		Images: map[string][]registry.Image{
			repo: {
				{Digest: "sha256:a", Tags: []string{"latest"}, PushedAt: now},
				{Digest: "sha256:b", Tags: []string{"v1"}, PushedAt: now.Add(-1 * 24 * time.Hour)},
			},
		},
	}

	cfg := engine.Config{Project: "proj", Location: "us-central1", DryRun: false}
	eng, err := engine.New(cfg, pol, engine.WithRegistry(fake))
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}

	if err := eng.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Deleted) != 0 {
		t.Errorf("expected no deletions for images within keep and age, got %v", fake.Deleted)
	}
}

func TestRun_UntaggedNotDeletedWhenPolicyDisabled(t *testing.T) {
	pol := mustPolicy(t, &policy.Policy{
		KeepCount:      0,
		MaxAge:         "30d",
		DeleteUntagged: false,
	})

	now := time.Now()
	repo := "repo"
	fake := &registry.FakeClient{
		Repos: []string{repo},
		Images: map[string][]registry.Image{
			repo: {
				{Digest: "sha256:untagged", Tags: nil, PushedAt: now},
			},
		},
	}

	cfg := engine.Config{Project: "proj", Location: "us-central1", DryRun: false}
	eng, err := engine.New(cfg, pol, engine.WithRegistry(fake))
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}

	if err := eng.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(fake.Deleted) != 0 {
		t.Errorf("expected no deletion when delete_untagged=false, got %v", fake.Deleted)
	}
}

// --- Test helpers ---

type fakeGuard struct {
	digests map[string]bool
}

func (f *fakeGuard) ProtectedDigests(_ context.Context) (map[string]bool, error) {
	return f.digests, nil
}

type errGuard struct {
	err error
}

func (g *errGuard) ProtectedDigests(_ context.Context) (map[string]bool, error) {
	return nil, g.err
}

type fakeAuditor struct {
	events []audit.Event
}

func (a *fakeAuditor) Write(_ context.Context, e audit.Event) error {
	a.events = append(a.events, e)
	return nil
}

func (a *fakeAuditor) Close() error { return nil }
