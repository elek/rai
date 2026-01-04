package templates

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRender(t *testing.T) {
	inp := `
foobar
<system>
system prompt
</system>
<shell>
echo "asd"
</shell>
`
	rendered, err := GoTemplateRender(inp, map[string]any{})
	require.NoError(t, err)
	require.Equal(t, "system prompt", rendered.System)

}
