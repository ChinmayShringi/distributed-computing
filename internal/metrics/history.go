// Package metrics provides in-memory metrics history storage for activity tracking
package metrics

import (
	"sync"
	"time"
)

const (
	// MaxHistoryPoints is the maximum number of samples to keep per device
	MaxHistoryPoints = 120 // 2 minutes at 1 sample/second

	// CleanupInterval is how often to run cleanup of old samples
	CleanupInterval = 5 * time.Minute

	// MetricsRetention is how long to keep metrics after last update
	MetricsRetention = 10 * time.Minute
)

// MetricsSample represents a single metrics snapshot
type MetricsSample struct {
	Timestamp     int64   `json:"timestamp_ms"`
	CPULoad       float64 `json:"cpu_load"`
	MemUsedMB     uint64  `json:"mem_used_mb"`
	MemTotalMB    uint64  `json:"mem_total_mb"`
	GPULoad       float64 `json:"gpu_load"`
	GPUMemUsedMB  uint64  `json:"gpu_mem_used_mb"`
	GPUMemTotalMB uint64  `json:"gpu_mem_total_mb"`
	NPULoad       float64 `json:"npu_load"`
}

// DeviceMetricsHistory stores historical metrics for a single device
type DeviceMetricsHistory struct {
	DeviceID   string
	DeviceName string
	Samples    []MetricsSample
	LastUpdate time.Time
	mu         sync.RWMutex
}

// MetricsStore manages metrics history for all devices
type MetricsStore struct {
	devices map[string]*DeviceMetricsHistory
	mu      sync.RWMutex
	stopCh  chan struct{}
}

// NewMetricsStore creates a new metrics store with background cleanup
func NewMetricsStore() *MetricsStore {
	s := &MetricsStore{
		devices: make(map[string]*DeviceMetricsHistory),
		stopCh:  make(chan struct{}),
	}

	// Start background cleanup goroutine
	go s.cleanupLoop()

	return s
}

// Stop stops the background cleanup goroutine
func (s *MetricsStore) Stop() {
	close(s.stopCh)
}

// AddSample adds a metrics sample for a device
func (s *MetricsStore) AddSample(deviceID, deviceName string, sample MetricsSample) {
	s.mu.Lock()
	history, exists := s.devices[deviceID]
	if !exists {
		history = &DeviceMetricsHistory{
			DeviceID:   deviceID,
			DeviceName: deviceName,
			Samples:    make([]MetricsSample, 0, MaxHistoryPoints),
		}
		s.devices[deviceID] = history
	}
	s.mu.Unlock()

	history.mu.Lock()
	defer history.mu.Unlock()

	// Update device name if provided
	if deviceName != "" {
		history.DeviceName = deviceName
	}

	// Append sample
	history.Samples = append(history.Samples, sample)

	// Trim to max size
	if len(history.Samples) > MaxHistoryPoints {
		// Remove oldest samples
		excess := len(history.Samples) - MaxHistoryPoints
		history.Samples = history.Samples[excess:]
	}

	history.LastUpdate = time.Now()
}

// GetHistory returns metrics history for a device since a given timestamp
func (s *MetricsStore) GetHistory(deviceID string, sinceMs int64) []MetricsSample {
	s.mu.RLock()
	history, exists := s.devices[deviceID]
	s.mu.RUnlock()

	if !exists {
		return nil
	}

	history.mu.RLock()
	defer history.mu.RUnlock()

	if sinceMs <= 0 {
		// Return all samples
		result := make([]MetricsSample, len(history.Samples))
		copy(result, history.Samples)
		return result
	}

	// Return samples after sinceMs
	var result []MetricsSample
	for _, sample := range history.Samples {
		if sample.Timestamp > sinceMs {
			result = append(result, sample)
		}
	}
	return result
}

// GetLatest returns the most recent metrics sample for a device
func (s *MetricsStore) GetLatest(deviceID string) *MetricsSample {
	s.mu.RLock()
	history, exists := s.devices[deviceID]
	s.mu.RUnlock()

	if !exists {
		return nil
	}

	history.mu.RLock()
	defer history.mu.RUnlock()

	if len(history.Samples) == 0 {
		return nil
	}

	// Return copy of latest sample
	sample := history.Samples[len(history.Samples)-1]
	return &sample
}

// GetAllDeviceIDs returns all device IDs with metrics
func (s *MetricsStore) GetAllDeviceIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.devices))
	for id := range s.devices {
		ids = append(ids, id)
	}
	return ids
}

// GetDeviceInfo returns device ID and name for a device
func (s *MetricsStore) GetDeviceInfo(deviceID string) (name string, exists bool) {
	s.mu.RLock()
	history, exists := s.devices[deviceID]
	s.mu.RUnlock()

	if !exists {
		return "", false
	}

	history.mu.RLock()
	defer history.mu.RUnlock()

	return history.DeviceName, true
}

// GetAllHistory returns metrics history for all devices
func (s *MetricsStore) GetAllHistory(sinceMs int64) map[string][]MetricsSample {
	s.mu.RLock()
	deviceIDs := make([]string, 0, len(s.devices))
	for id := range s.devices {
		deviceIDs = append(deviceIDs, id)
	}
	s.mu.RUnlock()

	result := make(map[string][]MetricsSample)
	for _, id := range deviceIDs {
		samples := s.GetHistory(id, sinceMs)
		if len(samples) > 0 {
			result[id] = samples
		}
	}
	return result
}

// Clear removes all metrics data
func (s *MetricsStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.devices = make(map[string]*DeviceMetricsHistory)
}

// ClearDevice removes metrics data for a specific device
func (s *MetricsStore) ClearDevice(deviceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.devices, deviceID)
}

// cleanupLoop periodically removes stale device metrics
func (s *MetricsStore) cleanupLoop() {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

// cleanup removes devices that haven't been updated recently
func (s *MetricsStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-MetricsRetention)

	for id, history := range s.devices {
		history.mu.RLock()
		lastUpdate := history.LastUpdate
		history.mu.RUnlock()

		if lastUpdate.Before(cutoff) {
			delete(s.devices, id)
		}
	}
}

// DeviceMetricsSummary provides a summary of a device's current metrics
type DeviceMetricsSummary struct {
	DeviceID     string         `json:"device_id"`
	DeviceName   string         `json:"device_name"`
	Latest       *MetricsSample `json:"latest,omitempty"`
	SampleCount  int            `json:"sample_count"`
	OldestSample int64          `json:"oldest_sample_ms,omitempty"`
	NewestSample int64          `json:"newest_sample_ms,omitempty"`
}

// GetSummary returns a summary of metrics for a device
func (s *MetricsStore) GetSummary(deviceID string) *DeviceMetricsSummary {
	s.mu.RLock()
	history, exists := s.devices[deviceID]
	s.mu.RUnlock()

	if !exists {
		return nil
	}

	history.mu.RLock()
	defer history.mu.RUnlock()

	summary := &DeviceMetricsSummary{
		DeviceID:    deviceID,
		DeviceName:  history.DeviceName,
		SampleCount: len(history.Samples),
	}

	if len(history.Samples) > 0 {
		latest := history.Samples[len(history.Samples)-1]
		summary.Latest = &latest
		summary.OldestSample = history.Samples[0].Timestamp
		summary.NewestSample = latest.Timestamp
	}

	return summary
}

// GetAllSummaries returns summaries for all devices
func (s *MetricsStore) GetAllSummaries() []*DeviceMetricsSummary {
	s.mu.RLock()
	deviceIDs := make([]string, 0, len(s.devices))
	for id := range s.devices {
		deviceIDs = append(deviceIDs, id)
	}
	s.mu.RUnlock()

	summaries := make([]*DeviceMetricsSummary, 0, len(deviceIDs))
	for _, id := range deviceIDs {
		if summary := s.GetSummary(id); summary != nil {
			summaries = append(summaries, summary)
		}
	}
	return summaries
}
