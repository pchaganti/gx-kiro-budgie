package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFilterDescription(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"sub-agent: Test description", "Test description"},
		{"Sub-Agent: Test description", "Test description"},
		{"SUB-AGENT: Test description", "Test description"},
		{"Regular description", "Regular description"},
		{"", ""},
	}

	for _, tt := range tests {
		result := FilterDescription(tt.input)
		if result != tt.expected {
			t.Errorf("FilterDescription(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestIsSubAgent(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"sub-agent: Test description", true},
		{"Sub-Agent: Test description", true},
		{"SUB-AGENT: Test description", true},
		{"sub-agent:", true},
		{"Regular description", false},
		{"Kubernetes Expert", false},
		{"", false},
	}

	for _, tt := range tests {
		result := IsSubAgent(tt.input)
		if result != tt.expected {
			t.Errorf("IsSubAgent(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestNormalizeToolName(t *testing.T) {
	tests := []struct {
		input      string
		toolPrefix string
		expected   string
	}{
		{"test-agent", "kiro-agent.", "kiro-agent.test-agent"},
		{"TestAgent", "kiro-agent.", "kiro-agent.testagent"},
		{"test_agent", "kiro-agent.", "kiro-agent.test_agent"},
		{"test.agent", "kiro-agent.", "kiro-agent.test.agent"},
		{"test agent", "kiro-agent.", "kiro-agent.test-agent"},
		{"test@agent", "kiro-agent.", "kiro-agent.test-agent"},
		{"test", "custom.", "custom.test"},
	}

	for _, tt := range tests {
		result := NormalizeToolName(tt.input, tt.toolPrefix)
		if result != tt.expected {
			t.Errorf("NormalizeToolName(%q, %q) = %q, want %q", tt.input, tt.toolPrefix, result, tt.expected)
		}
	}
}

func TestLoad(t *testing.T) {
	tmpDir := t.TempDir()

	agent1 := Agent{Name: "agent1", Description: "Test agent 1"}
	agent2 := Agent{Name: "agent2", Description: "sub-agent: Test agent 2"}
	
	writeAgent(t, tmpDir, "agent1.json", agent1)
	writeAgent(t, tmpDir, "agent2.json", agent2)
	
	// Invalid JSON
	os.WriteFile(filepath.Join(tmpDir, "invalid.json"), []byte("{invalid"), 0644)
	
	// Non-JSON file
	os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("readme"), 0644)

	agents, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(agents))
	}

	if agents[0].Name != "agent1" {
		t.Errorf("Expected agent1, got %s", agents[0].Name)
	}
}

func TestLoad_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	agents, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(agents) != 0 {
		t.Errorf("Expected 0 agents, got %d", len(agents))
	}
}

func TestLoad_NoName(t *testing.T) {
	tmpDir := t.TempDir()

	agent := Agent{Name: "", Description: "No name"}
	writeAgent(t, tmpDir, "noname.json", agent)

	agents, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(agents) != 0 {
		t.Errorf("Expected 0 agents (no name should be skipped), got %d", len(agents))
	}
}

func writeAgent(t *testing.T, dir, filename string, agent Agent) {
	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("Failed to marshal agent: %v", err)
	}
	
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("Failed to write agent file: %v", err)
	}
}
