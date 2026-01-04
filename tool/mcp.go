package tool

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"

	"charm.land/fantasy"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
)

func NewMcpAgentTool(ctx context.Context, command string, args []string) ([]fantasy.AgentTool, func(), error) {
	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)

	transport := &mcp.CommandTransport{Command: exec.Command(command, args...)}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, func() {}, errors.WithStack(err)
	}

	var agentTools []fantasy.AgentTool
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
	info    fantasy.ToolInfo
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
		info: fantasy.ToolInfo{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  parameters,
			Required:    required,
		},
	}
}

func (m McpAgentTool) Info() fantasy.ToolInfo {
	return m.info
}

func (m McpAgentTool) Run(ctx context.Context, params fantasy.ToolCall) (fantasy.ToolResponse, error) {
	// Parse the input JSON string into arguments
	var arguments map[string]any
	if params.Input != "" {
		if err := json.Unmarshal([]byte(params.Input), &arguments); err != nil {
			return fantasy.NewTextErrorResponse("Failed to parse tool input: " + err.Error()), errors.WithStack(err)
		}
	}

	// Call the MCP tool through the session
	result, err := m.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      m.info.Name,
		Arguments: arguments,
	})
	if err != nil {
		return fantasy.NewTextErrorResponse("Failed to call MCP tool: " + err.Error()), errors.WithStack(err)
	}

	// Convert the MCP result to fantasy.ToolResponse
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

	response := fantasy.ToolResponse{
		Type:    "text",
		Content: contentBuilder.String(),
		IsError: result.IsError,
	}

	// Add structured content as metadata if present
	if result.StructuredContent != nil {
		metadataJSON, err := json.Marshal(result.StructuredContent)
		if err == nil {
			response.Metadata = string(metadataJSON)
		}
	}

	return response, nil
}

func (m McpAgentTool) ProviderOptions() fantasy.ProviderOptions {
	return fantasy.ProviderOptions{}
}

func (m McpAgentTool) SetProviderOptions(opts fantasy.ProviderOptions) {
	// noop
}

var _ fantasy.AgentTool = (*McpAgentTool)(nil)
