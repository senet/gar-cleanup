package registry

import (
	"context"
	"fmt"
	"log/slog"
)

// GARClient is a Google Artifact Registry implementation of Client.
// It wraps the Artifact Registry REST API via the google.golang.org/api SDK.
// In production this is wired to the real API; in tests it is replaced by a
// FakeClient.
type GARClient struct {
	project  string
	location string
}

// NewGARClient creates a GARClient for the given GCP project and location.
// Authentication uses Application Default Credentials (Workload Identity on
// Cloud Run, gcloud ADC locally).
func NewGARClient(project, location string) (*GARClient, error) {
	if project == "" {
		return nil, fmt.Errorf("project must not be empty")
	}
	if location == "" {
		location = "us-central1"
	}
	return &GARClient{project: project, location: location}, nil
}

// ListRepositories returns all repository paths under the project/location.
func (c *GARClient) ListRepositories(ctx context.Context, parent string) ([]string, error) {
	// TODO(phase1): implement using
	// artifactregistry.NewRESTClient(ctx) + ListRepositories RPC.
	// Stub returns empty list; real impl would call the GAR API.
	slog.InfoContext(ctx, "ListRepositories stub", "parent", parent)
	return nil, nil
}

// ListImages returns all tagged and untagged images in a repository.
func (c *GARClient) ListImages(ctx context.Context, repository string) ([]Image, error) {
	slog.InfoContext(ctx, "ListImages stub", "repository", repository)
	return nil, nil
}

// DeleteImage deletes an image digest from GAR.
func (c *GARClient) DeleteImage(ctx context.Context, repository, digest string, dryRun bool) error {
	if dryRun {
		slog.InfoContext(ctx, "[dry-run] would delete image",
			"repository", repository,
			"digest", digest)
		return nil
	}
	// TODO(phase1): call artifactregistry DeleteVersion RPC.
	slog.InfoContext(ctx, "deleting image",
		"repository", repository,
		"digest", digest)
	return nil
}

// UntagImage removes a tag from an image without deleting the digest.
func (c *GARClient) UntagImage(ctx context.Context, repository, digest, tag string, dryRun bool) error {
	if dryRun {
		slog.InfoContext(ctx, "[dry-run] would untag image",
			"repository", repository,
			"digest", digest,
			"tag", tag)
		return nil
	}
	// TODO(phase1): call artifactregistry DeleteTag RPC.
	slog.InfoContext(ctx, "untagging image",
		"repository", repository,
		"digest", digest,
		"tag", tag)
	return nil
}

// FakeClient is an in-memory Client for use in tests.
type FakeClient struct {
	Repos  []string
	Images map[string][]Image
	// Deleted records digest strings passed to DeleteImage.
	Deleted []string
	// Untagged records "digest:tag" strings passed to UntagImage.
	Untagged []string
}

func (f *FakeClient) ListRepositories(_ context.Context, _ string) ([]string, error) {
	return f.Repos, nil
}

func (f *FakeClient) ListImages(_ context.Context, repository string) ([]Image, error) {
	return f.Images[repository], nil
}

func (f *FakeClient) DeleteImage(_ context.Context, _, digest string, dryRun bool) error {
	if !dryRun {
		f.Deleted = append(f.Deleted, digest)
	}
	return nil
}

func (f *FakeClient) UntagImage(_ context.Context, _, digest, tag string, dryRun bool) error {
	if !dryRun {
		f.Untagged = append(f.Untagged, digest+":"+tag)
	}
	return nil
}

// Compile-time interface checks.
var _ Client = (*FakeClient)(nil)
var _ Client = (*GARClient)(nil)
