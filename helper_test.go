package golden

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read file")
	return b
}

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()
	err := os.WriteFile(path, content, 0644)
	require.NoError(t, err, "failed to write file")
}
