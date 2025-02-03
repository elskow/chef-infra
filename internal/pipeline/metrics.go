package pipeline

import (
	"sync"
	"time"
)

type BuildMetrics struct {
	StartTime      time.Time
	EndTime        time.Time
	BuildDuration  time.Duration
	DeployDuration time.Duration
	Status         string
	ErrorCount     int
	WarningCount   int
}

type MetricsCollector struct {
	metrics map[string]*BuildMetrics
	mu      sync.RWMutex
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]*BuildMetrics),
	}
}

func (mc *MetricsCollector) StartBuild(buildID string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics[buildID] = &BuildMetrics{
		StartTime: time.Now(),
		Status:    "running",
	}
}

func (mc *MetricsCollector) EndBuild(buildID string, status string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if m, exists := mc.metrics[buildID]; exists {
		m.EndTime = time.Now()
		m.BuildDuration = m.EndTime.Sub(m.StartTime)
		m.Status = status
	}
}
