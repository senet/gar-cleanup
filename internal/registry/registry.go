// Package registry defines the interface for container registry operations.
package registry

import (
	"context"
	"time"
)

// Image represents a registry image with its tags and digest.
type Image struct {
	// Repository is the full repository path, e.g.
	// us-central1-docker.pkg.dev/my-project/my-repo/my-image
	Repository string
	// Digest is the content-addressable sha256 digest.
	Digest string
	// Tags associated with this digest.
	Tags []string
	// PushedAt is the time the image was pushed.
	PushedAt time.Time
	// SizeBytes is the compressed image size in bytes.
	SizeBytes int64
}

// Client is the interface for interacting with a container registry.
type Client interface {
	// ListRepositories returns all repository paths under the given parent.
	ListRepositories(ctx context.Context, parent string) ([]string, error)
	// ListImages returns all images in a repository.
	ListImages(ctx context.Context, repository string) ([]Image, error)
	// DeleteImage removes a specific image digest from the registry.
	// If dryRun is true, the deletion is logged but not executed.
	DeleteImage(ctx context.Context, repository, digest string, dryRun bool) error
	// UntagImage removes a tag without deleting the underlying digest.
	UntagImage(ctx context.Context, repository, digest, tag string, dryRun bool) error
}
