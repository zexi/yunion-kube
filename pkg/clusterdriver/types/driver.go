package types

import (
	"context"
)

// Driver defines the interface that each driver plugin should implement
type Driver interface {
	// Create the cluster
	Create(ctx context.Context, opts *DriverOptions, info *ClusterInfo) (*ClusterInfo, error)

	// Update the cluster
	Update(ctx context.Context, opts *DriverOptions, info *ClusterInfo) (*ClusterInfo, error)

	// Remove the cluster
	Remove(ctx context.Context, info *ClusterInfo) error
}
