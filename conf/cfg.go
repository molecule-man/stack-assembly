package conf

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/mitchellh/mapstructure"
	"github.com/molecule-man/stack-assembly/aws"
	"github.com/molecule-man/stack-assembly/awscf"
	"github.com/molecule-man/stack-assembly/depgraph"
	yaml "gopkg.in/yaml.v2"
)

type settingsConfig struct {
	Aws        aws.Config
	S3Settings aws.S3Settings
}

// Config is a struct holding stacks configurations.
type Config struct {
	Name       string
	Path       string
	Body       string
	URL        string `json:",omitempty" yaml:",omitempty" toml:",omitempty"`
	Parameters map[string]string
	Tags       map[string]string `json:",omitempty" yaml:",omitempty" toml:",omitempty"`
	DependsOn  []string          `json:",omitempty" yaml:",omitempty" toml:",omitempty"`
	Blocked    []string          `json:",omitempty" yaml:",omitempty" toml:",omitempty"`
	Hooks      struct {
		Pre        HookCmds `json:",omitempty" yaml:",omitempty" toml:",omitempty"`
		Post       HookCmds `json:",omitempty" yaml:",omitempty" toml:",omitempty"`
		PreCreate  HookCmds `json:",omitempty" yaml:",omitempty" toml:",omitempty"`
		PostCreate HookCmds `json:",omitempty" yaml:",omitempty" toml:",omitempty"`
		PreUpdate  HookCmds `json:",omitempty" yaml:",omitempty" toml:",omitempty"`
		PostUpdate HookCmds `json:",omitempty" yaml:",omitempty" toml:",omitempty"`
	} `json:",omitempty" yaml:",omitempty" toml:",omitempty"`

	RollbackConfiguration *cloudformation.RollbackConfiguration `json:",omitempty" yaml:",omitempty" toml:",omitempty"`
	UsePreviousTemplate   bool                                  `json:",omitempty" yaml:",omitempty" toml:",omitempty"`

	RoleARN          string         `json:",omitempty" yaml:",omitempty" toml:",omitempty"`
	ClientToken      string         `json:",omitempty" yaml:",omitempty" toml:",omitempty"`
	NotificationARNs []string       `json:",omitempty" yaml:",omitempty" toml:",omitempty"`
	Capabilities     []string       `json:",omitempty" yaml:",omitempty" toml:",omitempty"`
	ResourceTypes    []string       `json:",omitempty" yaml:",omitempty" toml:",omitempty"`
	Settings         settingsConfig `json:",omitempty" yaml:",omitempty" toml:",omitempty"`

	Stacks map[string]Config `json:",omitempty" yaml:",omitempty" toml:",omitempty"`

	aws AwsProv
}

func (cfg Config) StackConfigsSortedByExecOrder() ([]Config, error) {
	stackCfgs := make([]Config, len(cfg.Stacks))
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
		chSets[i] = s.ChangeSet()
	}

	return chSets, nil
}

func (cfg Config) Stack() *awscf.Stack {
	prov := cfg.aws.Must(cfg.Settings.Aws)

	return awscf.NewStack(
		cfg.Name,
		prov.CF,
		aws.NewS3Uploader(prov.S3UploadManager, prov.S3, cfg.Settings.S3Settings),
	)
}

func (cfg Config) ChangeSet() *awscf.ChangeSet {
	return cfg.Stack().
		ChangeSet(cfg.Body).
		WithTemplateURL(cfg.URL).
		WithParameters(cfg.Parameters).
		WithTags(cfg.Tags).
		WithRollback(cfg.RollbackConfiguration).
		WithCapabilities(cfg.Capabilities).
		WithRoleARN(cfg.RoleARN).
		WithClientToken(cfg.ClientToken).
		WithNotificationARNs(cfg.NotificationARNs).
		WithUsePrevTpl(cfg.UsePreviousTemplate).
		WithResourceTypes(cfg.ResourceTypes)
}

func (cfg *Config) initAwsSettings() {
	for i, s := range cfg.Stacks {
		s.Settings.Aws.Merge(cfg.Settings.Aws)
		s.aws = cfg.aws

		s.Settings.S3Settings.Merge(cfg.Settings.S3Settings)

		s.initAwsSettings()

		cfg.Stacks[i] = s
	}
}

type AwsProv interface {
	Must(cfg aws.Config) *aws.AWS
	New(cfg aws.Config) (*aws.AWS, error)
}

func NewLoader(fs FileSystem, awsProvider AwsProv) *Loader {
	return &Loader{fs, awsProvider}
}

type Loader struct {
	fs  FileSystem
	aws AwsProv
}

func (l Loader) LoadConfig(cfgFiles []string, cfg *Config) error {
	err := l.decodeConfigs(cfg, cfgFiles)
	if err != nil {
		return err
	}

	return l.InitConfig(cfg)
}

func (l Loader) InitConfig(cfg *Config) error {
	cfg.aws = l.aws

	err := l.parseBodies(cfg)
	if err != nil {
		return err
	}

	cfg.initAwsSettings()

	return l.applyTemplating(cfg)
}

func (l Loader) parseBodies(stackCfg *Config) error {
	for i, nestedStack := range stackCfg.Stacks {
		nestedStack := nestedStack

		err := l.parseBodies(&nestedStack)
		if err != nil {
			return err
		}

		stackCfg.Stacks[i] = nestedStack
	}

	switch {
	case stackCfg.Body != "":
		return nil
	case stackCfg.Path == "":
		return nil
	}

	f, err := l.fs.Open(stackCfg.Path)
	if err != nil {
		return err
	}

	defer f.Close()

	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	stackCfg.Body = string(buf)

	return nil
}

func (l Loader) decodeConfigs(mainConfig *Config, cfgFiles []string) error {
	if len(cfgFiles) == 0 {
		tryCfgFiles := []string{
			"stack-assembly.yaml",
			"stack-assembly.yml",
			"stack-assembly.toml",
			"stack-assembly.json",
		}
		for _, f := range tryCfgFiles {
			if _, err := l.fs.Stat(f); err == nil {
				cfgFiles = []string{f}
				break
			}
		}
	}

	mainRawCfg := make(map[string]interface{})

	for _, cf := range cfgFiles {
		extraRawCfg := make(map[string]interface{})
		if err := l.parseFile(cf, &extraRawCfg); err != nil {
			return fmt.Errorf("error occurred while parsing config file %s: %w", cf, err)
		}

		merged := merge(mainRawCfg, extraRawCfg)
		mainRawCfg = merged.(map[string]interface{})
	}

	if d, ok := mainRawCfg["definitions"]; ok {
		delete(mainRawCfg, "definitions")

		d = normalizeRawCfgEntry(d)

		definitions, ok := d.(map[string]interface{})

		if !ok {
			return errors.New("error occurred while parsing config: `definitions` should be map")
		}

		if err := inheritDefinitions(&mainRawCfg, definitions); err != nil {
			return fmt.Errorf("error occurred while parsing config: %w", err)
		}
	}

	config := mapstructure.DecoderConfig{
		ErrorUnused: true,
		Result:      mainConfig,
	}

	decoder, err := mapstructure.NewDecoder(&config)
	if err != nil {
		return err
	}

	return decoder.Decode(mainRawCfg)
}

func inheritDefinitions(cfg *map[string]interface{}, definitions map[string]interface{}) error {
	if basedOn, ok := (*cfg)["$basedOn"]; ok {
		basedOnValue, ok := basedOn.(string)

		delete(*cfg, "$basedOn")

		if !ok {
			return errors.New("value of $basedOn must be string")
		}

		def, ok := definitions[basedOnValue]

		if !ok {
			return fmt.Errorf("definition for %s doesn't exist", basedOnValue)
		}

		merged := merge(def, *cfg)
		*cfg = merged.(map[string]interface{})
	}

	for k, v := range *cfg {
		v = normalizeRawCfgEntry(v)
		if m, ok := v.(map[string]interface{}); ok {
			err := inheritDefinitions(&m, definitions)

			if err != nil {
				return err
			}

			(*cfg)[k] = m
		}
	}

	return nil
}

func (l Loader) parseFile(filename string, cfg *map[string]interface{}) error {
	f, err := l.fs.Open(filename)
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
