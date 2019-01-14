package conf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/mitchellh/mapstructure"
	"github.com/molecule-man/stack-assembly/depgraph"
	"github.com/molecule-man/stack-assembly/stackassembly"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

type settingsConfig struct {
	Aws struct {
		Region   string
		Profile  string
		Endpoint string
	}
}

// Config is a struct holding stacks configurations
type Config struct {
	Settings settingsConfig

	Parameters map[string]string
	Stacks     map[string]StackConfig

	Hooks struct {
		Pre        stackassembly.HookCmds
		Post       stackassembly.HookCmds
		PreSync    stackassembly.HookCmds
		PostSync   stackassembly.HookCmds
		PreCreate  stackassembly.HookCmds
		PostCreate stackassembly.HookCmds
		PreUpdate  stackassembly.HookCmds
		PostUpdate stackassembly.HookCmds
	}
}

type StackConfig struct {
	Name       string
	Path       string
	Parameters map[string]string
	Tags       map[string]string
	DependsOn  []string
	Blocked    []string
	Hooks      struct {
		PreSync    stackassembly.HookCmds
		PostSync   stackassembly.HookCmds
		PreCreate  stackassembly.HookCmds
		PostCreate stackassembly.HookCmds
		PreUpdate  stackassembly.HookCmds
		PostUpdate stackassembly.HookCmds
	}
}

func (cfg Config) StackConfigsSortedByExecOrder() ([]StackConfig, error) {
	stackCfgs := make([]StackConfig, len(cfg.Stacks))
	dg := depgraph.DepGraph{}

	for id, stackCfg := range cfg.Stacks {
		dg.Add(id, stackCfg.DependsOn)
	}

	orderedIds, err := dg.Resolve()
	if err != nil {
		return stackCfgs, err
	}

	for i, id := range orderedIds {
		stackCfgs[i] = cfg.Stacks[id]
	}
	return stackCfgs, nil
}

func (cfg Config) ChangeSets() ([]*stackassembly.ChangeSet, error) {
	chSets := make([]*stackassembly.ChangeSet, len(cfg.Stacks))

	ss, err := cfg.StackConfigsSortedByExecOrder()
	if err != nil {
		return chSets, err
	}

	for i, s := range ss {
		chSets[i], err = cfg.ChangeSetFromStackConfig(s)
		if err != nil {
			return chSets, err
		}
	}

	return chSets, nil
}

func (cfg Config) ChangeSetFromStackConfig(stackCfg StackConfig) (*stackassembly.ChangeSet, error) {
	bodyBytes, err := ioutil.ReadFile(stackCfg.Path)
	if err != nil {
		return nil, err
	}

	body := ""
	data := struct{ Params map[string]string }{}
	data.Params = stackCfg.Parameters
	err = parseTpl(&body, string(bodyBytes), data)

	return stackassembly.NewStack(Cf(cfg), stackCfg.Name).
		ChangeSet(body).
		WithParameters(stackCfg.Parameters).
		WithTags(stackCfg.Tags), err
}

var cf *cloudformation.CloudFormation

func Cf(cfg Config) *cloudformation.CloudFormation {
	if cf != nil {
		return cf
	}

	opts := session.Options{}

	if cfg.Settings.Aws.Profile != "" {
		opts.Profile = cfg.Settings.Aws.Profile
	}

	awsCfg := aws.Config{}
	awsCfg.MaxRetries = aws.Int(7)

	if cfg.Settings.Aws.Region != "" {
		awsCfg.Region = aws.String(cfg.Settings.Aws.Region)
	}

	awsCfg.Endpoint = aws.String(cfg.Settings.Aws.Endpoint)

	httpClient := http.Client{
		Timeout: 2 * time.Second,
	}
	awsCfg.HTTPClient = &httpClient

	opts.Config = awsCfg

	sess := session.Must(session.NewSessionWithOptions(opts))

	return cloudformation.New(sess)
}

func LoadConfig(cfgFiles []string) (Config, error) {
	mainConfig := Config{}

	if len(cfgFiles) == 0 {
		if _, err := os.Stat("Stack-assembly.toml"); err == nil {
			cfgFiles = []string{"Stack-assembly.toml"}
		}
	}

	mainRawCfg := make(map[string]interface{})
	for _, cf := range cfgFiles {
		extraRawCfg := make(map[string]interface{})
		if err := parseFile(cf, &extraRawCfg); err != nil {
			return mainConfig, fmt.Errorf("error occured while parsing config file %s: %v", cf, err)
		}
		merged := merge(mainRawCfg, extraRawCfg)
		mainRawCfg = merged.(map[string]interface{})
	}

	config := mapstructure.DecoderConfig{
		ErrorUnused: true,
		Result:      &mainConfig,
	}

	decoder, err := mapstructure.NewDecoder(&config)
	if err != nil {
		return mainConfig, err
	}

	err = decoder.Decode(mainRawCfg)
	if err != nil {
		return mainConfig, err
	}

	err = applyTemplating(&mainConfig)
	if err != nil {
		return mainConfig, err
	}

	return mainConfig, initEnvSettings(&mainConfig.Settings)
}

func initEnvSettings(settings *settingsConfig) error {
	viper.SetEnvPrefix("STAS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetConfigType("json")
	buf := bytes.Buffer{}
	enc := json.NewEncoder(&buf)

	if err := enc.Encode(settings); err != nil {
		return err
	}

	if err := viper.ReadConfig(&buf); err != nil {
		return err
	}

	return viper.Unmarshal(settings)
}

func parseFile(filename string, cfg *map[string]interface{}) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".yaml", ".yml":
		d := yaml.NewDecoder(f)
		return d.Decode(cfg)
	case ".json":
		d := json.NewDecoder(f)
		return d.Decode(cfg)
	case ".toml":
		_, err := toml.DecodeReader(f, cfg)
		return err
	}

	return fmt.Errorf("extension %s is not supported", ext)
}

func merge(x1, x2 interface{}) interface{} {
	x1 = normalizeRawCfgEntry(x1)
	x2 = normalizeRawCfgEntry(x2)
	switch x1 := x1.(type) {
	case map[string]interface{}:
		return mergeMaps(x1, x2)
	case []interface{}:
		x2, ok := x2.([]interface{})
		if !ok {
			return x1
		}
		return x2
	case nil:
		x2, ok := x2.(map[string]interface{})
		if ok {
			return x2
		}
		return x1
	}
	return x2
}

func mergeMaps(x1 map[string]interface{}, i2 interface{}) interface{} {
	x2, ok := i2.(map[string]interface{})
	if !ok {
		return x1
	}
	for k, v2 := range x2 {
		if v1, ok := x1[k]; ok {
			x1[k] = merge(v1, v2)
		} else {
			x1[k] = v2
		}
	}
	return x1
}

func normalizeRawCfgEntry(src interface{}) interface{} {
	x, ok := src.(map[interface{}]interface{})
	if !ok {
		return src
	}
	trg := map[string]interface{}{}
	for k, v := range x {
		trg[fmt.Sprintf("%v", k)] = v
	}
	return trg
}
