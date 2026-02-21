package tool

import (
	"os/exec"
)

type BashInput struct {
	Command string `json:"command" description:"The bash command to execute"`
}

func Bash(input BashInput) string {
	cmd := exec.Command("/bin/sh", "-c", input.Command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "Error: " + err.Error() + "\n" + string(out)
	}
	return string(out)
}
