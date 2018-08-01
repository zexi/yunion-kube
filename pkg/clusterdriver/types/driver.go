package types

import (
	"context"

	"k8s.io/client-go/rest"
)

// Driver defines the interface that each driver plugin should implement
type Driver interface {
	// Create the cluster
	Create(ctx context.Context, opts *DriverOptions, info *ClusterInfo) (*ClusterInfo, error)

	// Update the cluster
	Update(ctx context.Context, opts *DriverOptions, info *ClusterInfo) (*ClusterInfo, error)

	// Remove the cluster
	Remove(ctx context.Context, info *ClusterInfo) error

	// RemoveNode remove node from the cluster
	RemoveNode(ctx context.Context, opts *DriverOptions) error

	// GetK8sRestConfig return kubernetes rest api connection config
	GetK8sRestConfig(info *ClusterInfo) (*rest.Config, error)
}
