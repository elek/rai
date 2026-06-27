package llm

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// debugOut is the destination for debug traces. It is a package-level variable
// so tests can capture the output.
var debugOut io.Writer = os.Stderr

// debugRequest writes a human-readable trace of an outgoing request to the LLM.
func debugRequest(provider, model string, req Request) {
	_, _ = io.WriteString(debugOut, formatRequest(provider, model, req))
}

// debugTurn writes a human-readable trace of a turn received from the LLM.
func debugTurn(provider, model string, turn *Turn) {
	_, _ = io.WriteString(debugOut, formatTurn(provider, model, turn))
}

// formatRequest renders a Request as a human-readable, multi-line trace.
func formatRequest(provider, model string, req Request) string {
	var b strings.Builder
	fmt.Fprintf(&b, ">>> %s/%s request (max_tokens=%d", provider, model, req.MaxTokens)
	if req.Temperature > 0 {
		fmt.Fprintf(&b, ", temperature=%.2f", req.Temperature)
	}
	b.WriteString(")\n")
	if req.System != "" {
		fmt.Fprintf(&b, "  [system]\n%s\n", indent(req.System))
	}
	for _, msg := range req.Messages {
		writeBlocks(&b, string(msg.Role), msg.Blocks)
	}
	if len(req.Tools) > 0 {
		fmt.Fprintf(&b, "  [tools] %d offered\n", len(req.Tools))
		for _, t := range req.Tools {
			info := t.Info()
			fmt.Fprintf(&b, "    - %s: %s\n", info.Name, oneLine(info.Description))
			if len(info.Parameters) > 0 {
				schema, _ := json.Marshal(map[string]any{
					"properties": info.Parameters,
					"required":   info.Required,
				})
				fmt.Fprintf(&b, "        params: %s\n", schema)
			}
		}
	}
	return b.String()
}

// formatTurn renders a Turn as a human-readable, multi-line trace.
func formatTurn(provider, model string, turn *Turn) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<<< %s/%s response (stop=%s, tokens in=%d out=%d)\n",
		provider, model, turn.StopReason, turn.Usage.InputTokens, turn.Usage.OutputTokens)
	writeBlocks(&b, string(RoleAssistant), turn.Blocks)
	return b.String()
}

// writeBlocks renders a role and its content blocks.
func writeBlocks(b *strings.Builder, role string, blocks []Block) {
	fmt.Fprintf(b, "  [%s]\n", role)
	for _, blk := range blocks {
		switch blk.Type {
		case BlockText:
			fmt.Fprintf(b, "    text:\n%s\n", indent(blk.Text))
		case BlockToolUse:
			fmt.Fprintf(b, "    tool_use %s (id=%s): %s\n", blk.ToolName, blk.ToolCallID, blk.Input)
		case BlockToolResult:
			marker := "tool_result"
			if blk.IsError {
				marker = "tool_result(error)"
			}
			fmt.Fprintf(b, "    %s (id=%s):\n%s\n", marker, blk.ToolCallID, indent(blk.Text))
		}
	}
}

// oneLine collapses a multi-line string into a single line so it fits on one
// trace row; long values are truncated.
func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}

// indent prefixes every line of s with four spaces.
func indent(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = "    " + line
	}
	return strings.Join(lines, "\n")
}
