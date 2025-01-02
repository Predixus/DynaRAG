package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMetadataHashConsistency checks that a hash is possible
func TestMetadataHashPossible(t *testing.T) {
	testData := map[string]interface{}{
		"id":   123,
		"name": "test",
		"tags": []string{"a", "b", "c"},
		"nested": map[string]interface{}{
			"x": 1,
			"y": 2,
		},
	}
	hash1, err := CalculateMetadataHash(testData)
	assert.NoError(t, err)
	hash2, err := CalculateMetadataHash(testData)
	assert.Equal(t, hash1, hash2)
}

// TestEmptyInterface check that the empty interfaces passes successfully
func TestEmptyInterface(t *testing.T) {
	testData := map[string]interface{}{}
	hash1, err := CalculateMetadataHash(testData)
	assert.NoError(t, err)
	hash2, err := CalculateMetadataHash(testData)
	assert.NoError(t, err)
	assert.Equal(t, hash1, hash2)
}

// TestDifferentDataTypes verifies hashing works with various data types
func TestDifferentDataTypes(t *testing.T) {
	testData := map[string]interface{}{
		"string":     "hello",
		"int":        42,
		"float":      3.14,
		"bool":       true,
		"null":       nil,
		"array":      []interface{}{1, "two", 3.0},
		"emptyArray": []interface{}{},
	}

	hash1, err := CalculateMetadataHash(testData)
	assert.NoError(t, err)
	hash2, err := CalculateMetadataHash(testData)
	assert.NoError(t, err)
	assert.Equal(t, hash1, hash2)
}

// TestDeepNesting verifies hashing works with deeply nested structures
func TestDeepNesting(t *testing.T) {
	testData := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]interface{}{
					"value": "deep",
				},
			},
		},
	}

	hash1, err := CalculateMetadataHash(testData)
	assert.NoError(t, err)
	hash2, err := CalculateMetadataHash(testData)
	assert.NoError(t, err)
	assert.Equal(t, hash1, hash2)
}

// TestOrderIndependence verifies that field order doesn't affect the hash
func TestOrderIndependence(t *testing.T) {
	data1 := map[string]interface{}{
		"a": 1,
		"b": 2,
		"c": 3,
	}

	data2 := map[string]interface{}{
		"c": 3,
		"a": 1,
		"b": 2,
	}

	hash1, err := CalculateMetadataHash(data1)
	assert.NoError(t, err)
	hash2, err := CalculateMetadataHash(data2)
	assert.NoError(t, err)
	assert.Equal(t, hash1, hash2)
}

// TestArrayOrderDependence verifies that array order does affect the hash
func TestArrayOrderDependence(t *testing.T) {
	data1 := map[string]interface{}{
		"array": []interface{}{1, 2, 3},
	}

	data2 := map[string]interface{}{
		"array": []interface{}{3, 2, 1},
	}

	hash1, err := CalculateMetadataHash(data1)
	assert.NoError(t, err)
	hash2, err := CalculateMetadataHash(data2)
	assert.NoError(t, err)
	assert.NotEqual(t, hash1, hash2)
}

// TestSpecialCharacters verifies handling of special characters in keys and values
func TestSpecialCharacters(t *testing.T) {
	testData := map[string]interface{}{
		"special chars": "!@#$%^&*()",
		"unicode":       "Hello, 世界",
		"emoji":         "👋 🌍",
		"quotes":        "\"quoted\"",
		"newlines":      "line1\nline2",
		"tabs":          "tab\there",
	}

	hash1, err := CalculateMetadataHash(testData)
	assert.NoError(t, err)
	hash2, err := CalculateMetadataHash(testData)
	assert.NoError(t, err)
	assert.Equal(t, hash1, hash2)
}

// TestLargeValues verifies handling of large values
func TestLargeValues(t *testing.T) {
	// Create a large string
	largeString := make([]byte, 1000000)
	for i := range largeString {
		largeString[i] = 'a'
	}

	testData := map[string]interface{}{
		"large": string(largeString),
	}

	hash1, err := CalculateMetadataHash(testData)
	assert.NoError(t, err)
	hash2, err := CalculateMetadataHash(testData)
	assert.NoError(t, err)
	assert.Equal(t, hash1, hash2)
}

// TestDistinctHashes verifies that different data produces different hashes
func TestDistinctHashes(t *testing.T) {
	data1 := map[string]interface{}{"key": "value1"}
	data2 := map[string]interface{}{"key": "value2"}

	hash1, err := CalculateMetadataHash(data1)
	assert.NoError(t, err)
	hash2, err := CalculateMetadataHash(data2)
	assert.NoError(t, err)
	assert.NotEqual(t, hash1, hash2)
}

// TestNilMap verifies handling of nil map
func TestNilMap(t *testing.T) {
	var testData map[string]interface{}
	hash, err := CalculateMetadataHash(testData)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
}

// TestNumericPrecision verifies that numeric precision is maintained
func TestNumericPrecision(t *testing.T) {
	testData := map[string]interface{}{
		"integer":       123456789,
		"large_integer": 9223372036854775807, // max int64
		"float":         123.456789,
		"scientific":    1.23456789e+08,
		"small_decimal": 0.0000000123456789,
	}

	hash1, err := CalculateMetadataHash(testData)
	assert.NoError(t, err)
	hash2, err := CalculateMetadataHash(testData)
	assert.NoError(t, err)
	assert.Equal(t, hash1, hash2)
}
