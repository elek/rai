package tool

import (
	"os"
	"strings"

	"github.com/pkg/errors"
)

type InsertInput struct {
	Path       string `json:"path" description:"The path of the file where the content should be written to."`
	InsertLine int    `json:"insert_line" description:"The line number at which to insert the new content (0-based index)."`
	NewStr     string `json:"new_str" description:"The file content to write to the specified path"`
}

func Insert(input InsertInput) (string, error) {
	if input.Path == "" {
		return "", errors.New("path is required")
	}
	stat, err := os.Stat(input.Path)
	if err != nil {
		return "", errors.Wrap(err, "error stating file")
	}
	current, err := os.ReadFile(input.Path)
	if err != nil {
		return "", errors.Wrap(err, "error reading file")
	}
	out := ""
	for ix, line := range strings.Split(string(current), "\n") {
		out += line + "\n"
		if ix == input.InsertLine {
			out += input.NewStr + "\n"
		}
	}
	err = os.WriteFile(input.Path, []byte(out), stat.Mode())
	if err != nil {
		return "", errors.Wrap(err, "error writing file")
	}
	return "File updated successfully at " + input.Path, nil
}
