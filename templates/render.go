package templates

import (
	"strings"

	"github.com/pkg/errors"
)

func RenderPrompt(tmpl string, args []string) (Render, error) {
	templateEngine := "gotemplate" // default engine

	// Check if the first line starts with %name
	lines := strings.Split(tmpl, "\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "%") {
		engineName := strings.TrimSpace(strings.TrimPrefix(lines[0], "%"))
		if engineName == "pongo2" || engineName == "gotemplate" {
			templateEngine = engineName

			tmpl = strings.Join(lines[1:], "\n")
		}
	}

	var err error
	var rendered Render
	switch templateEngine {
	case "pongo2":
		rendered, err = RenderPongo2(tmpl, map[string]any{
			"Args": args,
		})

	default:
		return Render{}, errors.Errorf("unknown template engine: %s", templateEngine)
	}
	if err != nil {
		return Render{}, errors.WithStack(err)
	}
	return rendered, nil
}
