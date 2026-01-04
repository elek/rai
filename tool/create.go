package tool

import (
	"os"

	"github.com/pkg/errors"
)

type CreateInput struct {
	Path     string `json:"path" description:"The path of the file to be written"`
	FileText string `json:"file_text" description:"The file content to write to the specified path"`
}

func Create(input CreateInput) (string, error) {
	if input.Path == "" {
		return "", errors.New("path is required")
	}
	err := os.WriteFile(input.Path, []byte(input.FileText), 0644)
	if err != nil {
		return "", errors.Wrap(err, "error writing file")
	}
	return "File created successfully at " + input.Path, nil
}
