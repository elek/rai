package acp

import "encoding/json"

// JSON-RPC 2.0 base types

// Request represents a JSON-RPC 2.0 request message.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response message.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// Notification represents a JSON-RPC 2.0 notification (no ID, no response expected).
type Notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Initialize

// InitializeParams contains the parameters for the initialize request.
type InitializeParams struct {
	ProtocolVersion    int                `json:"protocolVersion"`
	ClientCapabilities ClientCapabilities `json:"clientCapabilities"`
	ClientInfo         ImplementationInfo `json:"clientInfo"`
}

// ClientCapabilities describes what the client supports.
type ClientCapabilities struct {
	Fs       *FsCapabilities `json:"fs,omitempty"`
	Terminal bool            `json:"terminal,omitempty"`
}

// FsCapabilities describes the client's filesystem capabilities.
type FsCapabilities struct {
	ReadTextFile  bool `json:"readTextFile,omitempty"`
	WriteTextFile bool `json:"writeTextFile,omitempty"`
}

// ImplementationInfo provides metadata about a client or agent implementation.
type ImplementationInfo struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version,omitempty"`
}

// InitializeResult contains the result of the initialize request.
type InitializeResult struct {
	ProtocolVersion   int                `json:"protocolVersion"`
	AgentCapabilities AgentCapabilities  `json:"agentCapabilities"`
	AgentInfo         ImplementationInfo `json:"agentInfo"`
}

// AgentCapabilities describes what the agent supports.
type AgentCapabilities struct {
	PromptCapabilities *PromptCapabilities `json:"promptCapabilities,omitempty"`
}

// PromptCapabilities describes the agent's prompt handling capabilities.
type PromptCapabilities struct {
	Text bool `json:"text,omitempty"`
}

// Session

// NewSessionParams contains the parameters for the session/new request.
type NewSessionParams struct {
	Cwd        string `json:"cwd"`
	McpServers []any  `json:"mcpServers,omitempty"`
}

// NewSessionResult contains the result of the session/new request.
type NewSessionResult struct {
	SessionID string `json:"sessionId"`
}

// Prompt

// PromptParams contains the parameters for the session/prompt request.
type PromptParams struct {
	SessionID string         `json:"sessionId"`
	Prompt    []ContentBlock `json:"prompt"`
}

// ContentBlock represents a block of content within a prompt or response.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// PromptResult contains the result of the session/prompt request.
type PromptResult struct {
	StopReason string     `json:"stopReason"`
	Usage      *UsageInfo `json:"usage,omitempty"`
	Meta       *RaiMeta   `json:"rai,omitempty"`
}

// UsageInfo contains token usage statistics for a prompt.
type UsageInfo struct {
	InputTokens         int64 `json:"inputTokens"`
	OutputTokens        int64 `json:"outputTokens"`
	TotalTokens         int64 `json:"totalTokens"`
	CacheCreationTokens int64 `json:"cacheCreationTokens,omitempty"`
	CacheReadTokens     int64 `json:"cacheReadTokens,omitempty"`
}

// RaiMeta contains rai-specific metadata including cost and model usage details.
type RaiMeta struct {
	TotalCostUSD float64                    `json:"totalCostUsd"`
	Model        string                     `json:"model"`
	ModelUsage   map[string]*ModelUsageInfo `json:"modelUsage"`
}

// ModelUsageInfo contains detailed usage statistics for a specific model.
type ModelUsageInfo struct {
	InputTokens              int64   `json:"inputTokens"`
	OutputTokens             int64   `json:"outputTokens"`
	CacheCreationInputTokens int64   `json:"cacheCreationInputTokens"`
	CacheReadInputTokens     int64   `json:"cacheReadInputTokens"`
	ContextWindow            int64   `json:"contextWindow"`
	MaxOutputTokens          int64   `json:"maxOutputTokens"`
	WebSearchRequests        int64   `json:"webSearchRequests"`
	CostUSD                  float64 `json:"costUSD"`
}

// Session Update notifications

// SessionUpdateNotification contains the top-level params for session/update notifications.
type SessionUpdateNotification struct {
	SessionID string              `json:"sessionId"`
	Update    SessionUpdateParams `json:"update"`
}

// SessionUpdateParams contains the update data within a session/update notification.
type SessionUpdateParams struct {
	SessionUpdate     string             `json:"sessionUpdate"`
	Content           *ContentBlock      `json:"content,omitempty"`
	ToolCall          *ToolCall          `json:"toolCall,omitempty"`
	ToolCallID        string             `json:"toolCallId,omitempty"`
	Status            string             `json:"status,omitempty"`
	AvailableCommands []AvailableCommand `json:"availableCommands,omitempty"`
}

// ToolCall represents a tool invocation within a session update notification.
type ToolCall struct {
	ToolCallID string `json:"toolCallId"`
	Title      string `json:"title"`
	Kind       string `json:"kind,omitempty"`
	Status     string `json:"status"`
}

// AvailableCommand describes a command the agent can execute.
type AvailableCommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AvailableCommandsUpdate contains the update data for an available_commands_update notification.
type AvailableCommandsUpdate struct {
	AvailableCommands []AvailableCommand `json:"availableCommands"`
}

// Cancel

// CancelParams contains the parameters for the session/cancel notification.
type CancelParams struct {
	SessionID string `json:"sessionId"`
}
