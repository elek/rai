package acp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerInitialize(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientCapabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}` + "\n"

	in := strings.NewReader(input)
	out := &bytes.Buffer{}

	srv := NewServer(nil)
	srv.ServeIO(in, out)

	var resp Response
	err := json.NewDecoder(out).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Nil(t, resp.Error)
}

func TestFullFlow(t *testing.T) {
	messages := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientCapabilities":{},"clientInfo":{"name":"test"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"session/new","params":{"cwd":"/tmp","mcpServers":[]}}`,
	}
	input := strings.Join(messages, "\n") + "\n"

	in := strings.NewReader(input)
	out := &bytes.Buffer{}

	srv := NewServer(nil)
	srv.ServeIO(in, out)

	decoder := json.NewDecoder(out)

	var initResp Response
	err := decoder.Decode(&initResp)
	assert.NoError(t, err)
	assert.Nil(t, initResp.Error)

	var sessResp Response
	err = decoder.Decode(&sessResp)
	assert.NoError(t, err)
	assert.Nil(t, sessResp.Error)

	resultBytes, _ := json.Marshal(sessResp.Result)
	var sessResult NewSessionResult
	err = json.Unmarshal(resultBytes, &sessResult)
	assert.NoError(t, err)
	assert.NotEmpty(t, sessResult.SessionID)
}

func TestMethodNotFound(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"nonexistent","params":{}}` + "\n"
	in := strings.NewReader(input)
	out := &bytes.Buffer{}

	srv := NewServer(nil)
	srv.ServeIO(in, out)

	var resp Response
	err := json.NewDecoder(out).Decode(&resp)
	assert.NoError(t, err)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, -32601, resp.Error.Code)
}

func TestSessionNotFound(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"session/prompt","params":{"sessionId":"nonexistent","prompt":[{"type":"text","text":"hello"}]}}` + "\n"
	in := strings.NewReader(input)
	out := &bytes.Buffer{}

	srv := NewServer(nil)
	srv.ServeIO(in, out)

	var resp Response
	err := json.NewDecoder(out).Decode(&resp)
	assert.NoError(t, err)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, -32002, resp.Error.Code)
}
