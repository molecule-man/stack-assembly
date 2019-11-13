package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeParams(t *testing.T) {
	testCases := []struct {
		input          []string
		expectedOutput []string
	}{{
		input:          []string{"aws", "--params", "key1=val1,key2=val2"},
		expectedOutput: []string{"aws", "--params", "key1=val1,key2=val2"},
	}, {
		input:          []string{"aws", "--params", "key1=val1", "key2=val2"},
		expectedOutput: []string{"aws", "--params", "key1=val1", "--params", "key2=val2"},
	}, {
		input:          []string{"aws", "--params", "foo", "bar", "buz", "--tags", "tag"},
		expectedOutput: []string{"aws", "--params", "foo", "--params", "bar", "--params", "buz", "--tags", "tag"},
	}, {
		input:          []string{"aws", "--params", "foo", "bar", "buz", "--tags", "tag1", "tag2"},
		expectedOutput: []string{"aws", "--params", "foo", "--params", "bar", "--params", "buz", "--tags", "tag1", "--tags", "tag2"},
	}, {
		input:          []string{"aws", "--profile", "foo", "bar", "--tags", "tag1", "tag2"},
		expectedOutput: []string{"aws", "--profile", "foo", "bar", "--tags", "tag1", "--tags", "tag2"},
	}, {
		input:          []string{"aws", "foo", "bar"},
		expectedOutput: []string{"aws", "foo", "bar"},
	}}

	for _, tc := range testCases {
		output := normalizeAwsParams([]string{"--params", "--tags"}, tc.input)
		assert.Equal(t, tc.expectedOutput, output, "input: %+v", tc.input)
	}
}
