package llm

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/elek/rai/config"
	"github.com/pkg/errors"
)

// noTextNotice is shown when an agent run completes without producing any final
// text (for example when a model degenerates and only emits tool calls), so the
// absence of an answer is not mistaken for a silent failure.
const noTextNotice = "(no text returned by the model)"

// AgentCallback runs a prompt against a model with the given system prompt and
// tools, returning the model's final text response. ExecPrompt and DryRun both
// satisfy this type.
type AgentCallback func(ctx context.Context, model config.Model, system string, prompt string, tools []Tool) (string, error)

// Executor runs prompts against models created from a configuration.
type Executor struct {
	cfg   config.Config
	debug bool
	// out is where streamed text, tool-call notices, and the empty-response
	// notice are written. It defaults to os.Stdout; tests inject a buffer.
	out io.Writer
}

// NewExecutor creates an Executor bound to a configuration. When debug is true,
// every request and response is traced to stderr regardless of per-model config.
func NewExecutor(cfg config.Config, debug bool) *Executor {
	return &Executor{cfg: cfg, debug: debug, out: os.Stdout}
}

// ExecPrompt runs prompt through an agent loop, streaming the model's text to
// stdout and reporting tool calls as they happen. If mdl is the zero value, the
// configured default model is used.
func (e *Executor) ExecPrompt(ctx context.Context, mdl config.Model, system string, prompt string, tools []Tool) (string, error) {
	if mdl == (config.Model{}) {
		def, found := e.cfg.FindDefaultModel()
		if !found {
			return "", errors.New("no default model configured")
		}
		mdl = def
	}
	// The --debug flag forces tracing on regardless of the model config.
	if e.debug {
		mdl.Debug = true
	}

	model, err := NewModel(ctx, e.cfg, mdl)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return e.runAgent(ctx, model, system, prompt, tools)
}

// runAgent drives the agent loop against an already-created model, streaming
// text and tool calls to the executor's output. When the run finishes without
// any final text, it writes noTextNotice so a degenerate response is not
// mistaken for no output at all.
func (e *Executor) runAgent(ctx context.Context, model Model, system string, prompt string, tools []Tool) (string, error) {
	out := e.out
	if out == nil {
		out = os.Stdout
	}

	agent := NewAgent(model, system, tools)
	result, err := agent.Run(ctx, prompt, RunOptions{
		OnTextDelta: func(delta string) { fmt.Fprint(out, delta) },
		OnToolCall: func(name, input string) {
			fmt.Fprintln(out, "Calling tool", name, "with input:", input)
		},
	})
	if err != nil {
		return "", errors.WithStack(err)
	}
	fmt.Fprintln(out)
	if strings.TrimSpace(result.Text) == "" {
		fmt.Fprintln(out, noTextNotice)
	}
	return result.Text, nil
}

// DryRun prints the resolved model, prompts, and tools without calling any
// provider. It satisfies AgentCallback.
func DryRun(ctx context.Context, model config.Model, system string, prompt string, tools []Tool) (string, error) {
	fmt.Println("Using model:", model.Provider, model.Model)
	if system != "" {
		fmt.Println("--- SYSTEM PROMPT ---")
		fmt.Println(system)
	}
	fmt.Println("--- PROMPT ----------")
	fmt.Println(prompt)
	fmt.Println("--- TOOLS -----------")
	fmt.Print(describeTools(tools))
	fmt.Println("---------------------")
	return "", nil
}
