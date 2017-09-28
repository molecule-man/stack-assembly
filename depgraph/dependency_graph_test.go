package depgraph

import (
	"sort"
	"strings"
	"testing"
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
		t.Run(tc.name, func(t *testing.T) {
			deps := tc.deps
			dg := DepGraph{}

			for id, dependsOn := range deps {
				dg.Add(id, dependsOn)
			}

			resolved, err := dg.Resolve()

			if err != nil {
				t.Errorf("Resolution shouldn't cause an error. But the following error was produced: %v", err)
			}

			t.Run("Resolved result has all the provided nodes", func(t *testing.T) {
				expected := make([]string, 0, len(deps))
				for k := range deps {
					expected = append(expected, k)
				}
				sort.Strings(expected)

				actual := make([]string, len(resolved))
				copy(actual, resolved)
				sort.Strings(actual)

				if strings.Join(expected, ",") != strings.Join(actual, ",") {
					t.Errorf("Resolving was supposed to produce %v. Got %v", expected, actual)
				}
			})

			t.Run("Resolved nodes are ordered correctly", func(t *testing.T) {
				processedNodes := make(map[string]bool)

				for _, id := range resolved {
					for _, dep := range deps[id] {
						if _, processed := processedNodes[dep]; !processed {
							t.Errorf("The node %s depends on %s, which is not yet processed. Order: %v", id, dep, resolved)
						}
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
		t.Run(tc.name, func(t *testing.T) {
			dg := DepGraph{}

			for id, dependsOn := range tc.deps {
				dg.Add(id, dependsOn)
			}

			_, err := dg.Resolve()

			if err == nil {
				t.Error("Resolution of the cyclic graph should produce an error")
			}
		})
	}
}

func TestInvalidInput(t *testing.T) {
	dg := DepGraph{}
	dg.Add("1", []string{})
	dg.Add("2", []string{"1", "3"})

	_, err := dg.Resolve()

	if err == nil {
		t.Error("Invalid input hasn't produced an error")
	}
}

func TestDuplications(t *testing.T) {
	dg := DepGraph{}
	dg.Add("1", []string{})
	dg.Add("2", []string{"1"})
	dg.Add("3", []string{"1", "2"})
	dg.Add("3", []string{"2"})
	dg.Add("3", []string{})

	resolved, err := dg.Resolve()

	if err != nil {
		t.Errorf("Resolution shouldn't cause an error. But the following error was produced: %v", err)
	}

	if expected := "1,2,3"; expected != strings.Join(resolved, ",") {
		t.Errorf("Resolving was supposed to produce %s. Got %v", expected, resolved)
	}
}
