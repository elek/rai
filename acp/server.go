package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/elek/catwalk-open/providers"
	"github.com/elek/rai/config"
	"github.com/elek/rai/llm"
	"github.com/elek/rai/templates"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// Session holds the state for a single ACP session.
type Session struct {
	ID             string
	Cwd            string
	Model          config.Model
	System         string
	Tools          []llm.Tool
	TemplatePrompt string
	FirstPrompt    bool
	Cancel         context.CancelFunc
}

// Server implements the ACP JSON-RPC 2.0 stdio server.
type Server struct {
	cfg          *config.Config
	parsed       *templates.ParsedTemplate
	defaultModel *config.Model
	sessions     map[string]*Session
	mu           sync.Mutex
	out          io.Writer
	outMu        sync.Mutex
}

// NewServer creates a new ACP server with the given parsed template.
func NewServer(parsed *templates.ParsedTemplate) *Server {
	return &Server{
		parsed:   parsed,
		sessions: make(map[string]*Session),
	}
}

// SetConfig sets the configuration for the server.
func (s *Server) SetConfig(cfg config.Config) {
	s.cfg = &cfg
}

// SetDefaultModel sets a default model override for sessions without a template model.
func (s *Server) SetDefaultModel(m config.Model) {
	s.defaultModel = &m
}

// Serve reads JSON-RPC messages from os.Stdin and writes responses to os.Stdout.
func (s *Server) Serve() error {
	return s.ServeIO(os.Stdin, os.Stdout)
}

// ServeIO reads JSON-RPC messages from the given reader and writes responses to the given writer.
// Messages are newline-delimited JSON.
func (s *Server) ServeIO(in io.Reader, out io.Writer) error {
	s.out = out
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(nil, -32700, "Parse error", err)
			continue
		}

		if req.ID == nil {
			s.handleNotification(req)
			continue
		}

		result, rpcErr := s.handleRequest(req)
		if rpcErr != nil {
			s.sendResponse(Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   rpcErr,
			})
		} else {
			s.sendResponse(Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  result,
			})
		}
	}
	return scanner.Err()
}

func (s *Server) handleRequest(req Request) (any, *RPCError) {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "session/new":
		return s.handleNewSession(req)
	case "session/prompt":
		return s.handlePrompt(req)
	default:
		return nil, &RPCError{Code: -32601, Message: "Method not found: " + req.Method}
	}
}

func (s *Server) handleNotification(req Request) {
	switch req.Method {
	case "session/cancel":
		var params CancelParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return
		}
		s.mu.Lock()
		sess, ok := s.sessions[params.SessionID]
		s.mu.Unlock()
		if ok && sess.Cancel != nil {
			sess.Cancel()
		}
	}
}

func (s *Server) handleInitialize(_ Request) (any, *RPCError) {
	return InitializeResult{
		ProtocolVersion: 1,
		AgentCapabilities: AgentCapabilities{
			PromptCapabilities: &PromptCapabilities{
				Text: true,
			},
		},
		AgentInfo: ImplementationInfo{
			Name:    "rai",
			Title:   "RAI Agent",
			Version: "0.1.0",
		},
	}, nil
}

func (s *Server) handleNewSession(req Request) (any, *RPCError) {
	var params NewSessionParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &RPCError{Code: -32602, Message: "Invalid params: " + err.Error()}
	}

	id := uuid.New().String()
	sess := &Session{
		ID:          id,
		Cwd:         params.Cwd,
		FirstPrompt: true,
	}
	if s.parsed != nil {
		sess.Model = s.parsed.Model
		sess.System = s.parsed.System
		sess.Tools = s.parsed.Tools
		sess.TemplatePrompt = s.parsed.Prompt
	}

	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()

	if len(sess.Tools) > 0 {
		var cmds []AvailableCommand
		for _, t := range sess.Tools {
			info := t.Info()
			cmds = append(cmds, AvailableCommand{
				Name:        info.Name,
				Description: info.Description,
			})
		}
		s.sendNotification(Notification{
			JSONRPC: "2.0",
			Method:  "session/update",
			Params: SessionUpdateNotification{
				SessionID: id,
				Update: SessionUpdateParams{
					SessionUpdate:     "available_commands_update",
					AvailableCommands: cmds,
				},
			},
		})
	}

	return NewSessionResult{SessionID: id}, nil
}

func (s *Server) handlePrompt(req Request) (any, *RPCError) {
	var params PromptParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &RPCError{Code: -32602, Message: "Invalid params: " + err.Error()}
	}

	s.mu.Lock()
	sess, ok := s.sessions[params.SessionID]
	s.mu.Unlock()
	if !ok {
		return nil, &RPCError{Code: -32002, Message: "Session not found"}
	}

	var promptText string
	for _, block := range params.Prompt {
		if block.Type == "text" {
			promptText += block.Text
		}
	}

	if sess.FirstPrompt && sess.TemplatePrompt != "" {
		promptText = sess.TemplatePrompt + "\n" + promptText
		sess.FirstPrompt = false
	}

	ctx, cancel := context.WithCancel(context.Background())
	sess.Cancel = cancel
	defer cancel()

	if s.cfg == nil {
		return nil, &RPCError{Code: -32603, Message: "Server not configured"}
	}

	model := sess.Model
	if model == (config.Model{}) {
		if s.defaultModel != nil {
			model = *s.defaultModel
		} else {
			var found bool
			model, found = s.cfg.FindDefaultModel()
			if !found {
				return nil, &RPCError{Code: -32603, Message: "No default model configured"}
			}
		}
	}

	lm, err := llm.NewModel(ctx, *s.cfg, model)
	if err != nil {
		return nil, &RPCError{Code: -32603, Message: "Failed to create model: " + err.Error()}
	}

	agent := llm.NewAgent(lm, sess.System, sess.Tools)

	result, err := agent.Run(ctx, promptText, llm.RunOptions{
		OnTextDelta: func(token string) {
			s.sendNotification(Notification{
				JSONRPC: "2.0",
				Method:  "session/update",
				Params: SessionUpdateNotification{
					SessionID: params.SessionID,
					Update: SessionUpdateParams{
						SessionUpdate: "agent_message_chunk",
						Content: &ContentBlock{
							Type: "text",
							Text: token,
						},
					},
				},
			})
		},
		OnToolCall: func(name, input string) {
			tcID := uuid.New().String()
			s.sendNotification(Notification{
				JSONRPC: "2.0",
				Method:  "session/update",
				Params: SessionUpdateNotification{
					SessionID: params.SessionID,
					Update: SessionUpdateParams{
						SessionUpdate: "tool_call",
						ToolCall: &ToolCall{
							ToolCallID: tcID,
							Title:      name,
							Kind:       toolKind(name),
							Status:     "in_progress",
						},
					},
				},
			})
		},
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return PromptResult{StopReason: "cancelled"}, nil
		}
		return nil, &RPCError{Code: -32603, Message: "Agent error: " + err.Error()}
	}

	usage := result.Usage

	modelID := lm.Name()
	meta := &RaiMeta{
		Model: modelID,
		ModelUsage: map[string]*ModelUsageInfo{
			modelID: {
				InputTokens:  usage.InputTokens,
				OutputTokens: usage.OutputTokens,
			},
		},
	}

	for _, provider := range providers.GetAll() {
		if string(provider.ID) != lm.Provider() {
			continue
		}
		for _, m := range provider.Models {
			if m.ID == modelID {
				mu := meta.ModelUsage[modelID]
				mu.ContextWindow = m.ContextWindow
				mu.MaxOutputTokens = m.DefaultMaxTokens
				mu.CostUSD = m.CostPer1MIn*float64(usage.InputTokens)/1_000_000 +
					m.CostPer1MOut*float64(usage.OutputTokens)/1_000_000
				meta.TotalCostUSD = mu.CostUSD
				break
			}
		}
		break
	}

	return PromptResult{
		StopReason: "end_turn",
		Usage: &UsageInfo{
			InputTokens:  usage.InputTokens,
			OutputTokens: usage.OutputTokens,
			TotalTokens:  usage.TotalTokens,
		},
		Meta: meta,
	}, nil
}

func (s *Server) sendResponse(resp Response) {
	s.outMu.Lock()
	defer s.outMu.Unlock()
	data, _ := json.Marshal(resp)
	fmt.Fprintf(s.out, "%s\n", data)
}

func (s *Server) sendNotification(notif Notification) {
	s.outMu.Lock()
	defer s.outMu.Unlock()
	data, _ := json.Marshal(notif)
	fmt.Fprintf(s.out, "%s\n", data)
}

func (s *Server) sendError(id json.RawMessage, code int, message string, err error) {
	s.sendResponse(Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    err.Error(),
		},
	})
}

func toolKind(name string) string {
	switch name {
	case "cat", "files":
		return "read"
	case "create", "insert":
		return "edit"
	case "git":
		return "execute"
	case "bash":
		return "execute"
	default:
		return "other"
	}
}
