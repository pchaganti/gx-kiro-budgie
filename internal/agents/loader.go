package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Agent struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func Load(agentsDir string) ([]Agent, error) {
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return nil, err
	}

	var agents []Agent
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(agentsDir, entry.Name()))
		if err != nil {
			continue
		}

		var agent Agent
		if err := json.Unmarshal(data, &agent); err != nil {
			continue
		}

		if agent.Name == "" {
			continue
		}

		agents = append(agents, agent)
	}

	return agents, nil
}

func FilterDescription(desc string) string {
	prefix := "sub-agent:"
	if strings.HasPrefix(strings.ToLower(desc), prefix) {
		return strings.TrimSpace(desc[len(prefix):])
	}
	return desc
}

func IsSubAgent(desc string) bool {
	prefix := "sub-agent:"
	return strings.HasPrefix(strings.ToLower(desc), prefix)
}

func NormalizeToolName(name string, toolPrefix string) string {
	var result strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-' {
			result.WriteRune(r)
		} else {
			result.WriteRune('-')
		}
	}
	return toolPrefix + result.String()
}
