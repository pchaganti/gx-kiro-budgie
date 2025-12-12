package frontmatter

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type AgentMetadata struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	Capabilities []string `yaml:"capabilities"`
	UseWhen      []string `yaml:"use_when"`
	AvoidWhen    []string `yaml:"avoid_when"`
	Tools        []string `yaml:"tools"`
	Model        string   `yaml:"model"`
	Tags         []string `yaml:"tags"`
}

func LoadFromPrompt(promptsDir, agentName string) (*AgentMetadata, error) {
	promptPath := filepath.Join(promptsDir, agentName+".md")
	
	data, err := os.ReadFile(promptPath)
	if err != nil {
		return nil, err
	}

	content := string(data)
	
	// Check if file starts with frontmatter
	if !strings.HasPrefix(content, "---\n") {
		return nil, nil
	}

	// Find end of frontmatter
	parts := strings.SplitN(content[4:], "\n---\n", 2)
	if len(parts) < 2 {
		return nil, nil
	}

	var metadata AgentMetadata
	if err := yaml.Unmarshal([]byte(parts[0]), &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

func (m *AgentMetadata) EnhancedDescription() string {
	var parts []string
	
	parts = append(parts, m.Description)
	
	if len(m.Capabilities) > 0 {
		parts = append(parts, "\n\nCapabilities:")
		for _, cap := range m.Capabilities {
			parts = append(parts, "- "+cap)
		}
	}
	
	if len(m.UseWhen) > 0 {
		parts = append(parts, "\n\nUse when:")
		for _, use := range m.UseWhen {
			parts = append(parts, "- "+use)
		}
	}
	
	if len(m.AvoidWhen) > 0 {
		parts = append(parts, "\n\nAvoid when:")
		for _, avoid := range m.AvoidWhen {
			parts = append(parts, "- "+avoid)
		}
	}
	
	return strings.Join(parts, "\n")
}
