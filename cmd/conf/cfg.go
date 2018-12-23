package conf

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/molecule-man/stack-assembly/awsprov"
	"github.com/molecule-man/stack-assembly/cli"
	"github.com/molecule-man/stack-assembly/stackassembly"
	yaml "gopkg.in/yaml.v2"
)

type (
	// Config is a struct holding templates configurations
	Config struct {
		Parameters map[string]string
		Templates  map[string]TemplateConfig
	}

	// TemplateConfig is a configuration of a stack template
	TemplateConfig struct {
		Path       string
		Name       string
		Parameters map[string]string
		DependsOn  []string
		Blocked    []string
	}
)

// merge merges otherConfig into this config
func (c *Config) merge(otherConfig Config) {
	if c.Parameters == nil {
		c.Parameters = make(map[string]string)
	}
	for k, v := range otherConfig.Parameters {
		c.Parameters[k] = v
	}

	if c.Templates == nil {
		c.Templates = make(map[string]TemplateConfig)
	}
	for k, tc := range otherConfig.Templates {
		t := c.Templates[k]
		t.merge(tc)
		c.Templates[k] = t
	}
}

func (tc *TemplateConfig) merge(otherTpl TemplateConfig) {
	if otherTpl.Path != "" {
		tc.Path = otherTpl.Path
	}

	if otherTpl.Name != "" {
		tc.Name = otherTpl.Name
	}

	if otherTpl.DependsOn != nil {
		tc.DependsOn = otherTpl.DependsOn
	}

	if otherTpl.Blocked != nil {
		tc.Blocked = otherTpl.Blocked
	}

	if tc.Parameters == nil {
		tc.Parameters = make(map[string]string)
	}
	for k, v := range otherTpl.Parameters {
		tc.Parameters[k] = v
	}
}

func InitStasService(cfg Config) stackassembly.Service {
	awsConfig := awsprov.Config{}

	if profile, ok := cfg.Parameters["profile"]; ok && profile != "" {
		awsConfig.Profile = profile
	}

	if region, ok := cfg.Parameters["region"]; ok && region != "" {
		awsConfig.Region = region
	}

	return stackassembly.Service{
		Approver:      &cli.Approval{},
		Log:           log.New(os.Stderr, "", log.LstdFlags),
		CloudProvider: awsprov.New(awsConfig),
	}
}

func LoadConfig(cfgFiles []string) (Config, error) {
	mainConfig := Config{}

	if len(cfgFiles) == 0 {
		if _, err := os.Stat("Stack-assembly.toml"); err == nil {
			cfgFiles = []string{"Stack-assembly.toml"}
		}
	}

	for _, cf := range cfgFiles {
		extraCfg := Config{}
		if err := parseFile(cf, &extraCfg); err != nil {
			return mainConfig, fmt.Errorf("error occured while parsing config file %s: %v", cf, err)
		}
		mainConfig.merge(extraCfg)
	}

	return mainConfig, nil
}

func parseFile(filename string, cfg *Config) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".yaml", ".yml":
		d := yaml.NewDecoder(f)
		d.SetStrict(true)
		return d.Decode(cfg)
	case ".json":
		d := json.NewDecoder(f)
		d.DisallowUnknownFields()
		return d.Decode(cfg)
	case ".toml":
		m, err := toml.DecodeReader(f, cfg)
		if err != nil {
			return err
		}

		unknownFields := m.Undecoded()
		if len(unknownFields) > 0 {
			return fmt.Errorf("the config contains unknown fields %v", unknownFields)
		}
		return nil
	}

	return fmt.Errorf("extension %s is not supported", ext)
}
