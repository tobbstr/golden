package golden

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	gjsonpkg "github.com/tobbstr/golden/gjson"
	"google.golang.org/grpc/status"
)

// filesWritten keeps track of the files that have been written to. This is to prevent writing to the same file twice.
var filesWritten sync.Map

// OptionType represents the type of an Option for sorting purposes
type OptionType int

const (
	// OptionTypeCheck indicates options that perform validation/checking without modifying the JSON structure
	OptionTypeCheck OptionType = iota
	// OptionTypeModifier indicates options that modify the JSON structure or content
	OptionTypeModifier
)

// golden is a model of the golden file.
type golden struct {
	result []byte
}

// Option is an interface that defines operations on the golden file. It is used to apply modifications or checks
// to the golden file before comparing it with the actual result.
type Option interface {
	// Apply executes the option's operation on the golden file.
	//
	// Parameters:
	//   - t: the testing.T value.
	//   - failNow: if true, if any errors happen the test is marked as failed and stops execution. Otherwise, the test is
	//     marked as failed, but execution continues.
	//   - g: a wrapper around the resulting golden file.
	//   - path: the path to the golden file.
	Apply(t *testing.T, failNow bool, g *golden, path string)

	// IsType returns the type of this option for sorting purposes.
	// Check options should run before modifier options to validate the original data.
	IsType() OptionType
}

// KeepNull overrides the WithSkippedFields' default behaviour for a specific field. It is used when the caller wants to
// distinguish between a non-null value and a null value, which would otherwise be replaced with "--* SKIPPED *--".
// With the WithSkippedFields default behaviour the fields are always replaced with skipped, but in some cases it is
// desirable to be able to distinguish whether a field had a null value or not, while not caring what the actual value
// was. An example of this, is an optional field such as an updatedAt timestamp and the caller simply wants to know
// whether it was set or not.
//
// The rules for replacing a field's value are as follows:
//   - If the field's JSON-value is null, then it is left untouched.
//   - If the field's JSON-value is not null, then it is replaced with skipped.
//
// Example: homePhone field is of a nilable Go-type and has the JSON-value null
//
// Before calling WithSkippedFields() the JSON is:
//
//	{
//	    "data": {
//	        "user": {
//	            "homePhone": null,
//	        }
//	    }
//	}
//
// After calling WithSkippedFields(KeepNull("data.user.homePhone")) the JSON is still the same since the value of homePhone was null:
//
//	{
//	    "data": {
//	        "user": {
//	            "homePhone": null,
//	        }
//	    }
//	}
//
// NOTE! Had it not been null, it would have been replaced with "--* SKIPPED *--".
type KeepNull string

// WithSkippedFields replaces values of the fields with "--* SKIPPED *--".
// The fields are specified by their GJSON path.
// See https://github.com/tidwall/gjson/blob/master/SYNTAX.md
//
// It accepts either strings or KeepNulls. For strings the values are always replaced by "--* SKIPPED *--".
// For KeepNulls, see the KeepNull definition for details.
//
// Example: Replacing the value of the "Name" field with "--* SKIPPED *--"
//
// Before calling WithSkippedFields("data.user.Name") the JSON is:
//
//	{
//	    "data": {
//	        "user": {
//	            "Name": "John",
//	        }
//	    }
//	}
//
// After calling WithSkippedFields("data.user.Name") the JSON is:
//
//	{
//	    "data": {
//	        "user": {
//	            "Name": "--* SKIPPED *--",
//	        }
//	    }
//	}
//
// skippedFieldsOption implements Option for skipping fields
type skippedFieldsOption[T string | KeepNull] struct {
	fields []T
}

func (s skippedFieldsOption[T]) Apply(t *testing.T, failNow bool, g *golden, _ string) {
	for _, fld := range s.fields {
		var path string
		var keepNull bool
		switch v := any(fld).(type) {
		case KeepNull:
			path = string(v)
			keepNull = true
		case string:
			path = v
			keepNull = false
		default:
			if failNow {
				require.Fail(t, "invalid field type", "field = %T", fld)
			}
			assert.Fail(t, "invalid field type", "field = %T", fld)
			return
		}

		expandedPaths := gjsonpkg.ExpandPath(g.result, path)
		for _, expPath := range expandedPaths {
			gres := gjson.GetBytes(g.result, expPath)
			if !gres.Exists() {
				if failNow {
					require.Fail(t, "path not found", "path = %s", expPath)
				}
				assert.Fail(t, "path not found", "path = %s", expPath)
				continue
			}
			if keepNull && gres.Type == gjson.Null {
				continue
			}
			res, err := sjson.SetBytes(g.result, expPath, "--* SKIPPED *--")
			if err != nil {
				if failNow {
					require.Fail(t, "setting field value", "path = %s", expPath)
				}
				assert.Fail(t, "setting field value", "path = %s", expPath)
				continue
			}
			g.result = res
		}
	}
}

func (s skippedFieldsOption[T]) IsType() OptionType {
	return OptionTypeModifier
}

func WithSkippedFields[T string | KeepNull](fields ...T) Option {
	return skippedFieldsOption[T]{fields: fields}
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

// WithFieldComments adds comments to fields in the golden file. This is useful for making it easier for the reader to
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
// fieldCommentsOption implements Option for adding field comments
type fieldCommentsOption struct {
	fieldComments []FieldComment
}

func (f fieldCommentsOption) Apply(t *testing.T, failNow bool, g *golden, _ string) {
	// Add the comments to the fields
	var err error
	for _, fieldComment := range f.fieldComments {
		value := gjson.GetBytes(g.result, fieldComment.Path)
		if !value.Exists() {
			if failNow {
				require.Fail(t, "path not found", "path = %s", fieldComment.Path)
			}
			assert.Fail(t, "path not found", "path = %s", fieldComment.Path)
			continue
		}
		g.result, err = sjson.SetRawBytes(g.result, fieldComment.Path, []byte(value.Raw+` // `+fieldComment.Comment))
		if !failNow && !assert.NoError(t, err, "setting field comment for path = %s", fieldComment.Path) {
			return
		} else {
			require.NoError(t, err, "setting field comment for path = %s", fieldComment.Path)
		}
	}

	// Fix misplaced commas. When the field value is replaced, if the line ends with a comma, the comment is added
	// before the comma. This function moves the comma before the comment.
	correctedJSON, err := correctMisplacedCommas(g.result)
	if !failNow && !assert.NoError(t, err, "correcting misplaced commas in JSON") {
		return
	} else {
		require.NoError(t, err, "correcting misplaced commas in JSON")
	}
	g.result = correctedJSON
}

func (f fieldCommentsOption) IsType() OptionType {
	return OptionTypeModifier
}

func WithFieldComments(fieldComments []FieldComment) Option {
	return fieldCommentsOption{fieldComments: fieldComments}
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

// WithFileComment adds a comment to the top of the golden file. This is useful for providing context to the reader.
//
// NOTE! Adding comments to JSON makes it invalid, since JSON does not support comments. To keep you IDE happy,
// i.e., for it not to show errors, make the file extension .jsonc. To do that, make sure the "want" file argument
// in the JSON() function call has the .jsonc extension.
// fileCommentOption implements Option for adding file comments
type fileCommentOption struct {
	comment string
}

func (f fileCommentOption) Apply(t *testing.T, _ bool, g *golden, _ string) {
	g.result = append([]byte("/*\n"+f.comment+"\n*/\n\n"), g.result...)
}

func (f fileCommentOption) IsType() OptionType {
	return OptionTypeModifier
}

func WithFileComment(comment string) Option {
	return fileCommentOption{comment: comment}
}

// UpdateGoldenFiles updates the golden files with the actual values instead of comparing with them.
// This is useful when the actual values are correct and the golden files need to be updated.
//
// NOTE! This option should normally not be invoked directly. Instead, set the environment variable
// "UPDATE_GOLDENS" to "1" to update the golden files, when running the tests.
//
// Example: UPDATE_GOLDENS=1 go test ./...
// updateGoldenFilesOption implements Option for updating golden files
type updateGoldenFilesOption struct{}

func (u updateGoldenFilesOption) Apply(t *testing.T, failNow bool, g *golden, path string) {
	writeGoldenFile(t, failNow, path, g.result)
}

func (u updateGoldenFilesOption) IsType() OptionType {
	return OptionTypeModifier
}

func UpdateGoldenFiles() Option {
	return updateGoldenFilesOption{}
}

// CheckNotZeroTime checks if the time at the specified path is not zero, and fails the test if the time is zero.
//
// Parameters:
//   - path: the GJSON path to the time.
//   - layout: the layout of the time. See https://golang.org/pkg/time/#pkg-constants
//
// Example: CheckNotZeroTime("data.user.updatedAt", time.RFC3339)
// checkNotZeroTimeOption implements Option for checking non-zero times
type checkNotZeroTimeOption struct {
	path   string
	layout string
}

func (c checkNotZeroTimeOption) Apply(t *testing.T, failNow bool, g *golden, _ string) {
	expandedPaths := gjsonpkg.ExpandPath(g.result, c.path)
	for _, expPath := range expandedPaths {
		res := gjson.GetBytes(g.result, expPath)
		if !res.Exists() {
			if failNow {
				require.Fail(t, "path not found in JSON", "path = %s", expPath)
			}
			assert.Fail(t, "path not found in JSON", "path = %s", expPath)
			return
		}
		if res.Type != gjson.String {
			if failNow {
				require.Fail(t, "path's value is not a string", "path = %s", expPath)
			}
			assert.Fail(t, "path's value is not a string", "path = %s", expPath)
			return
		}

		tide, err := time.Parse(c.layout, res.String())
		if err != nil {
			if failNow {
				require.Fail(t, "parsing time", "path = %s", expPath)
			}
			assert.Fail(t, "parsing time", "path = %s", expPath)
			return
		}

		if tide.IsZero() {
			if failNow {
				require.Fail(t, "time is zero", "path = %s", expPath)
			}
			assert.Fail(t, "time is zero", "path = %s", expPath)
		}
	}
}

func (c checkNotZeroTimeOption) IsType() OptionType {
	return OptionTypeCheck
}

func CheckNotZeroTime(path string, layout string) Option {
	return checkNotZeroTimeOption{path: path, layout: layout}
}

// CheckEqualTimes checks if the times at the specified paths are equal, and fails the test if they are not.
//
// Parameters:
//   - a: the GJSON path to the first time. NO WILDCARDS SUPPORTED!!!
//   - b: the GJSON path to the second time. NO WILDCARDS SUPPORTED!!!
//   - layout: the layout of the time values. See https://golang.org/pkg/time/#pkg-constants
//
// Example: CheckEqualTimes("data.user.createdAt", "data.user.updatedAt", time.RFC3339)
// checkEqualTimesOption implements Option for checking equal times
type checkEqualTimesOption struct {
	a, b, layout string
}

func (c checkEqualTimesOption) Apply(t *testing.T, failNow bool, g *golden, _ string) {
	aRes := gjson.GetBytes(g.result, c.a)
	if !aRes.Exists() {
		if failNow {
			require.Fail(t, "a not found in JSON", "path = %s", c.a)
		}
		assert.Fail(t, "a not found in JSON", "path = %s", c.a)
		return
	}
	if aRes.Type != gjson.String {
		if failNow {
			require.Fail(t, "a's value is not a string", "path = %s", c.a)
		}
		assert.Fail(t, "a's value is not a string", "path = %s", c.a)
		return
	}

	aTide, err := time.Parse(c.layout, aRes.String())
	if err != nil {
		if failNow {
			require.Fail(t, "parsing a's time", "path = %s", c.a)
		}
		assert.Fail(t, "parsing a's time", "path = %s", c.a)
		return
	}

	bRes := gjson.GetBytes(g.result, c.b)
	if !bRes.Exists() {
		if failNow {
			require.Fail(t, "b not found in JSON", "path = %s", c.b)
		}
		assert.Fail(t, "b not found in JSON", "path = %s", c.b)
		return
	}
	if bRes.Type != gjson.String {
		if failNow {
			require.Fail(t, "b's value is not a string", "path = %s", c.b)
		}
		assert.Fail(t, "b's value is not a string", "path = %s", c.b)
		return
	}

	bTide, err := time.Parse(c.layout, bRes.String())
	if err != nil {
		if failNow {
			require.Fail(t, "parsing b's time", "path = %s", c.b)
		}
		assert.Fail(t, "parsing b's time", "path = %s", c.b)
		return
	}

	if !aTide.Equal(bTide) {
		if failNow {
			require.Fail(t, "times are not equal", "a = %s, b = %s", aTide.String(), bTide.String())
		}
		assert.Fail(t, "times are not equal", "a = %s, b = %s", aTide.String(), bTide.String())
	}
}

func (c checkEqualTimesOption) IsType() OptionType {
	return OptionTypeCheck
}

func CheckEqualTimes(a, b, layout string) Option {
	return checkEqualTimesOption{a: a, b: b, layout: layout}
}

// AssertJSON compares the expected JSON (want) with the actual value (got), and if they are different it marks
// the test as failed, but continues execution. The expected JSON is read from a golden file.
//
// To update the golden file with the actual value instead of comparing with it, set the environment variable
// "UPDATE_GOLDENS" to "1" when running the tests.
//
// Example: UPDATE_GOLDENS=1 go test ./...
func AssertJSON(t *testing.T, want string, got any, opts ...Option) {
	t.Helper()
	if os.Getenv("UPDATE_GOLDENS") == "1" {
		opts = append(opts, UpdateGoldenFiles())
	}
	compareJSON(t, false, want, got, opts...)
}

// RequireJSON does the same as AssertJSON, but if the expected JSON (want) and the actual value (got) are different,
// it marks the test as failed and stops execution.
func RequireJSON(t *testing.T, want string, got any, opts ...Option) {
	t.Helper()
	if os.Getenv("UPDATE_GOLDENS") == "1" {
		opts = append(opts, UpdateGoldenFiles())
	}
	compareJSON(t, true, want, got, opts...)
}

// sortOptions sorts the provided options so that check functions run before modifier functions.
// This ensures that validation operations happen on the original JSON before any modifications.
func sortOptions(opts []Option) []Option {
	var checkOpts []Option
	var modifierOpts []Option

	for _, opt := range opts {
		switch opt.IsType() {
		case OptionTypeCheck:
			checkOpts = append(checkOpts, opt)
		case OptionTypeModifier:
			modifierOpts = append(modifierOpts, opt)
		default:
			// Unknown types treated as modifier by default
			modifierOpts = append(modifierOpts, opt)
		}
	}

	// Combine check functions first, then modifier functions
	result := make([]Option, 0, len(checkOpts)+len(modifierOpts))
	result = append(result, checkOpts...)
	result = append(result, modifierOpts...)
	return result
}

func compareJSON(t *testing.T, failNow bool, want string, got any, opts ...Option) {
	t.Helper()

	// Handle gRPC status errors by extracting their protobuf representation, as JSON marshaling skips unexported fields.
	if err, ok := got.(error); ok {
		if st, ok := status.FromError(err); ok {
			got = st.Proto()
		}
	}

	var gotBytes []byte
	gotBytes, err := json.MarshalIndent(got, "", "    ")
	if !failNow && !assert.NoError(t, err, "marshalling got") {
		return
	} else {
		require.NoError(t, err, "marshalling got")
	}

	g := &golden{result: gotBytes}

	// Sort options so that check functions run before modifier functions
	sortedOpts := sortOptions(opts)
	for _, opt := range sortedOpts {
		opt.Apply(t, failNow, g, want)
	}

	goldenBytes, err := os.ReadFile(want)
	if !failNow && !assert.NoError(t, err, "reading golden file") {
		return
	} else {
		require.NoError(t, err, "reading golden file")
	}

	if failNow {
		require.Equal(t, string(goldenBytes), string(g.result), "comparing with golden file")
	} else {
		assert.Equal(t, string(goldenBytes), string(g.result), "comparing with golden file")
	}
}

func writeGoldenFile(t *testing.T, required bool, path string, got []byte) {
	t.Helper()
	// check for duplicate writes
	if _, written := filesWritten.Load(path); written {
		if !required {
			assert.Equal(t, false, written, "writing golden file = %s: attempting to write to the same file twice", path)
			return
		}
		require.Equal(t, false, written, "writing golden file = %s: attempting to write to the same file twice", path)
		return
	}

	err := os.WriteFile(path, got, 0644)
	if !required {
		assert.NoError(t, err, "writing golden file = %s", path)
		return
	}
	require.NoError(t, err, "writing golden file = %s", path)

	// mark the file as written
	filesWritten.Store(path, struct{}{})
}
