package templates

import (
	"bytes"
	"html/template"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

func GoTemplateRender(tmplStr string, data map[string]any) (Render, error) {
	templateDef := template.New("prompt").Funcs(map[string]any{
		"bash":   bashFunc,
		"exec":   execFunc,
		"run":    RunFunc,
		"tmpDir": os.TempDir,
	})
	tpl, err := templateDef.Parse(tmplStr)
	if err != nil {
		return Render{}, errors.WithStack(err)
	}
	out := bytes.NewBuffer([]byte{})
	err = tpl.Execute(out, data)
	if err != nil {
		return Render{}, errors.WithStack(err)
	}
	render := Render{
		Prompt: out.String(),
	}

	system := tpl.Lookup("system")
	if system != nil {
		out := bytes.NewBuffer([]byte{})
		err := system.Execute(out, map[string]string{})
		if err != nil {
			return Render{}, errors.WithStack(err)
		}
		render.System = strings.TrimSpace(out.String())

	}
	return render, nil
}

func RunFunc(cmd string) (string, error) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "", nil
	}
	c := exec.Command(parts[0], parts[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	return "", err
}

func execFunc(cmd string) (string, error) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "", nil
	}
	out, err := exec.Command(parts[0], parts[1:]...).CombinedOutput()
	return string(out), err
}

func bashFunc(cmd string) (string, error) {
	out, err := exec.Command("bash", "-c", cmd).CombinedOutput()
	return string(out), err
}
