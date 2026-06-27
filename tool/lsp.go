package tool

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/elek/lspc/powernap/pkg/config"
	"github.com/elek/lspc/powernap/pkg/lsp"
	"github.com/elek/lspc/powernap/pkg/lsp/protocol"
	"github.com/elek/lspc/powernap/pkg/registry"
	"github.com/elek/rai/llm"
	"github.com/pkg/errors"
)

type SymbolInput struct {
	Paths []string `json:"path" description:"Relative path of the source files from the actual current directory, to list all including symbols."`
}

func NewLSPAgentTool(ctx context.Context, command string, root string) ([]llm.Tool, func(), error) {
	server, err := NewLSPServer(ctx, command, root)
	if err != nil {
		return nil, func() {}, err
	}

	var res []llm.Tool

	res = append(res, ToAgentTool[SymbolInput]("list-symbols", "List all available symbol names from a source code file (structs, functions, methods...) ", server.Symbols))

	return res, server.Close, nil
}

type LSPServer struct {
	client  *lsp.Client
	project string
}

func (s *LSPServer) Close() {
	ctx := context.TODO()
	_ = s.client.Shutdown(ctx)
}

func (s *LSPServer) Symbols(i SymbolInput) (string, error) {
	buff := bytes.NewBuffer([]byte{})

	ctx := context.TODO()
	for _, path := range i.Paths {
		resp, err := s.client.DocumentSymbols(ctx, path)
		if err != nil {
			return "", errors.WithStack(err)
		}

		var structs, methods, functions []string
		for _, res := range resp {
			switch res.Kind {
			case protocol.Struct:
				structs = append(structs, fmt.Sprintf("      %s from line %d to %d", res.Name, res.Range.Start.Line, res.Range.End.Line))
			case protocol.Method:
				methods = append(methods, fmt.Sprintf("      %s from line %d to %d with signature '%s'", res.Name, res.Range.Start.Line, res.Range.End.Line, res.Detail))
			case protocol.Function:
				functions = append(functions, fmt.Sprintf("      %s from line %d to %d with signature '%s'", res.Name, res.Range.Start.Line, res.Range.End.Line, res.Detail))

			}
		}

		buff.WriteString("Content of " + path + "\n")
		if len(structs) > 0 {
			buff.WriteString("   Defined structs:\n\n")
			buff.WriteString(strings.Join(structs, "\n"))
			buff.WriteString("\n\n")
		}

		if len(functions) > 0 {
			buff.WriteString("   Defined functions:\n\n")
			buff.WriteString(strings.Join(functions, "\n"))
			buff.WriteString("\n\n")
		}

		if len(methods) > 0 {
			buff.WriteString("   Defined methods:\n\n")
			buff.WriteString(strings.Join(methods, "\n"))
			buff.WriteString("\n\n")
		}

	}
	return buff.String(), nil
}

func NewLSPServer(ctx context.Context, command string, root string) (*LSPServer, error) {
	reg := registry.New()
	m := config.NewManager()
	m.LoadDefaults()
	err := reg.LoadConfig(m)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	project, _ := filepath.Abs(root)

	cl, err := reg.StartServer(ctx, command, project)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &LSPServer{
		client:  cl,
		project: project,
	}, nil
}

func ToAgentTool[I any](name string, desc string, f func(I) (string, error)) llm.Tool {
	return llm.NewTool[I](name, desc, func(ctx context.Context, input I) (string, error) {
		return f(input)
	})
}
