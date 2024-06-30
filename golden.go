package golden

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// update is a flag that is used to update the golden test files. If the flag is set to true, the golden test files
// will be updated with the new test results.
//
//	Example:
//	 * To set the flag to true, run 'go test -update'
//	 * Example: To set the flag to false, run 'go test'
var update = flag.Bool("update", false, "Update golden test file")

// fileWritten keeps track of the files that have been written to. This is to prevent writing to the same file twice.
var fileWritten = make(map[string]struct{})

// golden is a model of the golden file.
type golden struct {
	result []byte
}

// Option is a function that modifies the golden file. It is used to apply modifications to the golden file before
// comparing it with the actual result.
type Option func(*testing.T, *golden)

// SkippedFields replaces the value of the fields with "--* SKIPPED *--".
// The fields are specified by their JSON path.
//
// Example: "data.user.name" for the following JSON:
//
//	{
//	    "data": {
//	        "user": {
//	            "name": "--* SKIPPED *--",
//	        }
//	    }
//	}
func SkippedFields(fields ...string) Option {
	return func(t *testing.T, g *golden) {
		var err error
		for _, field := range fields {
			g.result, err = sjson.SetBytes(g.result, field, "--* SKIPPED *--")
			require.NoError(t, err, "skipping field = %s", field)
		}
	}
}

// FieldComment is a comment that describes what to look for when inspecting the JSON field. The comment is added to
// the field specified by its Path.
type FieldComment struct {
	// Path is the GJSON path to the field.
	// See https://github.com/tidwall/gjson/blob/master/SYNTAX.md
	//
	// Example: "data.user.name" for the following JSON:
	//
	//	{
	//	    "data": {
	//	        "user": {
	//	            "name": "John",
	//	        }
	//	    }
	//	}
	Path string
	// Comment is the comment that describes what to look for when inspecting the JSON field.
	Comment string
}

// FieldComments adds comments to fields in the golden file. This is useful for making it easier for the reader to
// understand what to look for when inspecting the JSON field.
//
//	Example:
//	 {
//	   "age": 30, // This my field comment
//	 }
//
// NOTE! Adding comments to JSON makes it invalid, since JSON does not support comments. To keep you IDE happy,
// i.e., for it not to show errors, make the file extension .jsonc. To do that, make sure the "want" file argument
// in the JSON() function call has the .jsonc extension.
func FieldComments(fieldComments ...FieldComment) Option {
	return func(t *testing.T, g *golden) {
		// Add the comments to the fields
		var err error
		for _, fieldComment := range fieldComments {
			value := gjson.GetBytes(g.result, fieldComment.Path)
			if !value.Exists() {
				continue
			}
			g.result, err = sjson.SetRawBytes(g.result, fieldComment.Path, []byte(value.Raw+` // `+fieldComment.Comment))
			require.NoError(t, err, "setting field comment for path = %s", fieldComment.Path)
		}

		// Fix misplaced commas. When the field value is replaced, if the line ends with a comma, the comment is added
		// before the comma. This function moves the comma before the comment.
		correctedJSON, err := correctMisplacedCommas(g.result)
		require.NoError(t, err, "correcting misplaced commas in JSON")
		g.result = correctedJSON
	}
}

// correctMisplacedCommas corrects commas directly after a comment in a JSON file.
func correctMisplacedCommas(input []byte) ([]byte, error) {
	var buffer bytes.Buffer
	lines := strings.Split(string(input), "\n")

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check if line contains a comment
		if commentIndex := strings.Index(line, "//"); commentIndex != -1 {
			// Remove any trailing comma after the comment
			comment := strings.TrimSuffix(line[commentIndex:], ",")

			// Extract the main content part and check if it needs a comma
			content := line[:commentIndex]
			if i+1 < len(lines) {
				nextLine := strings.TrimLeft(lines[i+1], " ")
				if !strings.HasPrefix(nextLine, "}") && !strings.HasPrefix(nextLine, "]") {
					// Remove any trailing whitespace
					content = strings.TrimRight(content, " ")
					content = content + ","
				} else {
					buffer.WriteString(line)
					buffer.WriteString("\n")
					continue
				}
			}

			// Add the line with correct content and comment
			buffer.WriteString(content)
			buffer.WriteString(" ")
			buffer.WriteString(comment)
			buffer.WriteString("\n")
		} else {
			buffer.WriteString(line)
			buffer.WriteString("\n")
		}
	}

	return buffer.Bytes(), nil
}

// FileComment adds a comment to the top of the golden file. This is useful for providing context to the reader.
//
// NOTE! Adding comments to JSON makes it invalid, since JSON does not support comments. To keep you IDE happy,
// i.e., for it not to show errors, make the file extension .jsonc. To do that, make sure the "want" file argument
// in the JSON() function call has the .jsonc extension.
func FileComment(comment string) Option {
	return func(t *testing.T, g *golden) {
		g.result = append([]byte("/*\n"+comment+"\n*/\n\n"), g.result...)
	}
}

// JSON compares the expected JSON (want) with the actual JSON (got). The expected JSON is read from a golden file.
// To update the golden file, set the update flag to true.
func JSON(t *testing.T, want string, got any, opts ...Option) {
	t.Helper()
	var gotBytes []byte
	gotBytes, err := json.MarshalIndent(got, "", "    ")
	require.NoError(t, err, "marshalling got")

	g := &golden{result: gotBytes}
	for _, applyOn := range opts {
		applyOn(t, g)
	}

	if update != nil && *update {
		writeGoldenFile(t, want, g.result)
		return
	}

	goldenBytes, err := os.ReadFile(want)
	require.NoError(t, err, "reading golden file")
	assert.Equal(t, goldenBytes, g.result, "comparing with golden file")
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
