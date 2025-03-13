package golden

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAssertJSON_UpdateFlag(t *testing.T) {
	type args struct {
		t       *testing.T
		want    string
		got     any
		options []Option
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
		// failure is true when the test case should fail
		failure bool
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
					want: "testdata/assert_json_update_flag/overwrites.json",
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
			name: "no-op when update flag is set to false",
			given: given{
				args: args{
					want: "testdata/assert_json_update_flag/no-op.json",
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			/* ---------------------------------- Given --------------------------------- */
			if tt.given.update {
				tt.given.args.options = append(tt.given.args.options, UpdateGoldenFiles())
			}
			initialGoldenFile := readFile(t, tt.given.args.want)

			// Restore the initial golden file if it will be updated
			if tt.want.goldenFileUpdated {
				defer writeFile(t, tt.given.args.want, initialGoldenFile)
			}

			tt.given.args.t = t

			/* ---------------------------------- When ---------------------------------- */
			AssertJSON(tt.given.args.t, tt.given.args.want, tt.given.args.got, tt.given.args.options...)

			/* ---------------------------------- Then ---------------------------------- */
			got := readFile(t, tt.given.args.want)

			if tt.want.goldenFileUpdated {
				require.NotEqual(t, initialGoldenFile, got, "golden file should be updated")
			}

			if tt.want.failure {
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
}

func TestAssertJSON_Failure(t *testing.T) {
	type args struct {
		t       *testing.T
		want    string
		got     any
		options []Option
	}
	type given struct {
		args args
	}
	type test struct {
		name  string
		given given
	}
	tests := []test{
		{
			name: "fails when the golden file's content is different from the got JSON",
			given: given{
				args: args{
					want: "testdata/assert_json_failure/json_different.json",
					got: map[string]interface{}{
						"name": "John",
						"age":  30,
						"colour": map[string]interface{}{
							"hair": "black",
							"eyes": "brown",
						},
					},
				},
			},
		},
		{
			name: "test fails when commenting on non-existent field",
			given: given{
				args: args{
					want: "testdata/assert_json_failure/empty.json",
					got: map[string]interface{}{
						"name": "John",
					},
					options: []Option{WithFieldComments([]FieldComment{{Path: "age", Comment: "This is a file comment"}})},
				},
			},
		},
		{
			name: "test fails when skipping non-existent field",
			given: given{
				args: args{
					want: "testdata/assert_json_failure/empty.json",
					got: map[string]interface{}{
						"name": "John",
					},
					options: []Option{WithSkippedFields("age")},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			/* ---------------------------------- Given --------------------------------- */
			initialGoldenFile := readFile(t, tt.given.args.want)
			defer writeFile(t, tt.given.args.want, initialGoldenFile)

			tt.given.args.t = &testing.T{} // test result recorder

			/* ---------------------------------- When ---------------------------------- */
			AssertJSON(tt.given.args.t, tt.given.args.want, tt.given.args.got, tt.given.args.options...)

			/* ---------------------------------- Then ---------------------------------- */
			// Assert that the test failed
			require.True(t, tt.given.args.t.Failed(), "want test failed got passed")
		})
	}
}

func TestAssertJSON(t *testing.T) {
	type args struct {
		t       *testing.T
		want    string
		got     any
		options []Option
	}
	type given struct {
		args args
	}
	type test struct {
		name  string
		given given
	}
	tests := []test{
		{
			name: "passes when the golden file's content is equal to the got JSON",
			given: given{
				args: args{
					want: "testdata/assert_json/same_content.json",
					got:  map[string]interface{}{"name": "John", "age": 30},
				},
			},
		},
		{
			name: "skips fields when map",
			given: given{
				args: args{
					want: "testdata/assert_json/skips_field_map.json",
					got: map[string]interface{}{
						"name": "John",
						"age":  30,
						"colour": map[string]interface{}{
							"hair": "black",
							"eyes": "brown",
						},
					},
					options: []Option{WithSkippedFields("colour.hair", "colour.eyes")},
				},
			},
		},
		{
			name: "skips fields when struct",
			given: given{
				args: func() args {
					type hair struct {
						Colour string `json:"colour"`
					}
					type sibling struct {
						Hair hair `json:"hair"`
					}
					type person struct {
						Name    string  `json:"name"`
						Age     int     `json:"age"`
						Sibling sibling `json:"sibling"`
					}

					return args{
						want: "testdata/assert_json/skips_field_struct.jsonc",
						got: person{
							Name: "John",
							Age:  30,
							Sibling: sibling{
								Hair: hair{Colour: "brown"},
							},
						},
						options: []Option{WithSkippedFields("sibling.hair.colour")},
					}
				}(),
			},
		},
		{
			name: "skips fields when slice",
			given: given{
				args: func() args {
					type hair struct {
						Colour string `json:"colour"`
					}
					type sibling struct {
						Hair hair `json:"hair"`
					}
					type person struct {
						Name     string    `json:"name"`
						Age      int       `json:"age"`
						Siblings []sibling `json:"siblings"`
					}

					return args{
						want: "testdata/assert_json/skips_fields_slice.jsonc",
						got: person{
							Name: "John",
							Age:  30,
							Siblings: []sibling{
								{Hair: hair{Colour: "black"}},
								{Hair: hair{Colour: "brown"}},
							},
						},
						options: []Option{WithSkippedFields("siblings")},
					}
				}(),
			},
		},
		{
			name: "skips field when slice",
			given: given{
				args: func() args {
					type hair struct {
						Colour string `json:"colour"`
					}
					type sibling struct {
						Hair hair `json:"hair"`
					}
					type person struct {
						Name     string    `json:"name"`
						Age      int       `json:"age"`
						Siblings []sibling `json:"siblings"`
					}

					return args{
						want: "testdata/assert_json/skips_field_slice.jsonc",
						got: person{
							Name: "John",
							Age:  30,
							Siblings: []sibling{
								{Hair: hair{Colour: "black"}},
								{Hair: hair{Colour: "brown"}},
							},
						},
						options: []Option{WithSkippedFields("siblings.1.hair.colour")},
					}
				}(),
			},
		},
		{
			name: "skips fields when array",
			given: given{
				args: func() args {
					type hair struct {
						Colour string `json:"colour"`
					}
					type sibling struct {
						Hair hair `json:"hair"`
					}
					type person struct {
						Name     string     `json:"name"`
						Age      int        `json:"age"`
						Siblings [2]sibling `json:"siblings"`
					}

					return args{
						want: "testdata/assert_json/skips_field_array.jsonc",
						got: person{
							Name: "John",
							Age:  30,
							Siblings: [2]sibling{
								{Hair: hair{Colour: "black"}},
								{Hair: hair{Colour: "brown"}},
							},
						},
						options: []Option{WithSkippedFields("siblings.1.hair.colour")},
					}
				}(),
			},
		},
		{
			// When a nil pointer is chosen to be skipped, then the expected output for that field is "null".
			// In other words, since a nil pointer field is unmarshalled into "null" by the encoding/json package,
			// this function just leaves the field as is.
			name: "skips fields when nil pointer",
			given: given{
				args: func() args {
					type hair struct {
						Colour string `json:"colour"`
					}
					type sibling struct {
						Hair hair `json:"hair"`
					}
					type person struct {
						Name    string   `json:"name"`
						Age     int      `json:"age"`
						Sibling *sibling `json:"sibling"`
					}

					return args{
						want: "testdata/assert_json/skips_field_nil_pointer.jsonc",
						got: person{
							Name:    "John",
							Age:     30,
							Sibling: nil,
						},
						options: []Option{WithSkippedFields("sibling")},
					}
				}(),
			},
		},
		{
			// When a non-nil pointer is chosen to be skipped, then the expected output for that field is "SKIPPED".
			name: "skips fields when non-nil pointer",
			given: given{
				args: func() args {
					type hair struct {
						Colour string `json:"colour"`
					}
					type sibling struct {
						Hair hair `json:"hair"`
					}
					type person struct {
						Name    string   `json:"name"`
						Age     int      `json:"age"`
						Sibling *sibling `json:"sibling"`
					}

					return args{
						want: "testdata/assert_json/skips_field_non-nil_pointer.jsonc",
						got: person{
							Name:    "John",
							Age:     30,
							Sibling: &sibling{Hair: hair{Colour: "brown"}},
						},
						options: []Option{WithSkippedFields("sibling")},
					}
				}(),
			},
		},
		{
			// When KeepNull is used on a nil pointer, then the expected output for that field is "null".
			name: "keeps null when KeepNull on nil pointer",
			given: given{
				args: func() args {
					type hair struct {
						Colour string `json:"colour"`
					}
					type sibling struct {
						Hair hair `json:"hair"`
					}
					type person struct {
						Name    string   `json:"name"`
						Age     int      `json:"age"`
						Sibling *sibling `json:"sibling"`
					}

					return args{
						want: "testdata/assert_json/skips_field_keep_null_on_nil_pointer.json",
						got: person{
							Name:    "John",
							Age:     30,
							Sibling: nil,
						},
						options: []Option{WithSkippedFields(KeepNull("sibling"))},
					}
				}(),
			},
		},
		{
			// When KeepNull is used on a non-nil pointer, then the expected output for that field is "SKIPPED".
			name: "skips field when KeepNull on non-nil pointer",
			given: given{
				args: func() args {
					type hair struct {
						Colour string `json:"colour"`
					}
					type sibling struct {
						Hair hair `json:"hair"`
					}
					type person struct {
						Name    string   `json:"name"`
						Age     int      `json:"age"`
						Sibling *sibling `json:"sibling"`
					}

					return args{
						want: "testdata/assert_json/skips_field_keep_null_on_non-nil_pointer.json",
						got: person{
							Name:    "John",
							Age:     30,
							Sibling: &sibling{Hair: hair{Colour: "brown"}},
						},
						options: []Option{WithSkippedFields(KeepNull("sibling"))},
					}
				}(),
			},
		},
		{
			name: "skips multiple fields",
			given: given{
				args: func() args {
					type hair struct {
						Colour string `json:"colour"`
					}
					type sibling struct {
						Hair hair      `json:"hair"`
						Born time.Time `json:"born"`
					}
					type person struct {
						Name     string    `json:"name"`
						Age      int       `json:"age"`
						Siblings []sibling `json:"siblings"`
					}

					return args{
						want: "testdata/assert_json/skips_multiple_fields.json",
						got: person{
							Name: "John",
							Age:  30,
							Siblings: []sibling{
								{Hair: hair{Colour: "brown"}, Born: time.Now().Add(-5 * time.Minute)},
								{Hair: hair{Colour: "blonde"}, Born: time.Now()},
							},
						},
						options: []Option{WithSkippedFields("siblings.#.born")},
					}
				}(),
			},
		},
		{
			name: "skips one and keeps null in another field when multi-selecting fields",
			given: given{
				args: func() args {
					type hair struct {
						Colour string `json:"colour"`
					}
					type sibling struct {
						Hair hair       `json:"hair"`
						Born *time.Time `json:"born"`
					}
					type person struct {
						Name     string    `json:"name"`
						Age      int       `json:"age"`
						Siblings []sibling `json:"siblings"`
					}

					now := time.Now()

					return args{
						want: "testdata/assert_json/skips_multiple_fields_keeps_one.json",
						got: person{
							Name: "John",
							Age:  30,
							Siblings: []sibling{
								{Hair: hair{Colour: "brown"}, Born: nil},
								{Hair: hair{Colour: "blonde"}, Born: &now},
							},
						},
						options: []Option{WithSkippedFields(KeepNull("siblings.#.born"))},
					}
				}(),
			},
		},
		{
			name: "adds field comments",
			given: given{
				args: args{
					want: "testdata/assert_json/adds_field_comments.jsonc",
					got: map[string]interface{}{
						"name": "John",
						"age":  30,
						"colour": map[string]interface{}{
							"hair": "black",
							"eyes": "brown",
						},
					},
					options: []Option{WithFieldComments([]FieldComment{
						{Path: "colour.hair", Comment: "Should be black. Since lorem ipsum dolor sit amet, consectetur adipiscing elit."},
						{Path: "colour.eyes", Comment: "Should be brown"},
					})},
				},
			},
		},
		{
			name: "adds file comment",
			given: given{
				args: args{
					want: "testdata/assert_json/adds_file_comment.jsonc",
					got: map[string]interface{}{
						"name": "John",
						"age":  30,
						"colour": map[string]interface{}{
							"hair": "black",
							"eyes": "brown",
						},
					},
					options: []Option{WithFileComment("This is a file comment")},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			/* ---------------------------------- Given --------------------------------- */
			tt.given.args.t = &testing.T{} // test result recorder

			/* ---------------------------------- When ---------------------------------- */
			AssertJSON(tt.given.args.t, tt.given.args.want, tt.given.args.got, tt.given.args.options...)

			/* ---------------------------------- Then ---------------------------------- */
			require.False(t, tt.given.args.t.Failed()) // All test cases should pass, failures are tested in TestAssertJSON_Failure
		})
	}
}

func TestWithNotZeroTime(t *testing.T) {
	type args struct {
		path   string
		layout string
	}
	type given struct {
		args args
		json string
		t    *testing.T // should not be initialized in the test cases
	}
	type want struct {
		failed bool // true if the test should fail, false if it should pass
	}
	type test struct {
		name  string
		given given
		want  want
	}
	tests := []test{
		{
			name: "passes when the time is not zero",
			given: given{
				args: args{
					path:   "data.user.updatedAt",
					layout: time.RFC3339,
				},
				json: "testdata/not_zero_time/passes_when_time_not_zero.json",
			},
			want: want{failed: false},
		},
		{
			name: "fails when the time is zero",
			given: given{
				args: args{
					path:   "data.user.updatedAt",
					layout: time.RFC3339,
				},
				json: "testdata/not_zero_time/fails_when_time_zero.json",
			},
			want: want{failed: true},
		},
		{
			name: "fails when the path does not exist",
			given: given{
				args: args{
					path:   "data.user.updatedAt",
					layout: time.RFC3339,
				},
				json: "testdata/not_zero_time/fails_when_path_does_not_exist.json",
			},
			want: want{failed: true},
		},
		{
			name: "fails when the value is not a string",
			given: given{
				args: args{
					path:   "data.user.age",
					layout: time.RFC3339,
				},
				json: "testdata/not_zero_time/fails_when_value_not_string.json",
			},
			want: want{failed: true},
		},
		{
			name: "fails when the date format is incorrect",
			given: given{
				args: args{
					path:   "data.user.updatedAt",
					layout: "2006-01-02", // This is not a valid layout which causes the test to fail
				},
				json: "testdata/not_zero_time/passes_when_time_not_zero.json", // This test requires a valid JSON file with a path that exists. That's why we use the same file as the first test case.
			},
			want: want{failed: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			/* ---------------------------------- Given --------------------------------- */
			require := require.New(t)
			tt.given.t = &testing.T{} // test result recorder
			fileBytes := readFile(t, tt.given.json)
			g := &golden{result: fileBytes}

			/* ---------------------------------- When ---------------------------------- */
			WithNotZeroTime(tt.given.args.path, tt.given.args.layout)(tt.given.t, false, g, "")

			/* ---------------------------------- Then ---------------------------------- */
			// Assert the test result
			require.Equal(tt.want.failed, tt.given.t.Failed())
		})
	}
}

func TestWithEqualTimes(t *testing.T) {
	type args struct {
		a      string
		b      string
		layout string
	}
	type given struct {
		args args
		json string
		t    *testing.T // should not be initialized in the test cases
	}
	type want struct {
		failed bool // true if the test should fail, false if it should pass
	}
	type test struct {
		name  string
		given given
		want  want
	}
	tests := []test{
		{
			name: "passes when the times are equal",
			given: given{
				args: args{
					a:      "data.user.createdAt",
					b:      "data.user.updatedAt",
					layout: time.RFC3339,
				},
				json: "testdata/equal_times/passes_when_times_equal.json",
			},
			want: want{failed: false},
		},
		{
			name: "fails when the times are not equal",
			given: given{
				args: args{
					a:      "data.user.createdAt",
					b:      "data.user.updatedAt",
					layout: time.RFC3339,
				},
				json: "testdata/equal_times/fails_when_times_not_equal.json",
			},
			want: want{failed: true},
		},
		{
			name: "fails when the first path does not exist",
			given: given{
				args: args{
					a:      "data.user.createdAt",
					b:      "data.user.updatedAt",
					layout: time.RFC3339,
				},
				json: "testdata/equal_times/fails_when_first_path_does_not_exist.json",
			},
			want: want{failed: true},
		},
		{
			name: "fails when the second path does not exist",
			given: given{
				args: args{
					a:      "data.user.createdAt",
					b:      "data.user.updatedAt",
					layout: time.RFC3339,
				},
				json: "testdata/equal_times/fails_when_second_path_does_not_exist.json",
			},
			want: want{failed: true},
		},
		{
			name: "fails when the first value is not a string",
			given: given{
				args: args{
					a:      "data.user.age",
					b:      "data.user.updatedAt",
					layout: time.RFC3339,
				},
				json: "testdata/equal_times/fails_when_first_value_not_string.json",
			},
			want: want{failed: true},
		},
		{
			name: "fails when the second value is not a string",
			given: given{
				args: args{
					a:      "data.user.createdAt",
					b:      "data.user.age",
					layout: time.RFC3339,
				},
				json: "testdata/equal_times/fails_when_second_value_not_string.json",
			},
			want: want{failed: true},
		},
		{
			name: "fails when the date format is incorrect",
			given: given{
				args: args{
					a:      "data.user.createdAt",
					b:      "data.user.updatedAt",
					layout: "2006-01-02", // This is not a valid layout which causes the test to fail
				},
				json: "testdata/equal_times/passes_when_times_equal.json", // This test requires a valid JSON file with paths that exist. That's why we use the same file as the first test case.
			},
			want: want{failed: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			/* ---------------------------------- Given --------------------------------- */
			require := require.New(t)
			tt.given.t = &testing.T{} // test result recorder
			fileBytes := readFile(t, tt.given.json)
			g := &golden{result: fileBytes}

			/* ---------------------------------- When ---------------------------------- */
			WithEqualTimes(tt.given.args.a, tt.given.args.b, tt.given.args.layout)(tt.given.t, false, g, "")

			/* ---------------------------------- Then ---------------------------------- */
			// Assert the test result
			require.Equal(tt.want.failed, tt.given.t.Failed())
		})
	}
}
