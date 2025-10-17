package gjson

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExpandPath(t *testing.T) {
	// Test Types
	type (
		args struct { // arguments to the function under test
			json []byte
			path string
		}
		fixture struct { // shared setup for all test cases
			json string
		}
		given struct { // test-case-specific setup
			args args
		}
		want struct { // expected results
			paths []string
		}
	)

	// Test Variables
	shared := fixture{
		json: `{
			"name": {"first": "Tom", "last": "Anderson"},
			"age":37,
			"children": ["Sara","Alex","Jack"],
			"fav.movie": "Deer Hunter",
			"friends": [
				{"first": "Dale", "last": "Murphy", "age": 44, "nets": ["ig", "fb", "tw"]},
				{"first": "Roger", "last": "Craig", "age": 68, "nets": ["fb", "tw"]},
				{"first": "Jane", "last": "Murphy", "age": 47, "nets": ["ig", "tw"]}
			],
			"vals": [
				{ "a": 1, "b": "data" },
				{ "a": 2, "b": true },
				{ "a": 3, "b": false },
				{ "a": 4, "b": "0" },
				{ "a": 5, "b": 0 },
				{ "a": 6, "b": "1" },
				{ "a": 7, "b": 1 },
				{ "a": 8, "b": "true" },
				{ "a": 9, "b": false },
				{ "a": 10, "b": null },
				{ "a": 11 }
			],
			"families": [
				{
					"surname": "Smith",
					"members": [
						{
							"name": "John",
							"age": 45,
							"hobbies": [
								{"name": "reading", "locations": ["library", "home", "cafe"]},
								{"name": "cycling", "locations": ["park", "trail"]}
							]
						},
						{
							"name": "Jane",
							"age": 42,
							"hobbies": [
								{"name": "cooking", "locations": ["kitchen", "restaurant"]},
								{"name": "gardening", "locations": ["backyard", "greenhouse", "community garden"]}
							]
						}
					]
				},
				{
					"surname": "Johnson",
					"members": [
						{
							"name": "Mike",
							"age": 38,
							"hobbies": [
								{"name": "fishing", "locations": ["lake", "river", "ocean"]},
								{"name": "photography", "locations": ["studio", "nature", "city"]}
							]
						},
						{
							"name": "Sarah",
							"age": 35,
							"hobbies": [
								{"name": "yoga", "locations": ["studio", "home", "beach"]},
								{"name": "traveling", "locations": ["mountains", "cities", "beaches", "forests"]}
							]
						}
					]
				},
				{
					"surname": "Williams",
					"members": [
						{
							"name": "David",
							"age": 50,
							"hobbies": [
								{"name": "woodworking", "locations": ["garage", "workshop"]},
								{"name": "hiking", "locations": ["mountains", "trails", "national parks"]}
							]
						}
					]
				}
			]
		}`,
	}

	// Test Cases
	tests := []struct {
		name  string
		given given
		want  want
	}{
		// Basic path navigation
		{
			name: "basic object field access",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "name.first",
				},
			},
			want: want{
				paths: []string{"name.first"},
			},
		},
		{
			name: "nested object field access",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "name.last",
				},
			},
			want: want{
				paths: []string{"name.last"},
			},
		},
		{
			name: "root level field access",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "age",
				},
			},
			want: want{
				paths: []string{"age"},
			},
		},
		{
			name: "array field access",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "children",
				},
			},
			want: want{
				paths: []string{"children"},
			},
		},
		{
			name: "array index access",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "children.0",
				},
			},
			want: want{
				paths: []string{"children.0"},
			},
		},
		{
			name: "array index access second element",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "children.1",
				},
			},
			want: want{
				paths: []string{"children.1"},
			},
		},
		{
			name: "nested array object access",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends.1",
				},
			},
			want: want{
				paths: []string{"friends.1"},
			},
		},
		{
			name: "deeply nested object field",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends.1.first",
				},
			},
			want: want{
				paths: []string{"friends.1.first"},
			},
		},

		// Wildcard tests
		{
			name: "wildcard asterisk matching multiple characters",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "child*.2",
				},
			},
			want: want{
				paths: []string{"children.2"},
			},
		},
		{
			name: "wildcard question mark matching single character",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "c?ildren.0",
				},
			},
			want: want{
				paths: []string{"children.0"},
			},
		},

		// Escape character tests
		{
			name: "escaped dot in field name",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "fav\\.movie",
				},
			},
			want: want{
				paths: []string{"fav\\.movie"},
			},
		},
		{
			name: "escaped asterisk wildcard",
			given: given{
				args: args{
					json: []byte(`{"field*name": "value", "fieldname": "other"}`),
					path: "field\\*name",
				},
			},
			want: want{
				paths: []string{"field\\*name"},
			},
		},
		{
			name: "escaped question mark wildcard",
			given: given{
				args: args{
					json: []byte(`{"field?name": "value", "fieldname": "other"}`),
					path: "field\\?name",
				},
			},
			want: want{
				paths: []string{"field\\?name"},
			},
		},
		{
			name: "escaped pipe separator",
			given: given{
				args: args{
					json: []byte(`{"field|name": "value"}`),
					path: "field\\|name",
				},
			},
			want: want{
				paths: []string{"field\\|name"},
			},
		},
		{
			name: "escaped hash character",
			given: given{
				args: args{
					json: []byte(`{"field#name": "value"}`),
					path: "field\\#name",
				},
			},
			want: want{
				paths: []string{"field\\#name"},
			},
		},
		{
			name: "escaped at sign modifier character",
			given: given{
				args: args{
					json: []byte(`{"field@name": "value"}`),
					path: "field\\@name",
				},
			},
			want: want{
				paths: []string{"field\\@name"},
			},
		},
		{
			name: "escaped exclamation literal character",
			given: given{
				args: args{
					json: []byte(`{"field!name": "value"}`),
					path: "field\\!name",
				},
			},
			want: want{
				paths: []string{"field\\!name"},
			},
		},

		// Array operations with # - Basic Level
		{
			name: "array length using # character",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends.#",
				},
			},
			want: want{
				paths: []string{"friends.#"},
			},
		},
		{
			name: "array length using # on children",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "children.#",
				},
			},
			want: want{
				paths: []string{"children.#"},
			},
		},
		{
			name: "array length using # on nested families",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "families.#",
				},
			},
			want: want{
				paths: []string{"families.#"},
			},
		},
		{
			name: "single level array map operation with #",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends.#.age",
				},
			},
			want: want{
				paths: []string{"friends.0.age", "friends.1.age", "friends.2.age"},
			},
		},
		{
			name: "single level array map operation on families",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "families.#.surname",
				},
			},
			want: want{
				paths: []string{"families.0.surname", "families.1.surname", "families.2.surname"},
			},
		},

		// Array operations with # - Two Levels Deep
		{
			name: "two level nested array length",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "families.#.members.#",
				},
			},
			want: want{
				paths: []string{"families.0.members.#", "families.1.members.#", "families.2.members.#"},
			},
		},
		{
			name: "two level nested array map operation",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "families.#.members.#.name",
				},
			},
			want: want{
				paths: []string{"families.0.members.0.name", "families.0.members.1.name", "families.1.members.0.name", "families.1.members.1.name", "families.2.members.0.name"},
			},
		},
		{
			name: "two level nested array map with different field",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "families.#.members.#.age",
				},
			},
			want: want{
				paths: []string{"families.0.members.0.age", "families.0.members.1.age", "families.1.members.0.age", "families.1.members.1.age", "families.2.members.0.age"},
			},
		},

		// Array operations with # - Three Levels Deep
		{
			name: "three level nested array length",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "families.#.members.#.hobbies.#",
				},
			},
			want: want{
				paths: []string{"families.0.members.0.hobbies.#", "families.0.members.1.hobbies.#", "families.1.members.0.hobbies.#", "families.1.members.1.hobbies.#", "families.2.members.0.hobbies.#"},
			},
		},
		{
			name: "three level nested array map operation",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "families.#.members.#.hobbies.#.name",
				},
			},
			want: want{
				paths: []string{
					"families.0.members.0.hobbies.0.name",
					"families.0.members.0.hobbies.1.name",
					"families.0.members.1.hobbies.0.name",
					"families.0.members.1.hobbies.1.name",
					"families.1.members.0.hobbies.0.name",
					"families.1.members.0.hobbies.1.name",
					"families.1.members.1.hobbies.0.name",
					"families.1.members.1.hobbies.1.name",
					"families.2.members.0.hobbies.0.name",
					"families.2.members.0.hobbies.1.name",
				},
			},
		},

		// Mixed array operations with # and specific indices
		{
			name: "mixed # and specific index access",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "families.0.members.#.hobbies.#.name",
				},
			},
			want: want{
				paths: []string{
					"families.0.members.0.hobbies.0.name",
					"families.0.members.0.hobbies.1.name",
					"families.0.members.1.hobbies.0.name",
					"families.0.members.1.hobbies.1.name",
				},
			},
		},
		{
			name: "mixed specific index and # access",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "families.#.members.0.hobbies.#.name",
				},
			},
			want: want{
				paths: []string{
					"families.0.members.0.hobbies.0.name",
					"families.0.members.0.hobbies.1.name",
					"families.1.members.0.hobbies.0.name",
					"families.1.members.0.hobbies.1.name",
					"families.2.members.0.hobbies.0.name",
					"families.2.members.0.hobbies.1.name",
				},
			},
		},
		{
			name: "complex mixed access pattern",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "families.#.members.1.hobbies.0.locations.#",
				},
			},
			want: want{
				paths: []string{
					"families.0.members.1.hobbies.0.locations.0",
					"families.0.members.1.hobbies.0.locations.1",
					"families.1.members.1.hobbies.0.locations.0",
					"families.1.members.1.hobbies.0.locations.1",
					"families.1.members.1.hobbies.0.locations.2",
				},
			},
		},

		// Query tests with #(...)
		{
			name: "query for exact match equality",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends.#(last==\"Murphy\")#.first",
				},
			},
			want: want{
				paths: []string{"friends.0.first", "friends.2.first"},
			},
		},
		{
			name: "query with greater than comparison",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends.#(age>45)#.last",
				},
			},
			want: want{
				paths: []string{"friends.1.last", "friends.2.last"},
			},
		},
		{
			name: "query with pattern matching using %",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends.#(first%\"D*\").last",
				},
			},
			want: want{
				paths: []string{"friends.0.last"},
			},
		},
		{
			name: "query with negative pattern matching using !%",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends.#(first!%\"D*\").last",
				},
			},
			want: want{
				paths: []string{"friends.1.last", "friends.2.last"},
			},
		},
		{
			name: "query array values with pattern matching",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "children.#(!%\"*a*\")",
				},
			},
			want: want{
				paths: []string{"children.1"},
			},
		},
		{
			name: "query array values with all matches pattern",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "children.#(%\"*a*\")#",
				},
			},
			want: want{
				paths: []string{"children.0", "children.2"},
			},
		},
		{
			name: "nested query with array matching",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends.#(nets.#(==\"fb\"))#.first",
				},
			},
			want: want{
				paths: []string{"friends.0.first", "friends.1.first"},
			},
		},
		{
			name: "query with not equal operator",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends.#(last!=\"Murphy\")#.first",
				},
			},
			want: want{
				paths: []string{"friends.1.first"},
			},
		},
		{
			name: "query with less than operator",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends.#(age<50)#.first",
				},
			},
			want: want{
				paths: []string{"friends.0.first", "friends.2.first"},
			},
		},
		{
			name: "query with less than or equal operator",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends.#(age<=47)#.first",
				},
			},
			want: want{
				paths: []string{"friends.0.first", "friends.2.first"},
			},
		},
		{
			name: "query with greater than or equal operator",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends.#(age>=47)#.first",
				},
			},
			want: want{
				paths: []string{"friends.1.first", "friends.2.first"},
			},
		},
		{
			name: "query with first match only (no # suffix)",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends.#(last==\"Murphy\").first",
				},
			},
			want: want{
				paths: []string{"friends.0.first"},
			},
		},
		{
			name: "legacy bracket query syntax for backwards compatibility",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends.#[last==\"Murphy\"]#.first",
				},
			},
			want: want{
				paths: []string{"friends.0.first", "friends.2.first"},
			},
		},

		{
			name: "deeply nested query with location pattern matching",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "families.#.members.#.hobbies.#.locations.#(==\"home\")",
				},
			},
			want: want{
				paths: []string{
					"families.0.members.0.hobbies.0.locations.1",
					"families.1.members.1.hobbies.0.locations.1",
				},
			},
		},

		// Tilde operator tests
		{
			name: "tilde true operator for truthy values",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "vals.#(b==~true)#.a",
				},
			},
			want: want{
				paths: []string{"vals.1.a", "vals.5.a", "vals.6.a", "vals.7.a"},
			},
		},
		{
			name: "tilde false operator for falsy values",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "vals.#(b==~false)#.a",
				},
			},
			want: want{
				paths: []string{"vals.2.a", "vals.3.a", "vals.4.a", "vals.8.a", "vals.9.a", "vals.10.a"},
			},
		},
		{
			name: "tilde null operator for null and non-existent values",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "vals.#(b==~null)#.a",
				},
			},
			want: want{
				paths: []string{"vals.9.a", "vals.10.a"},
			},
		},
		{
			name: "tilde asterisk operator for existing values",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "vals.#(b==~*)#.a",
				},
			},
			want: want{
				paths: []string{"vals.0.a", "vals.1.a", "vals.2.a", "vals.3.a", "vals.4.a", "vals.5.a", "vals.6.a", "vals.7.a", "vals.8.a", "vals.9.a"},
			},
		},
		{
			name: "tilde asterisk negation for non-existent values",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "vals.#(b!=~*)#.a",
				},
			},
			want: want{
				paths: []string{"vals.10.a"},
			},
		},

		// Dot vs Pipe separator tests
		{
			name: "dot separator basic usage",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends.0.first",
				},
			},
			want: want{
				paths: []string{"friends.0.first"},
			},
		},
		{
			name: "pipe separator basic usage",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends|0.first",
				},
			},
			want: want{
				paths: []string{"friends.0.first"},
			},
		},
		{
			name: "mixed dot and pipe separators",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends.0|first",
				},
			},
			want: want{
				paths: []string{"friends.0.first"},
			},
		},
		{
			name: "all pipe separators",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends|0|first",
				},
			},
			want: want{
				paths: []string{"friends.0.first"},
			},
		},
		{
			name: "pipe with array length",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends|#",
				},
			},
			want: want{
				paths: []string{"friends.#"},
			},
		},
		{
			name: "dot vs pipe with query results - dot processes each element",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends.#(last==\"Murphy\")#.first",
				},
			},
			want: want{
				paths: []string{"friends.0.first", "friends.2.first"},
			},
		},

		// Modifier tests
		{
			name: "reverse modifier on array",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "children.@reverse",
				},
			},
			want: want{
				paths: []string{"children.@reverse"},
			},
		},
		{
			name: "reverse modifier with index access",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "children.@reverse.0",
				},
			},
			want: want{
				paths: []string{"children.@reverse.0"},
			},
		},
		{
			name: "this modifier",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "@this",
				},
			},
			want: want{
				paths: []string{"@this"},
			},
		},
		{
			name: "keys modifier on object",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "name.@keys",
				},
			},
			want: want{
				paths: []string{"name.@keys"},
			},
		},
		{
			name: "values modifier on object",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "name.@values",
				},
			},
			want: want{
				paths: []string{"name.@values"},
			},
		},
		{
			name: "flatten modifier on nested arrays",
			given: given{
				args: args{
					json: []byte(`{"nested": [[1,2],[3,4],[5,6]]}`),
					path: "nested.@flatten",
				},
			},
			want: want{
				paths: []string{"nested.@flatten"},
			},
		},
		{
			name: "ugly modifier to remove whitespace",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "name.@ugly",
				},
			},
			want: want{
				paths: []string{"name.@ugly"},
			},
		},
		{
			name: "pretty modifier to format JSON",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "name.@pretty",
				},
			},
			want: want{
				paths: []string{"name.@pretty"},
			},
		},
		{
			name: "pretty modifier with arguments",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "name.@pretty:{\"sortKeys\":true}",
				},
			},
			want: want{
				paths: []string{"name.@pretty:{\"sortKeys\":true}"},
			},
		},
		{
			name: "valid modifier to validate JSON",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "@valid",
				},
			},
			want: want{
				paths: []string{"@valid"},
			},
		},
		{
			name: "join modifier to join objects",
			given: given{
				args: args{
					json: []byte(`{"objs": [{"a":1}, {"b":2}, {"c":3}]}`),
					path: "objs.@join",
				},
			},
			want: want{
				paths: []string{"objs.@join"},
			},
		},
		{
			name: "tostr modifier to convert to string",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "age.@tostr",
				},
			},
			want: want{
				paths: []string{"age.@tostr"},
			},
		},
		{
			name: "fromstr modifier to parse from string",
			given: given{
				args: args{
					json: []byte(`{"jsonStr": "{\"key\":\"value\"}"}`),
					path: "jsonStr.@fromstr",
				},
			},
			want: want{
				paths: []string{"jsonStr.@fromstr"},
			},
		},
		{
			name: "group modifier to group arrays",
			given: given{
				args: args{
					json: []byte(`{"items": [{"type":"A","val":1}, {"type":"B","val":2}, {"type":"A","val":3}]}`),
					path: "items.@group",
				},
			},
			want: want{
				paths: []string{"items.@group"},
			},
		},
		{
			name: "dig modifier to search for value",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "@dig:first",
				},
			},
			want: want{
				paths: []string{"@dig:first"},
			},
		},
		{
			name: "custom case modifier with upper argument",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "children.@case:upper",
				},
			},
			want: want{
				paths: []string{"children.@case:upper"},
			},
		},
		{
			name: "custom case modifier with lower argument",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "children.@case:lower",
				},
			},
			want: want{
				paths: []string{"children.@case:lower"},
			},
		},
		{
			name: "chained modifiers",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "children.@case:lower.@reverse",
				},
			},
			want: want{
				paths: []string{"children.@case:lower.@reverse"},
			},
		},

		// Multipath tests
		{
			name: "multipath array creation",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "[name.first,age]",
				},
			},
			want: want{
				paths: []string{"name.first", "age"},
			},
		},
		{
			name: "multipath object creation",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "{name.first,age}",
				},
			},
			want: want{
				paths: []string{"name.first", "age"},
			},
		},
		{
			name: "multipath object with custom key",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "{name.first,age,\"the_murphys\":friends.#(last==\"Murphy\")#.first}",
				},
			},
			want: want{
				paths: []string{"name.first", "age", "friends.0.first", "friends.2.first"},
			},
		},
		{
			name: "multipath array with nested family data",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "[families.#.surname,families.#.members.#.name]",
				},
			},
			want: want{
				paths: []string{
					"families.0.surname", "families.1.surname", "families.2.surname",
					"families.0.members.0.name", "families.0.members.1.name",
					"families.1.members.0.name", "families.1.members.1.name",
					"families.2.members.0.name",
				},
			},
		},
		{
			name: "complex multipath with mixed nesting levels",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "[families.#.surname,families.#.members.#.hobbies.#.name,families.#.members.#.hobbies.#.locations.#]",
				},
			},
			want: want{
				paths: []string{
					"families.0.surname", "families.1.surname", "families.2.surname",
					"families.0.members.0.hobbies.0.name",
					"families.0.members.0.hobbies.1.name",
					"families.0.members.1.hobbies.0.name",
					"families.0.members.1.hobbies.1.name",
					"families.1.members.0.hobbies.0.name",
					"families.1.members.0.hobbies.1.name",
					"families.1.members.1.hobbies.0.name",
					"families.1.members.1.hobbies.1.name",
					"families.2.members.0.hobbies.0.name",
					"families.2.members.0.hobbies.1.name",
					"families.0.members.0.hobbies.0.locations.0",
					"families.0.members.0.hobbies.0.locations.1",
					"families.0.members.0.hobbies.0.locations.2",
					"families.0.members.0.hobbies.1.locations.0",
					"families.0.members.0.hobbies.1.locations.1",
					"families.0.members.1.hobbies.0.locations.0",
					"families.0.members.1.hobbies.0.locations.1",
					"families.0.members.1.hobbies.1.locations.0",
					"families.0.members.1.hobbies.1.locations.1",
					"families.0.members.1.hobbies.1.locations.2",
					"families.1.members.0.hobbies.0.locations.0",
					"families.1.members.0.hobbies.0.locations.1",
					"families.1.members.0.hobbies.0.locations.2",
					"families.1.members.0.hobbies.1.locations.0",
					"families.1.members.0.hobbies.1.locations.1",
					"families.1.members.0.hobbies.1.locations.2",
					"families.1.members.1.hobbies.0.locations.0",
					"families.1.members.1.hobbies.0.locations.1",
					"families.1.members.1.hobbies.0.locations.2",
					"families.1.members.1.hobbies.1.locations.0",
					"families.1.members.1.hobbies.1.locations.1",
					"families.1.members.1.hobbies.1.locations.2",
					"families.1.members.1.hobbies.1.locations.3",
					"families.2.members.0.hobbies.0.locations.0",
					"families.2.members.0.hobbies.0.locations.1",
					"families.2.members.0.hobbies.1.locations.0",
					"families.2.members.0.hobbies.1.locations.1",
					"families.2.members.0.hobbies.1.locations.2",
				},
			},
		},

		// Literals tests
		{
			name: "string literal",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "name.first,!\"Happysoft\"",
				},
			},
			want: want{
				paths: []string{"name.first"},
			},
		},
		{
			name: "boolean literal true",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "age,!true",
				},
			},
			want: want{
				paths: []string{"age"},
			},
		},
		{
			name: "boolean literal false",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "age,!false",
				},
			},
			want: want{
				paths: []string{"age"},
			},
		},
		{
			name: "number literal",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "age,!42",
				},
			},
			want: want{
				paths: []string{"age"},
			},
		},
		{
			name: "null literal",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "age,!null",
				},
			},
			want: want{
				paths: []string{"age"},
			},
		},
		{
			name: "multipath with literals",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "{name.first,age,\"company\":!\"Happysoft\",\"employed\":!true}",
				},
			},
			want: want{
				paths: []string{"name.first", "age"},
			},
		},

		// Edge cases and boundary conditions
		{
			name: "empty path",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "",
				},
			},
			want: want{
				paths: []string{""},
			},
		},
		{
			name: "root only path",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "@this",
				},
			},
			want: want{
				paths: []string{"@this"},
			},
		},
		{
			name: "non-existent field",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "nonexistent",
				},
			},
			want: want{
				paths: []string{"nonexistent"},
			},
		},

		{
			name: "complex nested multipath",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "[{name.first,age},{\"friends\":friends.#.first}]",
				},
			},
			want: want{
				paths: []string{"name.first", "age", "friends.0.first", "friends.1.first", "friends.2.first"},
			},
		},
		{
			name: "wildcard with numeric field names",
			given: given{
				args: args{
					json: []byte(`{"field1": "a", "field2": "b", "field3": "c", "other": "d"}`),
					path: "field*",
				},
			},
			want: want{
				paths: []string{"field1", "field2", "field3"},
			},
		},
		{
			name: "query with empty result",
			given: given{
				args: args{
					json: []byte(shared.json),
					path: "friends.#(age>100)#.first",
				},
			},
			want: want{
				paths: []string{},
			},
		},
		{
			name: "multiple wildcards in path",
			given: given{
				args: args{
					json: []byte(`{"section1": {"item1": "a", "item2": "b"}, "section2": {"item1": "c", "item2": "d"}}`),
					path: "section*.item*",
				},
			},
			want: want{
				paths: []string{"section1.item1", "section1.item2", "section2.item1", "section2.item2"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			/* ---------------------------------- Given --------------------------------- */
			require := require.New(t)

			/* ---------------------------------- When ---------------------------------- */
			got := ExpandPath(tt.given.args.json, tt.given.args.path)

			/* ---------------------------------- Then ---------------------------------- */
			require.Equal(tt.want.paths, got, "ExpandPath() returned unexpected paths")
		})
	}
}

func BenchmarkExpandPath(b *testing.B) {
	// Shared test JSON data
	json := []byte(`{
		"name": {"first": "Tom", "last": "Anderson"},
		"age": 37,
		"children": ["Sara","Alex","Jack"],
		"fav.movie": "Deer Hunter",
		"friends": [
			{"first": "Dale", "last": "Murphy", "age": 44, "nets": ["ig", "fb", "tw"]},
			{"first": "Roger", "last": "Craig", "age": 68, "nets": ["fb", "tw"]},
			{"first": "Jane", "last": "Murphy", "age": 47, "nets": ["ig", "tw"]}
		],
		"vals": [
			{ "a": 1, "b": "data" },
			{ "a": 2, "b": true },
			{ "a": 3, "b": false },
			{ "a": 4, "b": "0" },
			{ "a": 5, "b": 0 },
			{ "a": 6, "b": "1" },
			{ "a": 7, "b": 1 },
			{ "a": 8, "b": "true" },
			{ "a": 9, "b": false },
			{ "a": 10, "b": null },
			{ "a": 11 }
		],
		"families": [
			{
				"surname": "Smith",
				"members": [
					{
						"name": "John",
						"age": 45,
						"hobbies": [
							{
								"name": "reading",
								"locations": ["home", "library", "park"]
							},
							{
								"name": "cycling",
								"locations": ["park", "trail"]
							}
						]
					},
					{
						"name": "Jane",
						"age": 42,
						"hobbies": [
							{
								"name": "cooking",
								"locations": ["home", "restaurant"]
							},
							{
								"name": "gardening",
								"locations": ["home", "community garden", "nursery"]
							}
						]
					}
				]
			},
			{
				"surname": "Johnson",
				"members": [
					{
						"name": "Bob",
						"age": 38,
						"hobbies": [
							{
								"name": "fishing",
								"locations": ["lake", "river", "ocean"]
							}
						]
					},
					{
						"name": "Alice",
						"age": 35,
						"hobbies": [
							{
								"name": "painting",
								"locations": ["studio", "outdoors", "gallery", "home"]
							}
						]
					}
				]
			}
		]
	}`)

	benchmarks := []struct {
		name string
		path string
	}{
		// Simple field access
		{"SimpleField", "name.first"},
		{"NestedField", "name.last"},
		{"ArrayIndex", "children.0"},

		// Array operations
		{"ArrayLength", "children.#"},
		{"ArrayMap", "friends.#.first"},
		{"NestedArrayMap", "friends.#.nets.#"},

		// Queries
		{"SimpleQuery", "friends.#(age>40)#.first"},
		{"PatternQuery", "friends.#(first%\"D*\")#.last"},
		{"NegativePatternQuery", "friends.#(first!%\"R*\")#.age"},

		// Complex nested operations
		{"DeepNesting", "families.#.members.#.name"},
		{"DeepQuery", "families.#.members.#(age<40)#.name"},

		// Wildcards
		{"Wildcard", "name.*"},
		{"MultipleWildcards", "friends.*.nets.*"},

		// Mixed operations
		{"ComplexMixed", "families.0.members.#.hobbies.#.name"},

		// Multipath
		{"SimpleMultipath", "[name.first,age]"},
		{"ComplexMultipath", "[name.first,friends.#.first,children.#]"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = ExpandPath(json, bm.path)
			}
		})
	}
}

// BenchmarkExpandPath_Allocations focuses specifically on allocation patterns
func BenchmarkExpandPath_Allocations(b *testing.B) {
	json := []byte(`{
		"users": [
			{"id": 1, "name": "Alice", "active": true},
			{"id": 2, "name": "Bob", "active": false},
			{"id": 3, "name": "Charlie", "active": true},
			{"id": 4, "name": "Diana", "active": true},
			{"id": 5, "name": "Eve", "active": false}
		]
	}`)

	testCases := []struct {
		name string
		path string
		desc string
	}{
		{
			name: "NoAllocation_SimpleField",
			path: "users.0.name",
			desc: "Should minimize allocations for simple field access",
		},
		{
			name: "MinimalAllocation_ArrayMap",
			path: "users.#.name",
			desc: "Should efficiently handle array mapping",
		},
		{
			name: "QueryAllocation_FilterActive",
			path: "users.#(active==true)#.name",
			desc: "Should handle query filtering efficiently",
		},
		{
			name: "ComplexAllocation_MultiLevel",
			path: "users.#.id,users.#.name",
			desc: "Should handle multipath operations efficiently",
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()

			// Warm up
			for i := 0; i < 10; i++ {
				_ = ExpandPath(json, tc.path)
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result := ExpandPath(json, tc.path)
				// Prevent compiler optimizations by using the result
				if len(result) == 0 && i == 0 {
					b.Logf("Unexpected empty result for path: %s", tc.path)
				}
			}
		})
	}
}

// BenchmarkExpandPath_MemoryProfile provides detailed memory profiling for specific scenarios
func BenchmarkExpandPath_MemoryProfile(b *testing.B) {
	// Large JSON data for more realistic memory profiling
	largeJSON := []byte(`{
		"users": [` +
		`{"id": 1, "name": "User1", "active": true, "tags": ["admin", "vip"]},` +
		`{"id": 2, "name": "User2", "active": false, "tags": ["user"]},` +
		`{"id": 3, "name": "User3", "active": true, "tags": ["moderator", "user"]},` +
		`{"id": 4, "name": "User4", "active": true, "tags": ["vip", "beta"]},` +
		`{"id": 5, "name": "User5", "active": false, "tags": ["user", "trial"]},` +
		`{"id": 6, "name": "User6", "active": true, "tags": ["admin", "moderator"]},` +
		`{"id": 7, "name": "User7", "active": true, "tags": ["user"]},` +
		`{"id": 8, "name": "User8", "active": false, "tags": ["trial"]},` +
		`{"id": 9, "name": "User9", "active": true, "tags": ["vip", "user"]},` +
		`{"id": 10, "name": "User10", "active": false, "tags": ["user", "inactive"]}` +
		`],
		"metadata": {
			"total": 10,
			"active": 6,
			"last_updated": "2025-10-03"
		}
	}`)

	memoryTests := []struct {
		name string
		path string
		desc string
	}{
		{
			name: "MemProfile_LargeArrayMap",
			path: "users.#.name",
			desc: "Memory usage for mapping over large arrays",
		},
		{
			name: "MemProfile_ComplexQuery",
			path: "users.#(active==true)#.tags.#",
			desc: "Memory usage for complex nested queries",
		},
		{
			name: "MemProfile_DeepMultipath",
			path: "[users.#.id,users.#.name,users.#.active,metadata.total]",
			desc: "Memory usage for complex multipath operations",
		},
	}

	for _, mt := range memoryTests {
		b.Run(mt.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result := ExpandPath(largeJSON, mt.path)
				// Prevent dead code elimination
				if result == nil {
					b.Fatal("Unexpected nil result")
				}
			}
		})
	}
}

// Helper comments for running benchmarks with profiling:
//
// To run benchmarks with CPU and memory profiling:
//   go test -bench=BenchmarkExpandPath -cpuprofile=cpu.prof -memprofile=mem.prof
//
// To analyze the profiles:
//   go tool pprof cpu.prof
//   go tool pprof mem.prof
//
// To run specific benchmark patterns:
//   go test -bench=BenchmarkExpandPath/SimpleField -benchmem
//   go test -bench=BenchmarkExpandPath_Allocations -benchmem -benchtime=100000x
//   go test -bench=BenchmarkExpandPath_MemoryProfile -benchmem
