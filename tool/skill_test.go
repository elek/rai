package tool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeSkill creates a skill directory with a SKILL.md file holding the given content.
func writeSkill(t *testing.T, baseDir, dirName, content string) {
	t.Helper()
	skillPath := filepath.Join(baseDir, dirName)
	require.NoError(t, os.MkdirAll(skillPath, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(content), 0o644))
}

func TestParseSkillFrontmatter(t *testing.T) {
	name, desc, body := parseSkillFrontmatter([]byte("---\nname: commit\ndescription: Use when committing changes\n---\n\nDo the commit steps.\n"))
	assert.Equal(t, "commit", name)
	assert.Equal(t, "Use when committing changes", desc)
	assert.Equal(t, "Do the commit steps.", body)
}

func TestParseSkillFrontmatterNoFrontmatter(t *testing.T) {
	name, desc, body := parseSkillFrontmatter([]byte("Just a body, no frontmatter.\n"))
	assert.Equal(t, "", name)
	assert.Equal(t, "", desc)
	assert.Equal(t, "Just a body, no frontmatter.", body)
}

func TestDiscoverSkills(t *testing.T) {
	base := t.TempDir()
	writeSkill(t, base, "commit", "---\nname: commit\ndescription: Use when committing changes\n---\n\nbody\n")
	writeSkill(t, base, "deploy", "---\nname: deploy\ndescription: Use when deploying\n---\n\nbody\n")

	skills := discoverSkills(base)
	require.Len(t, skills, 2)

	byName := map[string]skillMeta{}
	for _, s := range skills {
		byName[s.Name] = s
	}
	assert.Equal(t, "Use when committing changes", byName["commit"].Description)
	assert.Equal(t, "Use when deploying", byName["deploy"].Description)
}

func TestDiscoverSkillsFallsBackToDirName(t *testing.T) {
	base := t.TempDir()
	writeSkill(t, base, "mytool", "---\ndescription: No name field here\n---\n\nbody\n")

	skills := discoverSkills(base)
	require.Len(t, skills, 1)
	assert.Equal(t, "mytool", skills[0].Name)
}

func TestDiscoverSkillsMissingDir(t *testing.T) {
	skills := discoverSkills(filepath.Join(t.TempDir(), "does-not-exist"))
	assert.Empty(t, skills)
}

func TestSkillCatalogListsSkills(t *testing.T) {
	base := t.TempDir()
	writeSkill(t, base, "commit", "---\nname: commit\ndescription: Use when committing changes\n---\n\nbody\n")

	catalog := skillCatalog(base)
	assert.Contains(t, catalog, "commit")
	assert.Contains(t, catalog, "Use when committing changes")
}

func TestSkillCatalogEmpty(t *testing.T) {
	catalog := skillCatalog(filepath.Join(t.TempDir(), "none"))
	assert.Contains(t, catalog, "No skills")
}

func TestLoadSkillReturnsBody(t *testing.T) {
	base := t.TempDir()
	writeSkill(t, base, "commit", "---\nname: commit\ndescription: Use when committing changes\n---\n\nStep 1. Step 2.\n")

	body, err := loadSkill(base, "commit")
	require.NoError(t, err)
	assert.Equal(t, "Step 1. Step 2.", body)
}

func TestLoadSkillUnknownNameListsAvailable(t *testing.T) {
	base := t.TempDir()
	writeSkill(t, base, "commit", "---\nname: commit\ndescription: Use when committing changes\n---\n\nbody\n")

	_, err := loadSkill(base, "nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "commit")
}
