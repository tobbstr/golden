package golden

import (
	"encoding/json"
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/sjson"
)

var update = flag.Bool("update", false, "Update golden test file")

func JSON(t *testing.T, want string, got any, skipFields ...string) {
	t.Helper()
	var gotBytes []byte

	gotBytes, err := json.MarshalIndent(got, "", "    ")
	require.NoError(t, err, "failed to marshal got")

	for _, field := range skipFields {
		gotBytes, err = sjson.SetBytes(gotBytes, field, "--* SKIPPED *--")
		require.NoError(t, err, "failed to skip field = %s", field)
	}

	if update != nil && *update {
		overwriteGoldenFile(t, want, gotBytes)
		return
	}

	goldenBytes, err := os.ReadFile(want)
	require.NoError(t, err, "failed to read golden file")
	assert.Equal(t, goldenBytes, gotBytes, "comparison with golden file failed")
	// require.Equal(t, goldenBytes, gotBytes, "comparison with golden file failed")
}

func overwriteGoldenFile(t *testing.T, want string, got []byte) {
	t.Helper()
	err := os.WriteFile(want, got, 0644)
	if err != nil {
		t.Fatalf("failed to overwrite golden file: %v", err)
	}
}
