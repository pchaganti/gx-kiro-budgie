package health

import (
	"sync"
	"time"
)

type AgentMetrics struct {
	TotalCalls    int
	SuccessCalls  int
	FailedCalls   int
	TimeoutCalls  int
	TotalDuration time.Duration
	LastSuccess   time.Time
	LastFailure   time.Time
	LastError     string
}

type Monitor struct {
	mu      sync.RWMutex
	metrics map[string]*AgentMetrics
}

func NewMonitor() *Monitor {
	return &Monitor{
		metrics: make(map[string]*AgentMetrics),
	}
}

func (m *Monitor) RecordSuccess(agent string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.metrics[agent] == nil {
		m.metrics[agent] = &AgentMetrics{}
	}

	metrics := m.metrics[agent]
	metrics.TotalCalls++
	metrics.SuccessCalls++
	metrics.TotalDuration += duration
	metrics.LastSuccess = time.Now()
}

func (m *Monitor) RecordFailure(agent string, duration time.Duration, err string, isTimeout bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.metrics[agent] == nil {
		m.metrics[agent] = &AgentMetrics{}
	}

	metrics := m.metrics[agent]
	metrics.TotalCalls++
	metrics.FailedCalls++
	metrics.TotalDuration += duration
	metrics.LastFailure = time.Now()
	metrics.LastError = err

	if isTimeout {
		metrics.TimeoutCalls++
	}
}

func (m *Monitor) GetMetrics(agent string) *AgentMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.metrics[agent] == nil {
		return &AgentMetrics{}
	}

	metrics := *m.metrics[agent]
	return &metrics
}

func (m *Monitor) GetAllMetrics() map[string]*AgentMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*AgentMetrics)
	for agent, metrics := range m.metrics {
		copy := *metrics
		result[agent] = &copy
	}
	return result
}

func (m *AgentMetrics) SuccessRate() float64 {
	if m.TotalCalls == 0 {
		return 0.0
	}
	return float64(m.SuccessCalls) / float64(m.TotalCalls)
}

func (m *AgentMetrics) AvgDuration() time.Duration {
	if m.TotalCalls == 0 {
		return 0
	}
	return m.TotalDuration / time.Duration(m.TotalCalls)
}
