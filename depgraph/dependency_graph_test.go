package depgraph

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolving(t *testing.T) {
	testCases := []struct {
		name string
		deps map[string][]string
	}{
		{"simple graph", map[string][]string{
			"5": {"2", "6"},
			"3": {"1"},
			"7": {"4", "5", "6"},
			"6": {"3"},
			"4": {"2"},
			"2": {"1"},
			"1": {},
		}},
		{"liniar", map[string][]string{
			"1": {},
			"2": {"1"},
			"3": {"2"},
			"4": {"3"},
		}},
		{"one node", map[string][]string{
			"1": {},
		}},
		{"no dependencies", map[string][]string{
			"1": {},
			"2": {},
			"3": {},
		}},
	}

	for _, tc := range testCases {
		tc := tc // pinning unpinned variable. See scopelint
		t.Run(tc.name, func(t *testing.T) {
			deps := tc.deps
			dg := DepGraph{}

			for id, dependsOn := range deps {
				dg.Add(id, dependsOn)
			}

			resolved, err := dg.Resolve()

			require.NoError(t, err)

			t.Run("Resolved result has all the provided nodes", func(t *testing.T) {
				expected := make([]string, 0, len(deps))
				for k := range deps {
					expected = append(expected, k)
				}
				sort.Strings(expected)

				actual := make([]string, len(resolved))
				copy(actual, resolved)
				sort.Strings(actual)

				assert.Equal(t, expected, actual)
			})

			t.Run("Resolved nodes are ordered correctly", func(t *testing.T) {
				processedNodes := make(map[string]bool)

				for _, id := range resolved {
					for _, dep := range deps[id] {
						assert.Containsf(t, processedNodes, dep, "The node %s depends on %s, which is not yet processed. Order: %v", id, dep, resolved)
					}
					processedNodes[id] = true
				}
			})
		})
	}
}

func TestCycles(t *testing.T) {
	testCases := []struct {
		name string
		deps map[string][]string
	}{
		{"one node", map[string][]string{
			"1": {"1"},
		}},
		{"two nodes", map[string][]string{
			"1": {"2"},
			"2": {"1"},
		}},
		{"three nodes", map[string][]string{
			"1": {"3"},
			"2": {"1"},
			"3": {"2"},
		}},
	}

	for _, tc := range testCases {
		tc := tc // pinning unpinned variable. See scopelint
		t.Run(tc.name, func(t *testing.T) {
			dg := DepGraph{}

			for id, dependsOn := range tc.deps {
				dg.Add(id, dependsOn)
			}

			_, err := dg.Resolve()
			assert.Error(t, err)
		})
	}
}

func TestInvalidInput(t *testing.T) {
	dg := DepGraph{}
	dg.Add("1", []string{})
	dg.Add("2", []string{"1", "3"})

	_, err := dg.Resolve()
	assert.Error(t, err)
}

func TestDuplications(t *testing.T) {
	dg := DepGraph{}
	dg.Add("1", []string{})
	dg.Add("2", []string{"1"})
	dg.Add("3", []string{"1", "2"})
	dg.Add("3", []string{"2"})
	dg.Add("3", []string{})

	resolved, err := dg.Resolve()

	require.NoError(t, err)
	assert.Equal(t, []string{"1", "2", "3"}, resolved)
}
