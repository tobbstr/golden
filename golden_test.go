package golden

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSON(t *testing.T) {
	// Save the original value of the update flag, since this test modifies it and if we don't restore it,
	// it will affect other tests.
	originalUpdate := update

	type args struct {
		t          *testing.T
		want       string
		got        any
		skipFields []string
	}
	type given struct {
		args args
		// update flag is set to true when the golden file should be updated
		update bool
	}
	type want struct {
		// json is the expected JSON content of the golden file
		json string
		// goldenFileUpdated is true when the golden file should be updated in the test case
		goldenFileUpdated bool
		// failed is true when the test case should fail
		failed bool
	}
	type test struct {
		name  string
		given given
		want  want
	}
	tests := []test{
		{
			name: "overwrites the golden file when update flag is set to true",
			given: given{
				args: args{
					t:    &testing.T{},
					want: "testdata/json/overwrites.json",
					got:  map[string]interface{}{"name": "John", "age": 30},
				},
				update: true,
			},
			want: want{
				json: `{
    "age": 30,
    "name": "John"
}`,
				goldenFileUpdated: true,
			},
		},
		{
			name: "passes when the golden file's content is equal to the got JSON",
			given: given{
				args: args{
					t:    &testing.T{},
					want: "testdata/json/same_content.json",
					got:  map[string]interface{}{"name": "John", "age": 30},
				},
				update: false,
			},
			want: want{
				json: `{
    "age": 30,
    "name": "John"
}`,
				goldenFileUpdated: false,
			},
		},
		{
			name: "skips fields when provided",
			given: given{
				args: args{
					t:    &testing.T{},
					want: "testdata/json/skips_field.json",
					got: map[string]interface{}{
						"name": "John",
						"age":  30,
						"colour": map[string]interface{}{
							"hair": "black",
							"eyes": "brown",
						},
					},
					skipFields: []string{"colour.hair", "colour.eyes"},
				},
				update: false,
			},
			want: want{
				json: `{
    "age": 30,
    "colour": {
        "eyes": "--* SKIPPED *--",
        "hair": "--* SKIPPED *--"
    },
    "name": "John"
}`,
				goldenFileUpdated: false,
			},
		},
		{
			name: "fails when the golden file's content is different from the got JSON",
			given: given{
				args: args{
					t:    &testing.T{},
					want: "testdata/json/json_different.json",
					got: map[string]interface{}{
						"name": "John",
						"age":  30,
						"colour": map[string]interface{}{
							"hair": "black",
							"eyes": "brown",
						},
					},
				},
				update: false,
			},
			want: want{
				json: `{
    "age": 30,
    "colour": {
        "eyes": "green",
        "hair": "blonde"
    },
    "name": "John"
}`,
				goldenFileUpdated: false,
				failed:            true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			/* ---------------------------------- Given --------------------------------- */
			update = &tt.given.update
			initialGoldenFile := readFile(t, tt.given.args.want)

			// Restore the initial golden file if it will be updated
			if tt.want.goldenFileUpdated {
				defer writeFile(t, tt.given.args.want, initialGoldenFile)
			}

			/* ---------------------------------- When ---------------------------------- */
			JSON(tt.given.args.t, tt.given.args.want, tt.given.args.got, tt.given.args.skipFields...)

			/* ---------------------------------- Then ---------------------------------- */
			// Read the golden file
			got := readFile(t, tt.given.args.want)

			if tt.want.goldenFileUpdated {
				require.NotEqual(t, initialGoldenFile, got, "golden file should be updated")
			}

			if tt.want.failed {
				// Compare the golden file with the expected JSON
				require.NotEqual(t, tt.want.json, string(got), "comparison with golden file failed")
				// Check that the test failed
				require.True(t, tt.given.args.t.Failed(), "test failed")
			} else {
				// Compare the golden file with the expected JSON
				require.Equal(t, tt.want.json, string(got), "comparison with golden file failed")
				// Check that the test passed
				require.False(t, tt.given.args.t.Failed(), "test failed")
			}
		})
	}

	update = originalUpdate // Restore the original value of the update flag
}

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
