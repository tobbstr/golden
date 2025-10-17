package gjson

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// resultPool reuses slices to reduce allocations
var resultPool = sync.Pool{
	New: func() any {
		s := make([]string, 0, 8) // Pre-allocate common capacity
		return &s
	},
}

// getResultSlice gets a reusable slice from the pool
func getResultSlice() []string {
	if v := resultPool.Get(); v != nil {
		if s, ok := v.(*[]string); ok {
			return (*s)[:0]
		}
	}
	return make([]string, 0, 8)
}

// putResultSlice returns a slice to the pool
func putResultSlice(s []string) {
	if cap(s) < 64 { // Only pool reasonably-sized slices
		resultPool.Put(&s)
	}
}

// ExpandPath expands the GJSON path into concrete escaped paths found in the JSON document.
//
// For more information about the GJSON path syntax, see: https://github.com/tidwall/gjson/blob/master/SYNTAX.md
func ExpandPath(jsonData []byte, path string) []string {
	if path == "" {
		return []string{""}
	}

	// Parse JSON into any
	var data any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil
	}

	return expandPathWithData(data, path)
}

// expandPathWithData is the internal function that avoids re-parsing JSON
func expandPathWithData(data any, path string) []string {
	if path == "" {
		return []string{""}
	}

	// Handle special root cases
	if path == "@this" {
		return []string{"@this"}
	}

	// Handle multipath syntax [path1,path2] and {key1:path1,key2:path2}
	if strings.HasPrefix(path, "[") && strings.HasSuffix(path, "]") {
		return expandMultipathArrayWithData(data, path[1:len(path)-1])
	}
	if strings.HasPrefix(path, "{") && strings.HasSuffix(path, "}") {
		return expandMultipathObjectWithData(data, path[1:len(path)-1])
	}

	// Handle literals (return just the path without the literal)
	if strings.Contains(path, ",!") {
		parts := strings.Split(path, ",!")
		if len(parts) > 0 {
			return expandPathWithData(data, parts[0])
		}
	}

	// Expand single path
	result := expandSinglePath(data, path, "")
	if result == nil {
		return []string{}
	}
	return result
}

func expandMultipathArrayWithData(data any, paths string) []string {
	result := getResultSlice()
	pathList := parseMultipathComponents(paths)

	for _, p := range pathList {
		// Skip literal values (starting with !)
		if strings.HasPrefix(strings.TrimSpace(p), "!") {
			continue
		}
		expanded := expandPathWithData(data, p)
		result = append(result, expanded...)
	}

	if len(result) == 0 {
		putResultSlice(result)
		return nil
	}

	// Make a copy to return since we're pooling the slice
	final := make([]string, len(result))
	copy(final, result)
	putResultSlice(result)
	return final
}

func expandMultipathObjectWithData(data any, paths string) []string {
	result := getResultSlice()
	components := parseMultipathObjectComponents(paths)

	for _, comp := range components {
		// Skip literal values (starting with !)
		if strings.HasPrefix(strings.TrimSpace(comp.path), "!") {
			continue
		}
		expanded := expandPathWithData(data, comp.path)
		result = append(result, expanded...)
	}

	if len(result) == 0 {
		putResultSlice(result)
		return nil
	}

	// Make a copy to return since we're pooling the slice
	final := make([]string, len(result))
	copy(final, result)
	putResultSlice(result)
	return final
}

type multipathComponent struct {
	key  string
	path string
}

func parseMultipathObjectComponents(paths string) []multipathComponent {
	if paths == "" {
		return nil
	}

	var components []multipathComponent
	var start int
	var inQuotes bool
	var escape bool

	for i, r := range paths {
		if escape {
			escape = false
			continue
		}

		if r == '\\' {
			escape = true
			continue
		}

		if r == '"' {
			inQuotes = !inQuotes
		} else if r == ',' && !inQuotes {
			if i > start {
				part := strings.TrimSpace(paths[start:i])
				if part != "" {
					if colonIdx := strings.Index(part, ":"); colonIdx != -1 && part[0] == '"' {
						// Extract key and path from "key":path format
						key := part[:colonIdx]
						path := part[colonIdx+1:]
						components = append(components, multipathComponent{key: key, path: path})
					} else {
						// Regular path without custom key
						components = append(components, multipathComponent{path: part})
					}
				}
			}
			start = i + 1
		}
	}

	if start < len(paths) {
		part := strings.TrimSpace(paths[start:])
		if part != "" {
			if colonIdx := strings.Index(part, ":"); colonIdx != -1 && part[0] == '"' {
				// Extract key and path from "key":path format
				key := part[:colonIdx]
				path := part[colonIdx+1:]
				components = append(components, multipathComponent{key: key, path: path})
			} else {
				// Regular path without custom key
				components = append(components, multipathComponent{path: part})
			}
		}
	}

	return components
}

func parseMultipathComponents(paths string) []string {
	if paths == "" {
		return nil
	}

	var components []string
	var start int
	var inQuotes bool
	var escape bool
	var depth int

	for i, r := range paths {
		if escape {
			escape = false
			continue
		}

		if r == '\\' {
			escape = true
			continue
		}

		if r == '"' {
			inQuotes = !inQuotes
		}

		if !inQuotes {
			switch r {
			case '(', '[', '{':
				depth++
			case ')', ']', '}':
				depth--
			}
		}

		if r == ',' && !inQuotes && depth == 0 {
			if i > start {
				component := strings.TrimSpace(paths[start:i])
				if component != "" {
					components = append(components, component)
				}
			}
			start = i + 1
		}
	}

	if start < len(paths) {
		component := strings.TrimSpace(paths[start:])
		if component != "" {
			components = append(components, component)
		}
	}

	return components
}

func expandSinglePath(data any, path string, currentPath string) []string {
	if path == "" {
		return []string{currentPath}
	}

	// Handle modifiers
	if strings.HasPrefix(path, "@") {
		return []string{appendPath(currentPath, path)}
	}

	// Split path into components, respecting separators
	components := parsePathComponents(path)
	if len(components) == 0 {
		return []string{currentPath}
	}

	return expandPathComponent(data, components, 0, currentPath)
}

func expandPathComponent(data any, components []PathComponent, index int, currentPath string) []string {
	if index >= len(components) {
		return []string{currentPath}
	}

	pathComp := components[index]
	component := pathComp.Component

	// Handle tilde operators
	if strings.HasPrefix(component, "~") {
		return []string{appendPath(currentPath, component)}
	}

	// Handle queries #(...) or #[...] - this must come before other # checks
	if (strings.Contains(component, "#(") && strings.Contains(component, ")")) ||
		(strings.Contains(component, "#[") && strings.Contains(component, "]")) {
		return expandQuery(data, component, components, index, currentPath)
	}

	// Handle pure array length #
	if component == "#" {
		// If this is the last component, decide whether to expand or return length path
		if index == len(components)-1 {
			// Count the number of # components in the path to determine behavior
			hashCount := 0
			for _, comp := range components {
				if comp.Component == "#" {
					hashCount++
				}
			}

			// Check if the current path indicates deep nesting with individual element access
			// Look for patterns like families.X.members.X.hobbies.X.locations.# (numbers in path)
			hasNumericIndices := false
			if currentPath != "" {
				pathParts := strings.Split(currentPath, ".")
				numericCount := 0
				for _, part := range pathParts {
					if _, err := strconv.Atoi(part); err == nil {
						numericCount++
					}
				}
				// If we have 3+ numeric indices, we're deep enough to expand to individual elements
				if numericCount >= 3 {
					hasNumericIndices = true
				}
			}

			if hasNumericIndices {
				if arr, ok := data.([]any); ok {
					var results []string
					for i := range arr {
						results = append(results, appendPath(currentPath, fmt.Sprintf("%d", i)))
					}
					return results
				}
			}

			// Default: return the # path (for array length queries)
			return []string{appendPath(currentPath, "#")}
		}

		// Check if next component uses pipe separator
		if index+1 < len(components) && components[index+1].Separator == "|" {
			// Pipe behavior: apply next component to the current data array as a whole
			if arr, ok := data.([]any); ok {
				// For pipe, we pass the array itself to the next component
				nextComponent := components[index+1].Component
				// Create a path to the array itself (without indices)
				arrayPath := currentPath

				// Apply the next component to the array data itself
				// This will typically fail since arrays don't have object fields like "first"
				if obj, ok := any(arr).(map[string]any); ok {
					if field, exists := obj[nextComponent]; exists {
						_ = field // Use the field value
						return expandPathComponent(field, components, index+2, appendPath(arrayPath, nextComponent))
					}
				}
				// If the array doesn't have the requested field, return empty
				return []string{}
			}
			return []string{appendPath(currentPath, "#")}
		}

		// Otherwise, this is array expansion - expand current data as array
		if arr, ok := data.([]any); ok {
			var results []string
			for i := range arr {
				indexPath := appendPath(currentPath, fmt.Sprintf("%d", i))
				// Continue with remaining components
				subResults := expandPathComponent(arr[i], components, index+1, indexPath)
				results = append(results, subResults...)
			}
			return results
		}

		return []string{appendPath(currentPath, "#")}
	}

	// Handle array operations with #
	if strings.Contains(component, "#") {
		return expandArrayOperation(data, component, components, index, currentPath)
	}

	// Handle wildcards (but not escaped ones)
	if (strings.Contains(component, "*") && !strings.Contains(component, "\\*")) ||
		(strings.Contains(component, "?") && !strings.Contains(component, "\\?")) {
		return expandWildcard(data, component, components, index, currentPath)
	}

	// Handle regular field access
	return expandRegularField(data, component, components, index, currentPath)
}

func expandArrayOperation(data any, component string, components []PathComponent, index int, currentPath string) []string {
	var results []string

	// Handle pure # (array length)
	if component == "#" {
		return []string{appendPath(currentPath, "#")}
	}

	// Handle array mapping with # like "members.#.name" or "friends.#.first"
	if strings.Contains(component, ".#.") {
		parts := strings.Split(component, ".#.")
		if len(parts) == 2 {
			fieldName := parts[0]
			afterField := parts[1]

			fieldPath := appendPath(currentPath, fieldName)
			if arr, ok := getFieldValue(data, fieldName).([]any); ok {
				for i := range arr {
					indexPath := appendPath(fieldPath, fmt.Sprintf("%d", i))
					subResults := expandSinglePath(arr[i], afterField, indexPath)
					results = append(results, subResults...)
				}
			}

			// Handle remaining components
			if len(components) > index+1 {
				var finalResults []string
				for _, result := range results {
					remaining := joinPathComponents(components[index+1:])
					subResults := expandSinglePath(getValueAtPath(data, result), remaining, result)
					finalResults = append(finalResults, subResults...)
				}
				return finalResults
			}

			return results
		}
	}

	// Handle array mapping with # like "members.#"
	if strings.HasSuffix(component, ".#") {
		fieldName := component[:len(component)-2]
		fieldPath := appendPath(currentPath, fieldName)

		if arr, ok := getFieldValue(data, fieldName).([]any); ok {
			for i := range arr {
				results = append(results, appendPath(fieldPath, fmt.Sprintf("%d", i)))
			}
		}

		if len(components) > index+1 {
			// Continue with remaining path components
			var finalResults []string
			for _, result := range results {
				remaining := joinPathComponents(components[index+1:])
				subResults := expandSinglePath(getValueAtPath(data, result), remaining, result)
				finalResults = append(finalResults, subResults...)
			}
			return finalResults
		}

		return results
	}

	// Handle # at the beginning or middle of component
	if strings.HasPrefix(component, "#") {
		if len(component) == 1 {
			return []string{appendPath(currentPath, "#")}
		}
		// Handle #.field pattern
		if strings.HasPrefix(component, "#.") {
			remainingPath := component[2:]
			if arr, ok := data.([]any); ok {
				for i := range arr {
					indexPath := appendPath(currentPath, fmt.Sprintf("%d", i))
					subResults := expandSinglePath(arr[i], remainingPath, indexPath)
					results = append(results, subResults...)
				}
			}

			if len(components) > index+1 {
				var finalResults []string
				for _, result := range results {
					remaining := joinPathComponents(components[index+1:])
					subResults := expandSinglePath(getValueAtPath(data, result), remaining, result)
					finalResults = append(finalResults, subResults...)
				}
				return finalResults
			}

			return results
		}
	}

	// Handle field.# pattern (array length of field)
	if strings.HasSuffix(component, "#") && len(component) > 1 {
		fieldName := component[:len(component)-1]
		fieldName = strings.TrimSuffix(fieldName, ".")

		fieldPath := appendPath(currentPath, fieldName)
		return []string{appendPath(fieldPath, "#")}
	}

	return []string{appendPath(currentPath, component)}
}

func expandQuery(data any, component string, components []PathComponent, index int, currentPath string) []string {
	var results []string

	// Parse query: field.#(condition)#.otherfield or field.#[condition]#.otherfield
	var queryStart, queryEnd int
	var queryOffset int

	if strings.Contains(component, "#(") {
		queryStart = strings.Index(component, "#(")
		queryEnd = strings.LastIndex(component, ")")
		queryOffset = 2 // "#(" length
	} else if strings.Contains(component, "#[") {
		queryStart = strings.Index(component, "#[")
		queryEnd = strings.LastIndex(component, "]")
		queryOffset = 2 // "#[" length
	} else {
		return []string{appendPath(currentPath, component)}
	}

	if queryStart == -1 || queryEnd == -1 {
		return []string{appendPath(currentPath, component)}
	}

	fieldPart := component[:queryStart]
	queryPart := component[queryStart+queryOffset : queryEnd]
	afterQuery := component[queryEnd+1:]

	var fieldPath string
	var arrayData []any

	if fieldPart == "" {
		// Direct query on current data
		if arr, ok := data.([]any); ok {
			arrayData = arr
			fieldPath = currentPath
		}
	} else {
		// Query on specific field
		fieldPath = appendPath(currentPath, fieldPart)
		if fieldValue := getFieldValue(data, fieldPart); fieldValue != nil {
			if arr, ok := fieldValue.([]any); ok {
				arrayData = arr
			}
		}
	}

	// Find matching indices
	matchingIndices := findMatchingIndices(arrayData, queryPart)

	// Handle suffix after query
	if strings.HasPrefix(afterQuery, "#") {
		// Return all matching elements (#.field or # alone)
		if afterQuery == "#" {
			// Just return the matching indices
			for _, idx := range matchingIndices {
				results = append(results, appendPath(fieldPath, fmt.Sprintf("%d", idx)))
			}
		} else if strings.HasPrefix(afterQuery, "#.") {
			// Continue with field access on matching elements
			remainingField := afterQuery[2:]
			for _, idx := range matchingIndices {
				indexPath := appendPath(fieldPath, fmt.Sprintf("%d", idx))
				if len(remainingField) > 0 && idx < len(arrayData) {
					subResults := expandSinglePath(arrayData[idx], remainingField, indexPath)
					results = append(results, subResults...)
				} else {
					results = append(results, indexPath)
				}
			}
		}
	} else if afterQuery == "" {
		// Query without # suffix - determine behavior based on context
		hasTrailingHash := strings.HasSuffix(component, "#")

		// Parse the query to check if it uses pattern operators
		condition := parseQueryCondition(queryPart)
		isPatternOperator := condition.operator == "%" || condition.operator == "!%"

		if hasTrailingHash || isPatternOperator {
			// Return ALL matches if:
			// 1. Query has trailing # (like "field.#(condition)#"), OR
			// 2. Query uses pattern operators (% or !%) regardless of #
			for _, idx := range matchingIndices {
				results = append(results, appendPath(fieldPath, fmt.Sprintf("%d", idx)))
			}
		} else {
			// Return first match only for other operators without trailing #
			if len(matchingIndices) > 0 {
				results = append(results, appendPath(fieldPath, fmt.Sprintf("%d", matchingIndices[0])))
			}
		}
	} else if strings.HasPrefix(afterQuery, ".") {
		// Continue with field access (.field) - return ALL matches for now
		remainingField := afterQuery[1:]
		for _, idx := range matchingIndices {
			if idx < len(arrayData) {
				indexPath := appendPath(fieldPath, fmt.Sprintf("%d", idx))
				subResults := expandSinglePath(arrayData[idx], remainingField, indexPath)
				results = append(results, subResults...)
			}
		}
	}

	// Handle remaining components
	if len(components) > index+1 {
		var finalResults []string
		for _, result := range results {
			remaining := joinPathComponents(components[index+1:])
			subResults := expandSinglePath(getValueAtPath(data, result), remaining, result)
			finalResults = append(finalResults, subResults...)
		}
		return finalResults
	}

	return results
}

func findMatchingIndices(arrayData []any, query string) []int {
	var indices []int

	if len(arrayData) == 0 {
		return indices
	}

	// Parse query condition
	condition := parseQueryCondition(query)

	for i, item := range arrayData {
		if evaluateCondition(item, condition) {
			indices = append(indices, i)
		}
	}

	return indices
}

type queryCondition struct {
	field    string
	operator string
	value    string
}

func parseQueryCondition(query string) queryCondition {
	// Handle nested array queries like "nets.#(=="fb")" - these are self-contained conditions
	if strings.Contains(query, ".#(") && strings.Contains(query, ")") {
		return queryCondition{
			field:    query,
			operator: "==",
			value:    "true", // The nested query itself is the condition
		}
	}

	// Handle operators: ==, !=, >, <, >=, <=, %, !%
	operators := []string{"!=", "!%", "==", ">=", "<=", ">", "<", "%"}

	for _, op := range operators {
		// Find operator position, but skip if it's inside parentheses
		idx := findOperatorOutsideParentheses(query, op)
		if idx != -1 {
			field := strings.TrimSpace(query[:idx])
			value := strings.TrimSpace(query[idx+len(op):])

			// Remove quotes from value if present
			if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
				value = value[1 : len(value)-1]
			}

			return queryCondition{
				field:    field,
				operator: op,
				value:    value,
			}
		}
	}

	// Handle simple equality (no operator means ==)
	if strings.Contains(query, "=") {
		parts := strings.SplitN(query, "=", 2)
		if len(parts) == 2 {
			field := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
				value = value[1 : len(value)-1]
			}
			return queryCondition{field: field, operator: "==", value: value}
		}
	}

	// Handle direct value comparison (no field specified)
	return queryCondition{field: "", operator: "==", value: query}
}

func findOperatorOutsideParentheses(query string, operator string) int {
	depth := 0
	for i := 0; i <= len(query)-len(operator); i++ {
		if query[i] == '(' {
			depth++
		} else if query[i] == ')' {
			depth--
		} else if depth == 0 && strings.HasPrefix(query[i:], operator) {
			return i
		}
	}
	return -1
}

func evaluateCondition(item any, condition queryCondition) bool {
	// Handle nested array queries like "nets.#(=="fb")" - these are self-contained conditions
	if strings.Contains(condition.field, ".#(") && strings.Contains(condition.field, ")") && condition.value == "true" {
		return evaluateNestedArrayQuery(item, condition)
	}

	// Handle tilde operators specially with field context
	if strings.HasPrefix(condition.value, "~") {
		if condition.field == "" {
			return evaluateTildeCondition(item, condition.operator, condition.value)
		} else {
			return evaluateTildeConditionWithContext(item, condition.field, condition.operator, condition.value)
		}
	}

	var itemValue any
	if condition.field == "" {
		itemValue = item
	} else {
		itemValue = getFieldValue(item, condition.field)
	}

	itemStr := fmt.Sprintf("%v", itemValue)
	conditionValue := condition.value

	switch condition.operator {
	case "==":
		return itemStr == conditionValue
	case "!=":
		return itemStr != conditionValue
	case ">":
		if itemNum, err := strconv.ParseFloat(itemStr, 64); err == nil {
			if condNum, err := strconv.ParseFloat(conditionValue, 64); err == nil {
				return itemNum > condNum
			}
		}
		return itemStr > conditionValue
	case "<":
		if itemNum, err := strconv.ParseFloat(itemStr, 64); err == nil {
			if condNum, err := strconv.ParseFloat(conditionValue, 64); err == nil {
				return itemNum < condNum
			}
		}
		return itemStr < conditionValue
	case ">=":
		if itemNum, err := strconv.ParseFloat(itemStr, 64); err == nil {
			if condNum, err := strconv.ParseFloat(conditionValue, 64); err == nil {
				return itemNum >= condNum
			}
		}
		return itemStr >= conditionValue
	case "<=":
		if itemNum, err := strconv.ParseFloat(itemStr, 64); err == nil {
			if condNum, err := strconv.ParseFloat(conditionValue, 64); err == nil {
				return itemNum <= condNum
			}
		}
		return itemStr <= conditionValue
	case "%":
		matched, _ := matchPattern(itemStr, conditionValue)
		return matched
	case "!%":
		matched, _ := matchPattern(itemStr, conditionValue)
		return !matched
	}

	return false
}

func evaluateTildeCondition(itemValue any, operator, tildeValue string) bool {
	tildeOp := tildeValue[1:] // Remove the ~

	var result bool
	switch tildeOp {
	case "true":
		result = isTruthy(itemValue)
	case "false":
		result = isFalsy(itemValue)
	case "null":
		result = isNull(itemValue)
	case "*":
		result = exists(itemValue)
	default:
		return false
	}

	// Apply the operator (== or !=)
	if operator == "!=" {
		return !result
	}
	return result
}

func evaluateTildeConditionWithContext(item any, field string, operator, tildeValue string) bool {
	tildeOp := tildeValue[1:] // Remove the ~

	var result bool
	switch tildeOp {
	case "*":
		// For exists operator, we need to check if the field actually exists
		if obj, ok := item.(map[string]any); ok {
			_, fieldExists := obj[field]
			result = fieldExists
		} else {
			result = false
		}
	case "false":
		// For false operator, we need to handle missing fields specially
		if obj, ok := item.(map[string]any); ok {
			if fieldValue, fieldExists := obj[field]; fieldExists {
				result = isFalsy(fieldValue)
			} else {
				// Missing field is considered falsy
				result = true
			}
		} else {
			result = true
		}
	default:
		// For other operators, get the field value normally
		itemValue := getFieldValue(item, field)
		switch tildeOp {
		case "true":
			result = isTruthy(itemValue)
		case "null":
			result = isNull(itemValue)
		default:
			return false
		}
	}

	// Apply the operator (== or !=)
	if operator == "!=" {
		return !result
	}
	return result
}

func isTruthy(value any) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v == "1" || v == "true"
	case float64:
		return v == 1
	case int:
		return v == 1
	default:
		return false
	}
}

func isFalsy(value any) bool {
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case bool:
		return !v
	case string:
		return v == "0" || v == "false" || v == ""
	case float64:
		return v == 0
	case int:
		return v == 0
	default:
		return false
	}
}

func isNull(value any) bool {
	return value == nil
}

func exists(_ any) bool {
	// For tilde * operator without field context (direct array element check).
	// If we reached this evaluation during array iteration, the element exists
	// in the array by definition, even if its value is null.
	// Field-level existence checks are handled by evaluateTildeConditionWithContext.
	return true
}

func evaluateNestedArrayQuery(item any, condition queryCondition) bool {
	// Handle nested array queries like "nets.#(=="fb")"
	// Parse the field: "nets.#(=="fb")"
	field := condition.field
	queryStart := strings.Index(field, ".#(")
	queryEnd := strings.LastIndex(field, ")")

	if queryStart == -1 || queryEnd == -1 {
		return false
	}

	fieldName := field[:queryStart]               // "nets"
	nestedQuery := field[queryStart+3 : queryEnd] // "=="fb""

	// Get the array field
	arrayValue := getFieldValue(item, fieldName)
	arr, ok := arrayValue.([]any)
	if !ok {
		return false
	}

	// Parse the nested condition
	nestedCondition := parseQueryCondition(nestedQuery)

	// Check if any element in the array matches the condition
	for _, element := range arr {
		if evaluateCondition(element, nestedCondition) {
			return condition.operator == "=="
		}
	}

	// No matches found
	return condition.operator == "!="
}

func matchPattern(text, pattern string) (bool, error) {
	// Convert GJSON pattern to regex
	regexPattern := strings.ReplaceAll(pattern, "*", ".*")
	regexPattern = strings.ReplaceAll(regexPattern, "?", ".")
	regexPattern = "^" + regexPattern + "$"

	return regexp.MatchString(regexPattern, text)
}

func expandWildcard(data any, component string, components []PathComponent, index int, currentPath string) []string {
	var results []string

	// Handle object field wildcard matching
	if obj, ok := data.(map[string]any); ok {
		// Get keys and sort them for deterministic order
		var keys []string
		for key := range obj {
			if matchWildcard(key, component) {
				keys = append(keys, key)
			}
		}
		// Sort keys to ensure consistent ordering
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				if keys[i] > keys[j] {
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
		}
		for _, key := range keys {
			results = append(results, appendPath(currentPath, key))
		}
	}

	// Handle remaining components
	if len(components) > index+1 {
		var finalResults []string
		for _, result := range results {
			remaining := joinPathComponents(components[index+1:])
			subResults := expandSinglePath(getValueAtPath(data, result), remaining, result)
			finalResults = append(finalResults, subResults...)
		}
		return finalResults
	}

	return results
}

func matchWildcard(text, pattern string) bool {
	// Convert wildcard pattern to regex
	regexPattern := regexp.QuoteMeta(pattern)
	regexPattern = strings.ReplaceAll(regexPattern, "\\*", ".*")
	regexPattern = strings.ReplaceAll(regexPattern, "\\?", ".")
	regexPattern = "^" + regexPattern + "$"

	matched, _ := regexp.MatchString(regexPattern, text)
	return matched
}

func expandRegularField(data any, component string, components []PathComponent, index int, currentPath string) []string {
	// Handle array index access
	if idx, err := strconv.Atoi(component); err == nil {
		indexPath := appendPath(currentPath, component)
		if len(components) > index+1 {
			remaining := joinPathComponents(components[index+1:])
			if arr, ok := data.([]any); ok && idx >= 0 && idx < len(arr) {
				return expandSinglePath(arr[idx], remaining, indexPath)
			}
			// Index out of bounds - return empty result
			return []string{}
		}
		// Final component - only return path if index is valid
		if arr, ok := data.([]any); ok && idx >= 0 && idx < len(arr) {
			return []string{indexPath}
		}
		return []string{}
	}

	// Handle object field access (including escaped field names)
	fieldPath := appendPath(currentPath, component)
	if len(components) > index+1 {
		remaining := joinPathComponents(components[index+1:])
		fieldValue := getFieldValue(data, component)
		return expandSinglePath(fieldValue, remaining, fieldPath)
	}

	// Check if field exists (for escaped field names, we need to check the actual field)
	actualFieldName := unescapeFieldName(component)
	if obj, ok := data.(map[string]any); ok {
		if _, exists := obj[actualFieldName]; exists {
			return []string{fieldPath}
		}
	}

	return []string{fieldPath}
}

func unescapeFieldName(fieldName string) string {
	// Remove escape characters for actual field lookup
	result := strings.ReplaceAll(fieldName, "\\.", ".")
	result = strings.ReplaceAll(result, "\\*", "*")
	result = strings.ReplaceAll(result, "\\?", "?")
	result = strings.ReplaceAll(result, "\\|", "|")
	result = strings.ReplaceAll(result, "\\#", "#")
	result = strings.ReplaceAll(result, "\\@", "@")
	result = strings.ReplaceAll(result, "\\!", "!")
	return result
}

// PathComponent represents a component and its preceding separator
type PathComponent struct {
	Component string
	Separator string // ".", "|", or "" for first component
}

// joinPathComponents joins path components back into a string using dots
func joinPathComponents(components []PathComponent) string {
	if len(components) == 0 {
		return ""
	}

	var parts []string
	for _, comp := range components {
		parts = append(parts, comp.Component)
	}
	return strings.Join(parts, ".")
}

func parsePathComponents(path string) []PathComponent {
	if path == "" {
		return nil
	}

	var components []PathComponent
	var start int
	var escape bool
	var inQuery bool
	var queryDepth int
	var lastSeparator string

	for i := 0; i < len(path); i++ {
		r := rune(path[i])

		if escape {
			escape = false
			continue
		}

		if r == '\\' {
			escape = true
			continue
		}

		// Handle query parentheses
		if r == '(' && !escape {
			inQuery = true
			queryDepth++
		} else if r == ')' && !escape {
			queryDepth--
			if queryDepth == 0 {
				inQuery = false
			}
		}

		// Handle separators (. and |) but not within queries
		if (r == '.' || r == '|') && !inQuery && !escape {
			if i > start {
				component := path[start:i]
				if component != "" {
					components = append(components, PathComponent{
						Component: component,
						Separator: lastSeparator,
					})
				}
			}
			lastSeparator = string(r)
			start = i + 1
		}
	}

	if start < len(path) {
		component := path[start:]
		if component != "" {
			components = append(components, PathComponent{
				Component: component,
				Separator: lastSeparator,
			})
		}
	}

	return components
}

func getFieldValue(data any, fieldName string) any {
	if obj, ok := data.(map[string]any); ok {
		// Try escaped field name first, then unescaped
		if val, exists := obj[fieldName]; exists {
			return val
		}
		// Try with unescaped field name
		actualFieldName := unescapeFieldName(fieldName)
		return obj[actualFieldName]
	}
	return nil
}

func getValueAtPath(rootData any, path string) any {
	if path == "" {
		return rootData
	}

	components := strings.Split(path, ".")
	current := rootData

	for _, component := range components {
		if component == "#" {
			continue
		}

		if idx, err := strconv.Atoi(component); err == nil {
			if arr, ok := current.([]any); ok && idx >= 0 && idx < len(arr) {
				current = arr[idx]
			} else {
				return nil
			}
		} else {
			if obj, ok := current.(map[string]any); ok {
				current = obj[component]
			} else {
				return nil
			}
		}
	}

	return current
}

func appendPath(currentPath, component string) string {
	if currentPath == "" {
		return component
	}
	return currentPath + "." + component
}
