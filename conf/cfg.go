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
	"github.com/molecule-man/stack-assembly/awscf"
	"github.com/molecule-man/stack-assembly/depgraph"
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
		Pre        HookCmds
		Post       HookCmds
		PreSync    HookCmds
		PostSync   HookCmds
		PreCreate  HookCmds
		PostCreate HookCmds
		PreUpdate  HookCmds
		PostUpdate HookCmds
	}
}

type StackConfig struct {
	Name       string
	Path       string
	Body       string
	Parameters map[string]string
	Tags       map[string]string
	DependsOn  []string
	Blocked    []string
	Hooks      struct {
		PreSync    HookCmds
		PostSync   HookCmds
		PreCreate  HookCmds
		PostCreate HookCmds
		PreUpdate  HookCmds
		PostUpdate HookCmds
	}
	RollbackConfiguration *cloudformation.RollbackConfiguration
	Capabilities          []string
	Stacks                map[string]StackConfig
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

func (cfg Config) ChangeSets() ([]*awscf.ChangeSet, error) {
	chSets := make([]*awscf.ChangeSet, len(cfg.Stacks))

	ss, err := cfg.StackConfigsSortedByExecOrder()
	if err != nil {
		return chSets, err
	}

	for i, s := range ss {
		chSets[i] = cfg.ChangeSetFromStackConfig(s)
	}

	return chSets, nil
}

func (cfg Config) ChangeSetFromStackConfig(stackCfg StackConfig) *awscf.ChangeSet {
	return awscf.NewStack(Cf(cfg), stackCfg.Name).
		ChangeSet(stackCfg.Body).
		WithParameters(stackCfg.Parameters).
		WithTags(stackCfg.Tags).
		WithRollback(stackCfg.RollbackConfiguration).
		WithCapabilities(stackCfg.Capabilities)
}

var cf *cloudformation.CloudFormation

func Cf(cfg Config) *cloudformation.CloudFormation {
	if cf != nil {
		return cf
	}

	sess := session.Must(session.NewSessionWithOptions(awsOpts(cfg)))

	return cloudformation.New(sess)
}

func awsOpts(cfg Config) session.Options {
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

	return opts
}

func LoadConfig(cfgFiles []string) (Config, error) {
	cfg, err := decodeConfigs(cfgFiles)
	if err != nil {
		return cfg, err
	}

	for i, stackCfg := range cfg.Stacks {
		stackCfg := stackCfg
		berr := parseBodies(i, &stackCfg)
		if berr != nil {
			return cfg, berr
		}

		cfg.Stacks[i] = stackCfg
	}

	err = applyTemplating(&cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, initEnvSettings(&cfg.Settings)
}

func parseBodies(id string, stackCfg *StackConfig) error {
	for i, nestedStack := range stackCfg.Stacks {
		nestedStack := nestedStack
		err := parseBodies(i, &nestedStack)
		if err != nil {
			return err
		}
		stackCfg.Stacks[i] = nestedStack
	}

	switch {
	case stackCfg.Body != "":
		return nil
	case stackCfg.Path == "" && len(stackCfg.Stacks) == 0:
		return fmt.Errorf("not possible to parse config for stack %s. "+
			"Either \"path\", \"body\" or non-empty \"stacks\" should be provided", id)
	case stackCfg.Path == "":
		return nil
	}

	buf, err := ioutil.ReadFile(stackCfg.Path)
	if err != nil {
		return err
	}

	stackCfg.Body = string(buf)

	return nil
}

func decodeConfigs(cfgFiles []string) (Config, error) {
	mainConfig := Config{}

	if len(cfgFiles) == 0 {
		tryCfgFiles := []string{
			"stack-assembly.yaml",
			"stack-assembly.yml",
			"stack-assembly.toml",
			"stack-assembly.json",
		}
		for _, f := range tryCfgFiles {
			if _, err := os.Stat(f); err == nil {
				cfgFiles = []string{f}
				break
			}
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

	return mainConfig, decoder.Decode(mainRawCfg)
}

func initEnvSettings(settings *settingsConfig) error {
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
