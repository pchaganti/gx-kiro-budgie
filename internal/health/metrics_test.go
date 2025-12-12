package health

import (
	"testing"
	"time"
)

func TestMonitor(t *testing.T) {
	m := NewMonitor()

	// Record success
	m.RecordSuccess("test-agent", 100*time.Millisecond)

	metrics := m.GetMetrics("test-agent")
	if metrics.TotalCalls != 1 {
		t.Errorf("Expected 1 call, got %d", metrics.TotalCalls)
	}
	if metrics.SuccessRate() != 1.0 {
		t.Errorf("Expected 100%% success, got %.2f", metrics.SuccessRate())
	}

	// Record failure
	m.RecordFailure("test-agent", 50*time.Millisecond, "test error", false)

	metrics = m.GetMetrics("test-agent")
	if metrics.TotalCalls != 2 {
		t.Errorf("Expected 2 calls, got %d", metrics.TotalCalls)
	}
	if metrics.SuccessRate() != 0.5 {
		t.Errorf("Expected 50%% success, got %.2f", metrics.SuccessRate())
	}
	if metrics.LastError != "test error" {
		t.Errorf("Expected 'test error', got '%s'", metrics.LastError)
	}
}

func TestMonitorTimeout(t *testing.T) {
	m := NewMonitor()

	m.RecordFailure("test-agent", 100*time.Millisecond, "timeout", true)

	metrics := m.GetMetrics("test-agent")
	if metrics.TimeoutCalls != 1 {
		t.Errorf("Expected 1 timeout, got %d", metrics.TimeoutCalls)
	}
}

func TestMonitorAvgDuration(t *testing.T) {
	m := NewMonitor()

	m.RecordSuccess("test-agent", 100*time.Millisecond)
	m.RecordSuccess("test-agent", 200*time.Millisecond)

	metrics := m.GetMetrics("test-agent")
	expected := 150 * time.Millisecond
	if metrics.AvgDuration() != expected {
		t.Errorf("Expected %v avg duration, got %v", expected, metrics.AvgDuration())
	}
}

func TestMonitorGetAllMetrics(t *testing.T) {
	m := NewMonitor()

	m.RecordSuccess("agent1", 100*time.Millisecond)
	m.RecordSuccess("agent2", 200*time.Millisecond)

	allMetrics := m.GetAllMetrics()
	if len(allMetrics) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(allMetrics))
	}

	if allMetrics["agent1"].TotalCalls != 1 {
		t.Errorf("Expected agent1 to have 1 call")
	}
	if allMetrics["agent2"].TotalCalls != 1 {
		t.Errorf("Expected agent2 to have 1 call")
	}
}

func TestMonitorEmptyAgent(t *testing.T) {
	m := NewMonitor()

	metrics := m.GetMetrics("nonexistent")
	if metrics.TotalCalls != 0 {
		t.Errorf("Expected 0 calls for nonexistent agent, got %d", metrics.TotalCalls)
	}
	if metrics.SuccessRate() != 0.0 {
		t.Errorf("Expected 0%% success for nonexistent agent, got %.2f", metrics.SuccessRate())
	}
}
