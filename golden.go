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

var fileWritten = make(map[string]struct{})

func JSON(t *testing.T, want string, got any, skipFields ...string) {
	t.Helper()
	var gotBytes []byte
	gotBytes, err := json.MarshalIndent(got, "", "    ")
	require.NoError(t, err, "marshalling got")

	for _, field := range skipFields {
		gotBytes, err = sjson.SetBytes(gotBytes, field, "--* SKIPPED *--")
		require.NoError(t, err, "skipping field = %s", field)
	}

	if update != nil && *update {
		writeGoldenFile(t, want, gotBytes)
		return
	}

	goldenBytes, err := os.ReadFile(want)
	require.NoError(t, err, "reading golden file")
	assert.Equal(t, goldenBytes, gotBytes, "comparing with golden file")
}

func writeGoldenFile(t *testing.T, want string, got []byte) {
	t.Helper()
	// check for duplicate writes
	if _, written := fileWritten[want]; written {
		t.Fatalf("writing golden file = %s: attempting to write to the same file twice", want)
		return
	}

	err := os.WriteFile(want, got, 0644)
	if err != nil {
		t.Fatalf("writing golden file: %v", err)
	}

	// mark the file as written
	fileWritten[want] = struct{}{}
}
