package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

// acpClient is a minimal ACP client that speaks newline-delimited JSON-RPC 2.0
// over stdio, mirroring what `acpp cat "rai acp"` does. It can drive either an
// in-process Server or a real `rai acp` subprocess.
type acpClient struct {
	t       *testing.T
	in      io.WriteCloser
	scanner *bufio.Scanner
}

// newACPClientIO builds a client over an arbitrary writer/reader pair, e.g. a
// subprocess's stdin and stdout.
func newACPClientIO(t *testing.T, in io.WriteCloser, out io.Reader) *acpClient {
	t.Helper()
	scanner := bufio.NewScanner(out)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	return &acpClient{t: t, in: in, scanner: scanner}
}

func newACPClient(t *testing.T, srv *Server) *acpClient {
	t.Helper()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()

	go func() {
		_ = srv.ServeIO(inR, outW)
		_ = outW.Close()
	}()

	return newACPClientIO(t, inW, outR)
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

// drivePromptFlow runs the full ACP conversation against the given client:
// initialize -> session/new -> session/prompt, and asserts that the agent
// streams text back and reports a normal end of turn.
func drivePromptFlow(t *testing.T, client *acpClient) {
	t.Helper()

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

// TestACPFullPromptFlow exercises the same path as `acpp cat "rai acp"`,
// driving the ACP server in-process.
func TestACPFullPromptFlow(t *testing.T) {
	srv := NewServer(nil)
	srv.SetConfig(fakeConfig())

	client := newACPClient(t, srv)
	defer client.close()

	drivePromptFlow(t, client)
}

// TestACPSubprocessPromptFlow is the real end-to-end test: it compiles the rai
// binary, launches `rai acp` as a subprocess, and drives the ACP protocol over
// the process's actual stdin/stdout - exactly what an ACP client like `acpp`
// does. It uses a temporary HOME with a config pointing at the fake provider so
// no API keys are required.
func TestACPSubprocessPromptFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess build/run in -short mode")
	}

	bin := buildRaiBinary(t)
	home := writeFakeHome(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, "acp")
	cmd.Env = append(os.Environ(), "HOME="+home)
	cmd.Stderr = os.Stderr // surface agent logs on failure

	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)

	require.NoError(t, cmd.Start(), "failed to start rai acp")
	t.Cleanup(func() {
		_ = stdin.Close()
		_ = cmd.Wait()
	})

	client := newACPClientIO(t, stdin, stdout)
	drivePromptFlow(t, client)
}

// buildRaiBinary compiles the rai binary from the repository root and returns
// its path.
func buildRaiBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "rai-acptest")

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// The acp package lives one directory below the module root.
	cmd := exec.CommandContext(ctx, "go", "build", "-o", bin, ".")
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "go build failed: %s", out)
	return bin
}

// writeFakeHome creates a temporary HOME directory containing a rai config that
// uses the fake provider as the default model.
func writeFakeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	cfgDir := filepath.Join(home, ".config", "rai")
	require.NoError(t, os.MkdirAll(cfgDir, 0o755))

	const cfg = `providers:
  - name: fake
    type: fake
models:
  - name: fake
    provider: fake
    model: fake-model
    default: true
`
	require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(cfg), 0o644))
	return home
}
