package tool

import (
	"fmt"
	"os"
	"path/filepath"
)

// SkillInput defines the input parameters for the Skill tool.
type SkillInput struct {
	Name string `json:"name" description:"Optional skill name to read. If empty, lists all available skills."`
}

func Skill(input SkillInput) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	skillDir := filepath.Join(home, ".config", "rai")

	if input.Name == "" {
		entries, err := os.ReadDir(skillDir)
		if err != nil {
			return fmt.Sprintf("Error reading skill directory: %v", err)
		}

		var names []string
		for _, e := range entries {
			if !e.IsDir() {
				names = append(names, e.Name())
			}
		}

		if len(names) == 0 {
			return "No skills found"
		}

		result := fmt.Sprintf("Available skills (%d):\n", len(names))
		for _, name := range names {
			result += fmt.Sprintf("- %s\n", name)
		}
		return result
	}

	content, err := os.ReadFile(filepath.Join(skillDir, input.Name))
	if err != nil {
		return fmt.Sprintf("Error reading skill %q: %v", input.Name, err)
	}
	return fmt.Sprintf("Skill: %s\n\n%s", input.Name, string(content))
}
