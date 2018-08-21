package cache

import (
	"time"

	"yunion.io/x/yunion-kube/pkg/helm/data"
	"yunion.io/x/yunion-kube/pkg/helm/jobs"
)

type refreshChartsData struct {
	chartsImplementation data.ICharts
	frequency            time.Duration
	name                 string
	firstRun             bool
}

// NewRefreshChartsData creates a new Periodic implementation that refreshes charts data
func NewRefreshChartsData(
	chartsImplementation data.ICharts,
	frequency time.Duration,
	name string,
	firstRun bool,
) jobs.Periodic {
	return &refreshChartsData{
		chartsImplementation: chartsImplementation,
		frequency:            frequency,
		name:                 name,
		firstRun:             firstRun,
	}
}

// Do implements the Periodic interface
func (r *refreshChartsData) Do() error {
	if err := r.chartsImplementation.Refresh(); err != nil {
		return err
	}
	return nil
}

// Frequency implements the Periodic interface
func (r *refreshChartsData) Frequency() time.Duration {
	return r.frequency
}

// FirstRun implements the Periodic interface
func (r *refreshChartsData) FirstRun() bool {
	return r.firstRun
}

// Name implements the Periodic interface
func (r *refreshChartsData) Name() string {
	return r.name
}
