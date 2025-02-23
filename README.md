# Golden

Welcome to the Golden, a robust Go library designed to enhance your testing workflow by enabling easy management and
comparison of golden files. A golden file is a reference file used to validate the output of a system
against a known, correct result. It serves as a benchmark or "gold standard" to ensure that the system produces
the expected output consistently. This library is perfect for developers looking to streamline their testing process,
ensuring that their applications work as expected.

## Features

- **Golden File Management**: Easily manage your expected return values with golden files.
- **Comment Support in JSON**: Unique functionality that allows you to add comments to your JSON files, enhancing readability and maintainability, despite JSON's native lack of comment support. Use the `.jsonc` extension for error-free IDE integration.
- **Precise JSON Comparison**: Compare your expected JSON (from golden files) with the actual output of your code, with detailed reporting on discrepancies.
- **Flexible Configuration**: Customize your testing with various options, including the ability to mark fields as skipped, whose values are non-deterministic.

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

## Features

### Adding a field comment

When reviewing someone else's PR, and more specifically their golden files, it's not always easy to know what to
check when reviewing. In these situations it would be helpful if it were possible to have comments on fields that
are especially important to check. This library provides this functionality.

This adds a field comment.

```go
// NOTE! The file extension is .jsonc, since standard JSON does not support comments.
want := "testdata/my_golden_file.jsonc"
golden.AssertJSON(t, want, got, golden.FieldComments([]golden.FieldComment{
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
golden.AssertJSON(t, want, got, golden.FileComment("This is my file comment"))
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
golden.AssertJSON(t, want, got, golden.SkippedFields("data.users.#.age"))
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
golden.SkippedFields(
    golden.KeepNull("data.users.0.age"),
    golden.KeepNull("data.users.1.age"),
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

**NOTE:** The `golden.KeepNull` option does not support wildcards in the GJSON path. This means it can only target
one field at a time.

## Contributing

Contributions are welcome! If you find any issues or have suggestions for improvements, please feel free to open an issue or submit a pull request.

## License

Golden is licensed under the [MIT License](https://github.com/tobbstr/golden/blob/main/LICENSE).
