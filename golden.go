package golden

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"reflect"
	"strconv"
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
type Option func(*testing.T, *golden, any)

// SkippedFields replaces the value of the fields with "--* SKIPPED *--".
// The fields are specified by their GJSON path.
// See https://github.com/tidwall/gjson/blob/master/SYNTAX.md
//
// The rules are as follows:
//   - If the field is a nilable type and is nil, then it is not marked as skipped, since its "null" value is already deterministic.
//   - If the field is a nilable type and has a non-nil value, then it is marked as skipped.
//   - If the field is a non-nilable type, then it is marked as skipped.
//
// Example: "data.user.Name" for the following JSON:
//
//	{
//	    "data": {
//	        "user": {
//	            "Name": "--* SKIPPED *--",
//	        }
//	    }
//	}
func SkippedFields(fields ...string) Option {
	return func(t *testing.T, g *golden, got any) {
		// prefix the fields with a dot to simplify comparisons later on
		for i, field := range fields {
			fields[i] = "." + field
		}

		walkGotValueForSkippingFields(t, g, got, "", fields)
	}
}

func markFieldAsSkipped(t *testing.T, json []byte, fieldPath string) []byte {
	var err error
	json, err = sjson.SetBytes(json, fieldPath, "--* SKIPPED *--")
	require.NoError(t, err, "skipping field = %s", fieldPath)
	return json
}

// walkGotValueForSkippingFields walks the got value and marks the fields that are to be skipped. The fields are
// specified by their GJSON path.
func walkGotValueForSkippingFields(t *testing.T, g *golden, next any, currentPath string, fields []string) {
	value := reflect.ValueOf(next)

	switch value.Kind() {
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i)
			var fieldName string
			fieldTag := value.Type().Field(i).Tag.Get("json")
			if fieldTag != "" {
				fieldName = fieldTag
			} else {
				fieldName = value.Type().Field(i).Name
			}
			fieldPath := currentPath + "." + fieldName

			if shouldReturn := processFieldPath(t, g, field, fieldPath, fields); shouldReturn {
				return
			}
		}
	case reflect.Map:
		// Return early if the map key type is not string, since we cannot build a path with a non-string key.
		keyType := value.Type().Key()
		if keyType.Kind() != reflect.String {
			return
		}

		var fieldPath string
		for _, key := range value.MapKeys() {
			fieldPath = currentPath + "." + key.String()
			if shouldReturn := processFieldPath(t, g, value.MapIndex(key), fieldPath, fields); shouldReturn {
				return
			}
		}
	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			elem := value.Index(i)
			var fieldPath string
			if currentPath == "" {
				fieldPath = strconv.Itoa(i)
			} else {
				fieldPath = currentPath + "." + strconv.Itoa(i)
			}
			walkGotValueForSkippingFields(t, g, elem.Interface(), fieldPath, fields)
		}
	case reflect.Array:
		for i := 0; i < value.Len(); i++ {
			elem := value.Index(i)
			fieldPath := currentPath + "." + strconv.Itoa(i)
			walkGotValueForSkippingFields(t, g, elem.Interface(), fieldPath, fields)
		}
	default:
	}
}

func processFieldPath(t *testing.T, g *golden, field reflect.Value, fieldPath string, fields []string) bool {
	proceed := false
	onlyPrefixFound := true
	for _, fld := range fields {
		if strings.HasPrefix(fld, fieldPath) {
			proceed = true
			if fieldPath == fld {
				onlyPrefixFound = false
			}
			break
		}
	}
	// Return early if the field is not in the fields list
	if !proceed {
		return false
	}
	// Keep walking if only a partial field path is found
	if onlyPrefixFound {
		walkGotValueForSkippingFields(t, g, field.Interface(), fieldPath, fields)
		return true
	}

	// If the field is nilable and is nil, then skip marking it as skipped since its value will be
	// deterministic.
	switch field.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		if field.IsNil() {
			return false
		}
	}
	// Otherwise, mark the field as skipped
	g.result = markFieldAsSkipped(t, g.result, fieldPath[1:]) // remove the leading dot
	return false
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
	return func(t *testing.T, g *golden, _ any) {
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
	return func(t *testing.T, g *golden, _ any) {
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
		applyOn(t, g, got)
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
