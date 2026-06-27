package tool

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/elek/rai/llm"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
)

func NewMcpAgentTool(ctx context.Context, command string, args []string) ([]llm.Tool, func(), error) {
	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)

	transport := &mcp.CommandTransport{Command: exec.Command(command, args...)}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, func() {}, errors.WithStack(err)
	}

	var agentTools []llm.Tool
	for tool, err := range session.Tools(ctx, &mcp.ListToolsParams{}) {
		if err != nil {
			return nil, func() {}, errors.WithStack(err)
		}

		agentTools = append(agentTools, NewMcpAgentToolMethod(session, tool))
	}
	return agentTools, func() {
		session.Close()
	}, nil

}

type McpAgentTool struct {
	session *mcp.ClientSession
	info    llm.ToolInfo
}

func NewMcpAgentToolMethod(session *mcp.ClientSession, tool *mcp.Tool) *McpAgentTool {
	// Extract parameters and required fields from InputSchema
	var parameters map[string]any
	var required []string

	if tool.InputSchema != nil {
		if schema, ok := tool.InputSchema.(map[string]any); ok {
			if props, ok := schema["properties"].(map[string]any); ok {
				parameters = props
			}
			if req, ok := schema["required"].([]any); ok {
				for _, r := range req {
					if reqStr, ok := r.(string); ok {
						required = append(required, reqStr)
					}
				}
			} else if req, ok := schema["required"].([]string); ok {
				required = req
			}
		}
	}

	return &McpAgentTool{
		session: session,
		info: llm.ToolInfo{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  parameters,
			Required:    required,
		},
	}
}

func (m McpAgentTool) Info() llm.ToolInfo {
	return m.info
}

func (m McpAgentTool) Run(ctx context.Context, params llm.ToolCall) (llm.ToolResult, error) {
	// Parse the input JSON string into arguments
	var arguments map[string]any
	if params.Input != "" {
		if err := json.Unmarshal([]byte(params.Input), &arguments); err != nil {
			return llm.ToolResult{Content: "Failed to parse tool input: " + err.Error(), IsError: true}, nil
		}
	}

	// Call the MCP tool through the session
	result, err := m.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      m.info.Name,
		Arguments: arguments,
	})
	if err != nil {
		return llm.ToolResult{Content: "Failed to call MCP tool: " + err.Error(), IsError: true}, nil
	}

	// Convert the MCP result to an llm.ToolResult
	var contentBuilder strings.Builder
	for i, content := range result.Content {
		if i > 0 {
			contentBuilder.WriteString("\n")
		}

		// Handle different content types
		switch c := content.(type) {
		case *mcp.TextContent:
			contentBuilder.WriteString(c.Text)
		default:
			// For other content types, marshal to JSON
			jsonData, err := json.Marshal(content)
			if err != nil {
				contentBuilder.WriteString("[Unable to serialize content]")
			} else {
				contentBuilder.Write(jsonData)
			}
		}
	}

	// Append any structured content so it is not lost.
	if result.StructuredContent != nil {
		if metadataJSON, err := json.Marshal(result.StructuredContent); err == nil {
			if contentBuilder.Len() > 0 {
				contentBuilder.WriteString("\n")
			}
			contentBuilder.Write(metadataJSON)
		}
	}

	return llm.ToolResult{
		Content: contentBuilder.String(),
		IsError: result.IsError,
	}, nil
}

var _ llm.Tool = (*McpAgentTool)(nil)
