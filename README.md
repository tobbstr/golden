# Golden

Welcome to the Golden, a robust Go library designed to enhance your testing workflow by enabling easy management and
comparison of golden files. A golden file is a reference file used to validate the output of a system
against a known, correct result. It serves as a benchmark or "gold standard" to ensure that the system produces
the expected output consistently. This library is perfect for developers looking to streamline their testing process,
ensuring that their applications work as expected.

## Features

- **Golden File Management**: Easily manage your expected return values with golden files.
- **Comment Support in JSON**: Unique functionality that allows you to add comments to your JSON files, enhancing 
  readability and maintainability, despite JSON's native lack of comment support. Use the `.jsonc` extension for 
  error-free IDE integration.
- **Precise JSON Comparison**: Compare your expected JSON (from golden files) with the actual output of your code, 
  with detailed reporting on discrepancies.
- **Flexible Configuration**: Customize your testing with various options, including the ability to mark fields as 
  skipped, whose values are non-deterministic.
- **Time Validation**: Built-in support for validating timestamps and comparing time values.
- **gRPC Support**: Automatic handling of gRPC status errors.

## When to Use This Library

Golden file testing is particularly well-suited for:

- **API Response Testing**: Validate REST or gRPC endpoints return the expected structure and 
  values, especially useful for integration tests.
- **Complex Data Structures**: When your output contains deeply nested objects or arrays that would 
  be tedious to manually assert field-by-field.
- **Serialization/Marshaling**: Verify that your data structures serialize correctly to JSON.
- **Regression Testing**: Ensure that changes to your codebase don't unexpectedly alter outputs.
- **Template/Document Generation**: Validate generated reports, emails, or documents against 
  expected formats.
- **Snapshot Testing**: Similar to Jest snapshots in JavaScript, capture and compare complex outputs 
  over time.

## When NOT to Use This Library

Consider alternatives when:

- **Simple Unit Tests**: For straightforward tests with a few fields, direct assertions 
  (`if got != want`) are clearer and more maintainable.
- **Frequently Changing Output**: If your data structure changes often, maintaining golden files 
  becomes a burden rather than a benefit.
- **Testing Business Logic**: Golden files test outputs, not logic. Use traditional unit tests to 
  verify algorithms, calculations, and business rules.
- **Entirely Non-Deterministic Data**: When every field is random or time-based and you'd need to 
  skip everything. Better to test specific properties or use other validation methods.
- **Performance-Critical Tests**: Golden files involve file I/O which adds overhead. For benchmarks 
  or performance tests, use Go's built-in benchmarking tools.
- **Testing Internal State**: Golden files work best for outputs (return values, responses). Use 
  traditional testing for internal state verification.

**Rule of Thumb**: If you find yourself writing more than 5-10 individual field assertions in a 
test, or if you're testing external-facing outputs like API responses, golden files will likely 
simplify your tests and improve maintainability.

## Getting Started

To install Golden, simply run the following command:

```shell
go get github.com/tobbstr/golden
```

## Usage

Using Golden is straightforward. Here's a simple example:

Let say there is a function `GetPerson() person`, and that there is a golden file in the `testdata/get_person`
subfolder that contains the output of the `GetPerson()` in JSON-format.

The project layout:

```
.
├── testdata/
│   └── get_person/
│       └── happy_path.value.json
├── get_person.go
└── get_person_test.go
```

The `GetPerson()` function:

```go
package domain

import (
    "fmt"
)

type person struct {
    Name string
    HairColour string
}

func GetPerson() person {
    // skipped for brevity
    return &person{Name: name, HairColour: hairColour}
}
```

Then verifying that the `GetPerson()` function works as expected is as easy as:

```go
package domain

import (
    "fmt"
    "testing"

    "github.com/tobbstr/golden"
)

func TestGetPerson(t *testing.t) {
    // -------------------------------- Given --------------------------------
    want := "testdata/get_person/happy_path.value.json"

    // -------------------------------- When ---------------------------------
    got := GetPerson()

    // -------------------------------- Then ---------------------------------
    // Assert the return value
    golden.AssertJSON(t, want, got)
}
```

### GJSON Path Syntax

This library uses [GJSON](https://github.com/tidwall/gjson) path syntax for navigating JSON structures. 
GJSON paths provide a simple way to query and extract values from JSON documents.

#### Basic Path Syntax

- **Dot notation**: Navigate nested objects using dots
  - `"name"` - access top-level field
  - `"data.user.name"` - access nested fields

- **Array indexing**: Access array elements by index (zero-based)
  - `"users.0"` - first element
  - `"users.1.name"` - name field of second element

- **Wildcards**: Use `#` to match all array elements
  - `"users.#.age"` - all age fields in users array
  - `"data.items.#.id"` - all id fields in nested items array

#### Examples

Given this JSON:
```json
{
    "data": {
        "users": [
            {"name": "John", "age": 25},
            {"name": "Eliana", "age": 32}
        ]
    }
}
```

Common paths:
- `"data.users.0.name"` → `"John"`
- `"data.users.1.age"` → `32`
- `"data.users.#.name"` → `["John", "Eliana"]`

For more advanced syntax including modifiers, queries, and nested structures, see the 
[GJSON Path Syntax documentation](https://github.com/tidwall/gjson#path-syntax).

### AssertJSON vs RequireJSON

The library provides two main functions for comparing JSON:

- **`AssertJSON`**: If the comparison fails, marks the test as failed but continues execution. This allows you to 
  see multiple assertion failures in a single test run.
- **`RequireJSON`**: If the comparison fails, marks the test as failed and stops execution immediately. Use this 
  when subsequent assertions would be meaningless if this one fails.

Example:

```go
// This will continue even if it fails
golden.AssertJSON(t, want, got)

// This will stop the test immediately if it fails
golden.RequireJSON(t, want, got)
```

### Updating Golden Files

When you need to update your golden files with new expected values (for example, after intentionally changing your 
code's output), use the `UPDATE_GOLDENS` environment variable:

```shell
UPDATE_GOLDENS=1 go test ./...
```

This will update all golden files with the actual values from your tests instead of comparing them. This is 
particularly useful when:
- You've made intentional changes to your API responses
- You're setting up tests for the first time
- You need to update multiple golden files at once

**IMPORTANT**: Always review the changes to your golden files after updating them to ensure the new values are 
correct.

## Features

### Adding a field comment

When reviewing someone else's PR, and more specifically their golden files, it's not always easy to know what to
check when reviewing. In these situations it would be helpful if it were possible to have comments on fields that
are especially important to check. This library provides this functionality.

This adds a field comment.

```go
// NOTE! The file extension is .jsonc, since standard JSON does not support comments.
want := "testdata/my_golden_file.jsonc"
golden.AssertJSON(t, want, got, golden.WithFieldComments([]golden.FieldComment{
    {Path:"data.users.0.age", Comment: "Should be 25"},
    // ... add more field comments here if needed ...
}))
```

And this is what the resulting golden file looks like:

```json
{
    "data": {
        "users": [
            {
                "name": "John",
                "age": 25 // Should be 25
            },
            {
                "name": "Eliana",
                "age": 32
            }
        ]
    }
}
```

### Adding a file comment

For the same reasons as in the [Adding a field comment](#adding-a-field-comment) section, it is beneficial to
be able to add a comment at the top of the golden file.

This adds a file comment.

```go
// NOTE! The file extension is .jsonc, since standard JSON does not support comments.
want := "testdata/my_golden_file.jsonc"
golden.AssertJSON(t, want, got, golden.WithFileComment("This is my file comment"))
```

And this is what the resulting golden file looks like:

```json
/*
This is my file comment
*/

{
    "data": {
        "users": [
            {
                "name": "John",
                "age": 25
            },
            {
                "name": "Eliana",
                "age": 32
            }
        ]
    }
}
```

### Skipping non-deterministic values

When the output of a function call contains non-deterministic values such as generated ids that are different every
time you invoke it, then to make the golden files deterministic these fields' values must be made static. To
achieve this, do the following.

```go
// NOTE! The file extension is .jsonc, since standard JSON does not support comments.
want := "testdata/my_golden_file.jsonc"
golden.AssertJSON(t, want, got, golden.WithSkippedFields("data.users.#.age"))
```

And this is what the resulting golden file looks like:

```json
{
    "data": {
        "users": [
            {
                "name": "John",
                "age": "--* SKIPPED *--"
            },
            {
                "name": "Eliana",
                "age": "--* SKIPPED *--"
            }
        ]
    }
}
```

#### Keep Nulls

For fields that can be `null`, you can use a special `golden.Option` to distinguish between fields that are `null`
and those that have values.

For example, consider the following JSON where the `age` field is nullable:

```json
{
    "data": {
        "users": [
            {
                "name": "John",
                "age": 35
            },
            {
                "name": "Eliana",
                "age": null
            }
        ]
    }
}
```

Instead of replacing all values with "--* SKIPPED *--", we want to retain the "null" values and skip the non-null
ones. For John, we skip the `age` field, but for Eliana, we keep it.

Using the `KeepNull` option:

```go
golden.WithSkippedFields(
    golden.KeepNull("data.users.#.age"),
)
```

the resulting JSON will be:

```json
{
    "data": {
        "users": [
            {
                "name": "John",
                "age": "--* SKIPPED *--"
            },
            {
                "name": "Eliana",
                "age": null
            }
        ]
    }
}
```

### Time Validation

The library provides built-in options for validating timestamp fields in your JSON.

#### CheckNotZeroTime

Validates that a timestamp field is not zero. This is useful for ensuring that timestamps like `createdAt` or 
`updatedAt` have been set even though they are indeterministic.

```go
golden.AssertJSON(
    t, 
    want, 
    got,
    golden.CheckNotZeroTime("data.user.createdAt", time.RFC3339),
    golden.WithSkippedFields("data.user.createdAt"), // Skip the actual value since it's non-deterministic
)
```

This will:
1. First check that the timestamp is not zero (validation runs before modifications). NOTE: The order of the golden
   options is irrelevant.
2. Then replace the actual timestamp value with "--* SKIPPED *--" in the golden file

**Parameters:**
- `path`: GJSON path to the timestamp field (supports wildcards like `#` for array elements)
- `layout`: Time layout format (e.g., `time.RFC3339`, `time.RFC3339Nano`)

#### CheckEqualTimes

Validates that two timestamp fields have equal values. This is useful when you expect certain timestamps to match,
such as when a record is created and not yet updated.

```go
golden.AssertJSON(
    t,
    want,
    got,
    golden.CheckEqualTimes("data.user.createdAt", "data.user.updatedAt", time.RFC3339),
    golden.WithSkippedFields("data.user.createdAt", "data.user.updatedAt"),
)
```

This will:
1. First verify that both timestamps are equal
2. Then replace both timestamp values with "--* SKIPPED *--"

**Parameters:**
- `a`: GJSON path to the first timestamp
- `b`: GJSON path to the second timestamp
- `layout`: Time layout format (must be the same for both timestamps)

**NOTE:** `CheckEqualTimes` does not support wildcards in the GJSON paths.

### gRPC Status Error Support

The library automatically handles gRPC status errors by extracting their protobuf representation for JSON 
comparison. This is particularly useful when testing gRPC handlers or services that return status errors.

When you pass a gRPC status error to `AssertJSON` or `RequireJSON`, it will automatically convert it to its 
protobuf representation before marshaling to JSON, ensuring that all relevant fields are captured.

Example:

```go
import (
    "testing"
    
    "github.com/tobbstr/golden"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

func TestGRPCHandler(t *testing.T) {
    // -------------------------------- Given --------------------------------
    want := "testdata/grpc_error.json"

    // -------------------------------- When ---------------------------------
    err := status.Error(codes.Aborted, "data race detected")

    // -------------------------------- Then ---------------------------------
    golden.AssertJSON(t, want, err)
}
```

The resulting golden file will contain:

```json
{
    "code": 10,
    "message": "data race detected"
}
```

## Contributing

Contributions are welcome! If you find any issues or have suggestions for improvements, please feel free to open 
an issue or submit a pull request.

## License

Golden is licensed under the [MIT License](https://github.com/tobbstr/golden/blob/main/LICENSE).
