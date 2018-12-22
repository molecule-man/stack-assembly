package conf

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var expectedParsedConfig = Config{
	Parameters: map[string]string{
		"Param1": "val1",
		"param2": "val2",
	},
	Templates: map[string]TemplateConfig{
		"tpl1": {
			Path: "path",
			Parameters: map[string]string{
				"Param3": "val3",
				"param4": "val4",
			},
		},
		"Tpl2": {
			Name:       "name1",
			Parameters: map[string]string{},
			DependsOn:  []string{"sns1"},
			Blocked:    []string{},
		},
	},
}

func TestParseJSON(t *testing.T) {
	jsonContent := `
{
  "parameters": {
    "Param1": "val1",
    "param2": "val2"
  },
  "templates": {
    "tpl1": {
      "path": "path",
      "parameters": {
        "Param3": "val3",
        "param4": "val4"
      }
    },
    "Tpl2": {
      "name": "name1",
      "dependson": [ "sns1" ],
      "blocked": []
    }
  }
}`
	fpath, cleanup := makeTestFile(t, ".json", jsonContent)
	defer cleanup()

	actualConfig, err := LoadConfig([]string{fpath})
	require.NoError(t, err)
	assert.Equal(t, expectedParsedConfig, actualConfig)
}

func TestParseYAML(t *testing.T) {
	yamlContent := `
---
parameters:
  Param1: val1
  param2: val2
templates:
  tpl1:
    path: path
    parameters:
      Param3: val3
      param4: val4
  Tpl2:
    name: name1
    dependson:
      - sns1
    blocked: []
`
	fpath, cleanup := makeTestFile(t, ".yaml", yamlContent)
	defer cleanup()

	actualConfig, err := LoadConfig([]string{fpath})
	require.NoError(t, err)
	assert.Equal(t, expectedParsedConfig, actualConfig)
}

func TestParseTOML(t *testing.T) {
	tomlContent := `
[parameters]
Param1 = "val1"
param2 = "val2"

[templates]

[templates.tpl1]
path = "path"

[templates.tpl1.parameters]
Param3 = "val3"
param4 = "val4"

[templates.Tpl2]
name = "name1"
dependson = [
  "sns1"
]
blocked = []
`
	fpath, cleanup := makeTestFile(t, ".toml", tomlContent)
	defer cleanup()

	actualConfig, err := LoadConfig([]string{fpath})
	require.NoError(t, err)
	assert.Equal(t, expectedParsedConfig, actualConfig)
}

func TestMergeConfigs(t *testing.T) {
	cfg1 := `
parameters:
  Param1: val1
templates:
  tpl1:
    name: name1
    path: path1
    dependson:
      - tpl2
    blocked:
      - sns1
    parameters:
      Param3: val3
      param4: val4
  tpl2:
    path: path2
    name: name2`

	cfg2 := `
parameters:
  Param1: overwriten_val1
templates:
  tpl1:
    path: overwriten_path1
    dependson:
      - overwriten_tpl1
    blocked:
      - sns
    parameters:
      param4: overwriten_val4
      param5: overwriten_val5
  tpl2:
    blocked:
      - sns2`

	expected := Config{
		Parameters: map[string]string{
			"Param1": "overwriten_val1",
		},
		Templates: map[string]TemplateConfig{
			"tpl1": {
				Name: "name1",
				Path: "overwriten_path1",
				Parameters: map[string]string{
					"Param3": "val3",
					"param4": "overwriten_val4",
					"param5": "overwriten_val5",
				},
				DependsOn: []string{"overwriten_tpl1"},
				Blocked:   []string{"sns"},
			},
			"tpl2": {
				Name:       "name2",
				Path:       "path2",
				Parameters: map[string]string{},
				Blocked:    []string{"sns2"},
			},
		},
	}
	fpath1, cleanup1 := makeTestFile(t, ".yml", cfg1)
	defer cleanup1()

	fpath2, cleanup2 := makeTestFile(t, ".yml", cfg2)
	defer cleanup2()

	actual, err := LoadConfig([]string{fpath1, fpath2})
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func makeTestFile(t *testing.T, ext, content string) (string, func()) {
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	fpath := filepath.Join(os.TempDir(), "stastest_"+suffix+ext)
	err := ioutil.WriteFile(fpath, []byte(content), 0700)
	require.NoError(t, err)

	return fpath, func() {
		os.Remove(fpath)
	}
}
