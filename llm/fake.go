package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// fakeModel is a Model that returns randomly generated text instead of calling
// a real provider. It is intended for tests and offline experiments (e.g.
// exercising the ACP server end-to-end without API keys). It never emits tool
// calls.
type fakeModel struct {
	provider string
	model    string
}

// NewFakeModel creates a fake model bound to the given provider and model names.
// The names are only echoed through Provider()/Name(); they have no effect on
// the generated output.
func NewFakeModel(provider, model string) Model {
	return &fakeModel{provider: provider, model: model}
}

func (f *fakeModel) Provider() string { return f.provider }
func (f *fakeModel) Name() string     { return f.model }

// Stream streams randomly generated text word by word via onText, then returns
// the complete turn. As special cases, a "[scenario1 count=N]" directive drives
// a scripted write/read loop (see scenario1Turn) and a prompt mentioning
// "commit" requests git tool calls (see commitTurn). Context cancellation is
// honored between words.
func (f *fakeModel) Stream(ctx context.Context, req Request, onText func(delta string)) (*Turn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if count, ok := scenario1Count(req.Messages); ok {
		return scenario1Turn(ctx, req, count, onText)
	}
	if turn := commitTurn(req); turn != nil {
		return turn, nil
	}

	text := randomText()
	if err := streamText(ctx, text, onText); err != nil {
		return nil, err
	}

	return &Turn{
		Blocks:     []Block{TextBlock(text)},
		Usage:      usageFor(text),
		StopReason: StopEnd,
	}, nil
}

// streamText delivers text to onText word by word, honoring context
// cancellation between words. onText may be nil.
func streamText(ctx context.Context, text string, onText func(delta string)) error {
	for _, word := range strings.SplitAfter(text, " ") {
		if err := ctx.Err(); err != nil {
			return err
		}
		if word == "" {
			continue
		}
		if onText != nil {
			onText(word)
		}
	}
	return nil
}

// commitTurn implements the fake model's only tool-using behavior: when the
// prompt mentions "commit", it asks the agent to stage all changes and commit
// them with a random message. It returns nil to fall through to plain text
// generation when the prompt is unrelated. Once the agent has fed the tool
// results back (the last message is a tool turn), it returns a plain
// completion so the agent loop terminates instead of committing forever.
func commitTurn(req Request) *Turn {
	if n := len(req.Messages); n > 0 && req.Messages[n-1].Role == RoleTool {
		text := "Done: staged and committed the changes."
		return &Turn{Blocks: []Block{TextBlock(text)}, Usage: usageFor(text), StopReason: StopEnd}
	}
	if !mentionsCommit(req.Messages) {
		return nil
	}
	message := randomCommitMessage()
	return &Turn{
		Blocks: []Block{
			gitToolUse("fake-tool-1", "git add -A"),
			gitToolUse("fake-tool-2", `git commit -m "`+message+`"`),
		},
		Usage:      usageFor(message),
		StopReason: StopToolUse,
	}
}

// mentionsCommit reports whether any user message text contains "commit".
func mentionsCommit(messages []Message) bool {
	for _, m := range messages {
		if m.Role != RoleUser {
			continue
		}
		for _, b := range m.Blocks {
			if b.Type == BlockText && strings.Contains(strings.ToLower(b.Text), "commit") {
				return true
			}
		}
	}
	return false
}

// scenario1Re matches the scenario directive in a prompt, e.g.
// "[scenario1 count=3]". The count group is optional.
var scenario1Re = regexp.MustCompile(`\[scenario1(?:\s+count=(\d+))?\]`)

// scenario1Count reports whether any user message requested scenario1 and, if
// so, how many write/read iterations it should run. A directive without an
// explicit count defaults to 1. It returns ok=false when no directive is found.
func scenario1Count(messages []Message) (count int, ok bool) {
	for _, m := range messages {
		if m.Role != RoleUser {
			continue
		}
		for _, b := range m.Blocks {
			if b.Type != BlockText {
				continue
			}
			match := scenario1Re.FindStringSubmatch(b.Text)
			if match == nil {
				continue
			}
			count = 1
			if match[1] != "" {
				if n, err := strconv.Atoi(match[1]); err == nil && n > 0 {
					count = n
				}
			}
			return count, true
		}
	}
	return 0, false
}

// scenario1Turn drives the "[scenario1 count=N]" behavior. On each model turn it
// streams a short text message and requests two tool calls — a "create" to write
// a file and a "cat" to read it back — then pauses 500ms to simulate work. The
// fake model is stateless, so it infers the current iteration from the number of
// tool-result turns already in the history. After N iterations it streams a
// plain completion (StopEnd) so the agent loop terminates.
func scenario1Turn(ctx context.Context, req Request, count int, onText func(delta string)) (*Turn, error) {
	iteration := countToolTurns(req.Messages)
	if iteration >= count {
		text := fmt.Sprintf("Done: completed %d write/read iteration(s).", count)
		if err := streamText(ctx, text, onText); err != nil {
			return nil, err
		}
		return &Turn{Blocks: []Block{TextBlock(text)}, Usage: usageFor(text), StopReason: StopEnd}, nil
	}

	text := fmt.Sprintf("Iteration %d of %d: writing then reading data.", iteration+1, count)
	if err := streamText(ctx, text, onText); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("%s/rai-scenario1-%d.txt", os.TempDir(), iteration)
	content := fmt.Sprintf("scenario1 iteration %d\n", iteration)
	createInput, _ := json.Marshal(map[string]string{"path": path, "file_text": content})
	catInput, _ := json.Marshal(map[string]string{"path": path})

	blocks := []Block{
		TextBlock(text),
		{Type: BlockToolUse, ToolCallID: fmt.Sprintf("scenario1-create-%d", iteration), ToolName: "create", Input: string(createInput)},
		{Type: BlockToolUse, ToolCallID: fmt.Sprintf("scenario1-cat-%d", iteration), ToolName: "cat", Input: string(catInput)},
	}

	// Pause between iterations to simulate work; honor cancellation.
	if err := sleep(ctx, 500*time.Millisecond); err != nil {
		return nil, err
	}

	return &Turn{Blocks: blocks, Usage: usageFor(text), StopReason: StopToolUse}, nil
}

// countToolTurns returns the number of tool-result turns in messages, which the
// stateless fake model uses to track how many scenario iterations have completed.
func countToolTurns(messages []Message) int {
	n := 0
	for _, m := range messages {
		if m.Role == RoleTool {
			n++
		}
	}
	return n
}

// sleep pauses for d, returning early with ctx.Err() if ctx is cancelled first.
func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// gitToolUse builds a tool_use block invoking the "git" tool with the given
// shell command.
func gitToolUse(id, command string) Block {
	input, _ := json.Marshal(map[string]string{"command": command})
	return Block{Type: BlockToolUse, ToolCallID: id, ToolName: "git", Input: string(input)}
}

// randomCommitMessage returns a short commit message of 2-5 random words.
func randomCommitMessage() string {
	n := 2 + rand.IntN(4)
	words := make([]string, n)
	for i := range words {
		words[i] = randomWord()
	}
	return strings.Join(words, " ")
}

// usageFor builds a plausible Usage value for the given generated text.
func usageFor(text string) Usage {
	out := int64(len(strings.Fields(text)))
	in := int64(8)
	return Usage{InputTokens: in, OutputTokens: out, TotalTokens: in + out}
}

// randomWord returns a random lowercase word between 3 and 10 characters long.
func randomWord() string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	n := 3 + rand.IntN(8)
	var b strings.Builder
	b.Grow(n)
	for i := 0; i < n; i++ {
		b.WriteByte(letters[rand.IntN(len(letters))])
	}
	return b.String()
}

// randomParagraph returns a paragraph of random words.
func randomParagraph() string {
	n := 15 + rand.IntN(36)
	words := make([]string, n)
	for i := range words {
		words[i] = randomWord()
	}
	return strings.Join(words, " ")
}

// randomText returns 1-3 paragraphs of random words separated by blank lines.
func randomText() string {
	n := 1 + rand.IntN(3)
	paragraphs := make([]string, n)
	for i := range paragraphs {
		paragraphs[i] = randomParagraph()
	}
	return strings.Join(paragraphs, "\n\n")
}
