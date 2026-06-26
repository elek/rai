package acp

import (
	"bufio"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/elek/rai/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeConfig returns a config whose default model uses the built-in "fake"
// provider, which generates random text without contacting any real API.
func fakeConfig() config.Config {
	return config.Config{
		Providers: []config.Provider{
			{Name: "fake", Type: "fake"},
		},
		Models: []config.Model{
			{Name: "fake", Provider: "fake", Model: "fake-model", Default: true},
		},
	}
}

// acpClient is a minimal ACP client that drives a Server over an io.Pipe,
// mirroring what `acpp cat "rai acp"` does: it speaks newline-delimited
// JSON-RPC 2.0 over stdio.
type acpClient struct {
	t       *testing.T
	in      *io.PipeWriter
	scanner *bufio.Scanner
}

func newACPClient(t *testing.T, srv *Server) *acpClient {
	t.Helper()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()

	go func() {
		_ = srv.ServeIO(inR, outW)
		_ = outW.Close()
	}()

	scanner := bufio.NewScanner(outR)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	return &acpClient{t: t, in: inW, scanner: scanner}
}

func (c *acpClient) send(msg string) {
	c.t.Helper()
	_, err := io.WriteString(c.in, msg+"\n")
	require.NoError(c.t, err)
}

func (c *acpClient) close() { _ = c.in.Close() }

// readUntilResponse reads messages, collecting notifications, until it sees a
// response (a message carrying the given JSON-RPC id). It returns the response
// and every notification seen along the way.
func (c *acpClient) readUntilResponse(id int) (Response, []Notification) {
	c.t.Helper()
	var notifs []Notification
	done := make(chan struct{})
	var resp Response
	var found bool

	go func() {
		defer close(done)
		for c.scanner.Scan() {
			line := c.scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			// Distinguish responses (have "id") from notifications (have "method", no "id").
			var probe struct {
				ID     json.RawMessage `json:"id"`
				Method string          `json:"method"`
			}
			_ = json.Unmarshal(line, &probe)
			if probe.ID == nil && probe.Method != "" {
				var n Notification
				if err := json.Unmarshal(line, &n); err == nil {
					notifs = append(notifs, n)
				}
				continue
			}
			var r Response
			require.NoError(c.t, json.Unmarshal(line, &r))
			if string(r.ID) == strconv.Itoa(id) {
				resp = r
				found = true
				return
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		c.t.Fatalf("timed out waiting for response id=%d", id)
	}
	require.True(c.t, found, "did not receive response id=%d", id)
	return resp, notifs
}

// TestACPFullPromptFlow exercises the same path as `acpp cat "rai acp"`:
// initialize -> session/new -> session/prompt, and verifies that the agent
// streams text back and reports a normal end of turn.
func TestACPFullPromptFlow(t *testing.T) {
	srv := NewServer(nil)
	srv.SetConfig(fakeConfig())

	client := newACPClient(t, srv)
	defer client.close()

	// 1. initialize
	client.send(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientCapabilities":{},"clientInfo":{"name":"acpp-test"}}}`)
	initResp, _ := client.readUntilResponse(1)
	require.Nil(t, initResp.Error)

	// 2. session/new
	client.send(`{"jsonrpc":"2.0","id":2,"method":"session/new","params":{"cwd":"/tmp","mcpServers":[]}}`)
	sessResp, _ := client.readUntilResponse(2)
	require.Nil(t, sessResp.Error)

	resultBytes, err := json.Marshal(sessResp.Result)
	require.NoError(t, err)
	var sessResult NewSessionResult
	require.NoError(t, json.Unmarshal(resultBytes, &sessResult))
	require.NotEmpty(t, sessResult.SessionID)

	// 3. session/prompt
	client.send(`{"jsonrpc":"2.0","id":3,"method":"session/prompt","params":{"sessionId":"` +
		sessResult.SessionID + `","prompt":[{"type":"text","text":"say hello"}]}}`)
	promptResp, notifs := client.readUntilResponse(3)
	require.Nil(t, promptResp.Error)

	// The prompt should have streamed agent_message_chunk notifications with text.
	var streamed strings.Builder
	var chunkCount int
	for _, n := range notifs {
		if n.Method != "session/update" {
			continue
		}
		paramBytes, err := json.Marshal(n.Params)
		require.NoError(t, err)
		var upd SessionUpdateNotification
		require.NoError(t, json.Unmarshal(paramBytes, &upd))
		if upd.Update.SessionUpdate == "agent_message_chunk" && upd.Update.Content != nil {
			chunkCount++
			streamed.WriteString(upd.Update.Content.Text)
			assert.Equal(t, sessResult.SessionID, upd.SessionID)
		}
	}
	assert.Positive(t, chunkCount, "expected streamed agent_message_chunk notifications")
	assert.NotEmpty(t, strings.TrimSpace(streamed.String()), "expected non-empty streamed text")

	// The final response should report a normal end of turn with usage.
	finalBytes, err := json.Marshal(promptResp.Result)
	require.NoError(t, err)
	var promptResult PromptResult
	require.NoError(t, json.Unmarshal(finalBytes, &promptResult))
	assert.Equal(t, "end_turn", promptResult.StopReason)
	require.NotNil(t, promptResult.Usage)
	assert.Positive(t, promptResult.Usage.OutputTokens)
}
