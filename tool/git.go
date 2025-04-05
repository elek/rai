package tool

import (
	"os/exec"
	"strings"
)

type GitInput struct {
	Command string `json:"command" description:"The git command to execute including git itself as. For example, "git status" or "git commit -m 'fix bug'"`
}

func Git(input GitInput) string {
	if !strings.HasPrefix(input.Command, "git ") {
		input.Command = "git " + input.Command
	}
	cmd := exec.Command("/bin/sh", "-c", input.Command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "Error: " + err.Error() + "\n" + string(out)
	}
	return string(out)
}
