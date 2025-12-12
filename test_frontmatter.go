package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"budgie/internal/frontmatter"
)

func main() {
	homeDir, _ := os.UserHomeDir()
	promptsDir := filepath.Join(homeDir, ".kiro", "sub-agents", "prompts")
	
	agentName := "codebase-locator"
	
	metadata, err := frontmatter.LoadFromPrompt(promptsDir, agentName)
	if err != nil {
		log.Fatalf("Error loading frontmatter: %v", err)
	}
	
	if metadata == nil {
		log.Fatal("No frontmatter found")
	}
	
	fmt.Printf("Agent: %s\n", metadata.Name)
	fmt.Printf("Description: %s\n\n", metadata.Description)
	fmt.Printf("Enhanced Description:\n%s\n", metadata.EnhancedDescription())
}
