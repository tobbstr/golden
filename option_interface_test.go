package golden

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestOptionSortingInterface verifies that check functions run before modifier functions.
func TestOptionSortingInterface(t *testing.T) {
	// Test with a mix of check and modifier functions in random order
	options := []Option{
		WithSkippedFields("test"),                                             // Modifier function
		CheckNotZeroTime("time", time.RFC3339),                                // Check function
		WithFieldComments([]FieldComment{{Path: "name", Comment: "comment"}}), // Modifier function
		CheckEqualTimes("createdAt", "updatedAt", time.RFC3339),               // Check function
		UpdateGoldenFiles(),                                                   // Modifier function
		WithFileComment("File comment"),                                       // Modifier function
	}

	sorted := sortOptions(options)

	assert.Equal(t, 6, len(sorted), "Should have 6 options after sorting")

	// Verify that check functions come first
	checkCount := 0
	modifierCount := 0

	for i, opt := range sorted {
		if opt.IsType() == OptionTypeCheck {
			checkCount++
			if modifierCount > 0 {
				t.Errorf("Check function found at position %d after modifier functions", i)
			}
		} else if opt.IsType() == OptionTypeModifier {
			modifierCount++
		}
	}

	// We should have 2 check functions and 4 modifier functions
	assert.Equal(t, 2, checkCount, "Should have 2 check functions")
	assert.Equal(t, 4, modifierCount, "Should have 4 modifier functions")
}

// TestOptionInterface verifies that all option implementations correctly identify themselves.
func TestOptionInterface(t *testing.T) {
	testCases := []struct {
		name         string
		option       Option
		expectedType OptionType
	}{
		{
			name:         "WithSkippedFields should be modifier",
			option:       WithSkippedFields("test"),
			expectedType: OptionTypeModifier,
		},
		{
			name:         "WithFieldComments should be modifier",
			option:       WithFieldComments([]FieldComment{{Path: "test", Comment: "comment"}}),
			expectedType: OptionTypeModifier,
		},
		{
			name:         "WithFileComment should be modifier",
			option:       WithFileComment("comment"),
			expectedType: OptionTypeModifier,
		},
		{
			name:         "UpdateGoldenFiles should be modifier",
			option:       UpdateGoldenFiles(),
			expectedType: OptionTypeModifier,
		},
		{
			name:         "CheckNotZeroTime should be check",
			option:       CheckNotZeroTime("time", time.RFC3339),
			expectedType: OptionTypeCheck,
		},
		{
			name:         "CheckEqualTimes should be check",
			option:       CheckEqualTimes("a", "b", time.RFC3339),
			expectedType: OptionTypeCheck,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedType, tc.option.IsType(), "IsType() result mismatch")
		})
	}
}
