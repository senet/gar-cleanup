package registry_test

import (
	"context"
	"testing"
	"time"

	"github.com/senet/gar-cleanup/internal/registry"
)

// --- NewGARClient ---

func TestNewGARClient_Valid(t *testing.T) {
	c, err := registry.NewGARClient("my-project", "us-central1")
	if err != nil {
		t.Fatalf("NewGARClient() returned error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewGARClient_EmptyProject(t *testing.T) {
	_, err := registry.NewGARClient("", "us-central1")
	if err == nil {
		t.Fatal("expected error for empty project")
	}
}

func TestNewGARClient_EmptyLocation(t *testing.T) {
	c, err := registry.NewGARClient("my-project", "")
	if err != nil {
		t.Fatalf("NewGARClient() returned error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client with default location")
	}
}

// --- GARClient stubs ---

func TestGARClient_ListRepositories_Stub(t *testing.T) {
	c, _ := registry.NewGARClient("my-project", "us-central1")
	repos, err := c.ListRepositories(context.Background(), "projects/my-project/locations/us-central1")
	if err != nil {
		t.Fatalf("ListRepositories() returned error: %v", err)
	}
	if repos != nil {
		t.Errorf("stub should return nil, got %v", repos)
	}
}

func TestGARClient_ListImages_Stub(t *testing.T) {
	c, _ := registry.NewGARClient("my-project", "us-central1")
	images, err := c.ListImages(context.Background(), "us-central1-docker.pkg.dev/my-project/repo/app")
	if err != nil {
		t.Fatalf("ListImages() returned error: %v", err)
	}
	if images != nil {
		t.Errorf("stub should return nil, got %v", images)
	}
}

func TestGARClient_DeleteImage_DryRun(t *testing.T) {
	c, _ := registry.NewGARClient("my-project", "us-central1")
	err := c.DeleteImage(context.Background(), "repo", "sha256:abc", true)
	if err != nil {
		t.Fatalf("DeleteImage(dryRun=true) returned error: %v", err)
	}
}

func TestGARClient_DeleteImage_Live(t *testing.T) {
	c, _ := registry.NewGARClient("my-project", "us-central1")
	err := c.DeleteImage(context.Background(), "repo", "sha256:abc", false)
	if err != nil {
		t.Fatalf("DeleteImage(dryRun=false) returned error: %v", err)
	}
}

func TestGARClient_UntagImage_DryRun(t *testing.T) {
	c, _ := registry.NewGARClient("my-project", "us-central1")
	err := c.UntagImage(context.Background(), "repo", "sha256:abc", "v1.0.0", true)
	if err != nil {
		t.Fatalf("UntagImage(dryRun=true) returned error: %v", err)
	}
}

func TestGARClient_UntagImage_Live(t *testing.T) {
	c, _ := registry.NewGARClient("my-project", "us-central1")
	err := c.UntagImage(context.Background(), "repo", "sha256:abc", "v1.0.0", false)
	if err != nil {
		t.Fatalf("UntagImage(dryRun=false) returned error: %v", err)
	}
}

// --- FakeClient ---

func TestFakeClient_ListRepositories(t *testing.T) {
	f := &registry.FakeClient{
		Repos: []string{"repo1", "repo2"},
	}
	repos, err := f.ListRepositories(context.Background(), "parent")
	if err != nil {
		t.Fatalf("ListRepositories() returned error: %v", err)
	}
	if len(repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(repos))
	}
}

func TestFakeClient_ListImages(t *testing.T) {
	now := time.Now()
	f := &registry.FakeClient{
		Images: map[string][]registry.Image{
			"repo1": {
				{Digest: "sha256:aaa", Tags: []string{"v1"}, PushedAt: now},
				{Digest: "sha256:bbb", PushedAt: now},
			},
		},
	}
	images, err := f.ListImages(context.Background(), "repo1")
	if err != nil {
		t.Fatalf("ListImages() returned error: %v", err)
	}
	if len(images) != 2 {
		t.Errorf("expected 2 images, got %d", len(images))
	}
}

func TestFakeClient_ListImages_UnknownRepo(t *testing.T) {
	f := &registry.FakeClient{
		Images: map[string][]registry.Image{},
	}
	images, err := f.ListImages(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("ListImages() returned error: %v", err)
	}
	if len(images) != 0 {
		t.Errorf("expected 0 images for unknown repo, got %d", len(images))
	}
}

func TestFakeClient_DeleteImage_Live(t *testing.T) {
	f := &registry.FakeClient{}
	err := f.DeleteImage(context.Background(), "repo1", "sha256:aaa", false)
	if err != nil {
		t.Fatalf("DeleteImage() returned error: %v", err)
	}
	if len(f.Deleted) != 1 || f.Deleted[0] != "sha256:aaa" {
		t.Errorf("expected Deleted=[sha256:aaa], got %v", f.Deleted)
	}
}

func TestFakeClient_DeleteImage_DryRun(t *testing.T) {
	f := &registry.FakeClient{}
	err := f.DeleteImage(context.Background(), "repo1", "sha256:aaa", true)
	if err != nil {
		t.Fatalf("DeleteImage() returned error: %v", err)
	}
	if len(f.Deleted) != 0 {
		t.Errorf("expected no deletions in dry-run, got %v", f.Deleted)
	}
}

func TestFakeClient_UntagImage_Live(t *testing.T) {
	f := &registry.FakeClient{}
	err := f.UntagImage(context.Background(), "repo1", "sha256:aaa", "latest", false)
	if err != nil {
		t.Fatalf("UntagImage() returned error: %v", err)
	}
	if len(f.Untagged) != 1 || f.Untagged[0] != "sha256:aaa:latest" {
		t.Errorf("expected Untagged=[sha256:aaa:latest], got %v", f.Untagged)
	}
}

func TestFakeClient_UntagImage_DryRun(t *testing.T) {
	f := &registry.FakeClient{}
	err := f.UntagImage(context.Background(), "repo1", "sha256:aaa", "latest", true)
	if err != nil {
		t.Fatalf("UntagImage() returned error: %v", err)
	}
	if len(f.Untagged) != 0 {
		t.Errorf("expected no untagging in dry-run, got %v", f.Untagged)
	}
}

func TestFakeClient_MultipleOperations(t *testing.T) {
	f := &registry.FakeClient{}
	ctx := context.Background()

	_ = f.DeleteImage(ctx, "repo", "sha256:aaa", false)
	_ = f.DeleteImage(ctx, "repo", "sha256:bbb", false)
	_ = f.UntagImage(ctx, "repo", "sha256:ccc", "v1", false)
	_ = f.UntagImage(ctx, "repo", "sha256:ccc", "v2", false)

	if len(f.Deleted) != 2 {
		t.Errorf("expected 2 deletions, got %d", len(f.Deleted))
	}
	if len(f.Untagged) != 2 {
		t.Errorf("expected 2 untags, got %d", len(f.Untagged))
	}
}

// --- Interface checks ---

func TestGARClient_ImplementsClientInterface(t *testing.T) {
	var _ registry.Client = (*registry.GARClient)(nil)
}

func TestFakeClient_ImplementsClientInterface(t *testing.T) {
	var _ registry.Client = (*registry.FakeClient)(nil)
}
