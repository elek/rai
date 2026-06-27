package tool

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// SkillInput defines the input parameters for the Skill tool.
type SkillInput struct {
	Name string `json:"name" description:"The exact name of the skill to load, as listed in the tool description."`
}

// skillMeta holds the discovered metadata for a single skill.
type skillMeta struct {
	Name        string
	Description string
	Path        string // path to the SKILL.md file
}

// skillFrontmatter is the YAML structure expected at the top of a SKILL.md file.
type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// defaultSkillsDir returns the directory where skills are stored:
// ~/.config/rai/skills.
func defaultSkillsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "rai", "skills")
}

// parseSkillFrontmatter splits a SKILL.md file into its YAML frontmatter fields
// (name, description) and the markdown body. If no frontmatter is present, the
// whole content is treated as the body.
func parseSkillFrontmatter(content []byte) (name, description, body string) {
	text := string(content)
	if strings.HasPrefix(text, "---\n") || strings.HasPrefix(text, "---\r\n") {
		rest := text[strings.Index(text, "\n")+1:]
		if idx := strings.Index(rest, "\n---"); idx != -1 {
			fm := rest[:idx]
			after := rest[idx+len("\n---"):]
			// Skip to the end of the closing delimiter line.
			if nl := strings.Index(after, "\n"); nl != -1 {
				body = after[nl+1:]
			}
			var parsed skillFrontmatter
			if err := yaml.Unmarshal([]byte(fm), &parsed); err == nil {
				name = parsed.Name
				description = parsed.Description
			}
			return name, description, strings.TrimSpace(body)
		}
	}
	return "", "", strings.TrimSpace(text)
}

// discoverSkills scans baseDir for subdirectories containing a SKILL.md file and
// returns metadata for each. Skills that cannot be read are skipped. The result
// is sorted by name for stable ordering.
func discoverSkills(baseDir string) []skillMeta {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil
	}

	var skills []skillMeta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(baseDir, e.Name(), "SKILL.md")
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		name, description, _ := parseSkillFrontmatter(content)
		if name == "" {
			name = e.Name()
		}
		skills = append(skills, skillMeta{
			Name:        name,
			Description: description,
			Path:        path,
		})
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})
	return skills
}

// skillCatalog builds the human-readable catalog of available skills used as the
// Skill tool's description, so the model knows what it can load.
func skillCatalog(baseDir string) string {
	skills := discoverSkills(baseDir)
	var b strings.Builder
	b.WriteString("Load a skill to get detailed, task-specific instructions. ")
	b.WriteString("Call this with the exact skill name to load its full instructions before working on a matching task.\n\n")
	if len(skills) == 0 {
		b.WriteString("No skills are currently available.")
		return b.String()
	}
	b.WriteString("Available skills:\n")
	for _, s := range skills {
		fmt.Fprintf(&b, "- %s: %s\n", s.Name, s.Description)
	}
	return b.String()
}

// loadSkill returns the markdown body (frontmatter stripped) of the named skill.
// If the skill is not found, the error lists the available skill names.
func loadSkill(baseDir, name string) (string, error) {
	skills := discoverSkills(baseDir)
	for _, s := range skills {
		if s.Name == name {
			content, err := os.ReadFile(s.Path)
			if err != nil {
				return "", errors.WithStack(err)
			}
			_, _, body := parseSkillFrontmatter(content)
			return body, nil
		}
	}

	var names []string
	for _, s := range skills {
		names = append(names, s.Name)
	}
	return "", errors.Errorf("skill %q not found. Available skills: %s", name, strings.Join(names, ", "))
}

// SkillToolDescription returns the dynamic description for the Skill tool,
// including the catalog of currently available skills.
func SkillToolDescription() string {
	return skillCatalog(defaultSkillsDir())
}

// Skill loads the named skill and returns its instructions.
func Skill(input SkillInput) string {
	body, err := loadSkill(defaultSkillsDir(), input.Name)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return fmt.Sprintf("Skill: %s\n\n%s", input.Name, body)
}
