package cli

import (
	"strings"
	"testing"
)

func TestTable(t *testing.T) {
	table := NewTable().
		Row().Cell("hello").Cell("World").
		Row().Cell("a").Cell("long long long long...").
		Row().Cell("b")

	expected := strings.Join([]string{
		"+-------+------------------------+",
		"| hello | World                  |",
		"| a     | long long long long... |",
		"| b     |                        |",
		"+-------+------------------------+\n",
	}, "\n")
	actual := table.Render()

	if expected != actual {
		t.Errorf("Rendered: \n%s\nExpected: \n%s\n", actual, expected)
	}
}

func TestTableWithHeader(t *testing.T) {
	table := NewTable().
		Header().Cell("HeaderCell1").Cell("HeaderCell2").
		Row().Cell("hello").Cell("World").
		Row().Cell("a").Cell("long long long long...")

	expected := strings.Join([]string{
		"+-------------+------------------------+",
		"| HeaderCell1 | HeaderCell2            |",
		"+-------------+------------------------+",
		"| hello       | World                  |",
		"| a           | long long long long... |",
		"+-------------+------------------------+\n",
	}, "\n")
	actual := table.Render()

	if expected != actual {
		t.Errorf("Rendered: \n%s\nExpected: \n%s\n", actual, expected)
	}
}
