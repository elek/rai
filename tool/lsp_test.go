package tool

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLspSymbols(t *testing.T) {
	ctx := t.Context()
	lsp, err := NewLSPServer(ctx, "gopls", ".")
	require.NoError(t, err)
	res, err := lsp.Symbols(SymbolInput{Paths: []string{"tool/lsp.go"}})
	require.NoError(t, err)
	fmt.Println(res)
}
