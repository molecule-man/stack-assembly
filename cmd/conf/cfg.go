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

// merge merges otherConfig into this config
func merge(c *stackassembly.Config, otherConfig stackassembly.Config) {
	if c.Parameters == nil {
		c.Parameters = make(map[string]string)
	}
	for k, v := range otherConfig.Parameters {
		c.Parameters[k] = v
	}

	if c.Templates == nil {
		c.Templates = make(map[string]stackassembly.StackTemplate)
	}
	for k, tc := range otherConfig.Templates {
		t := c.Templates[k]
		mergeTemplates(&t, tc)
		c.Templates[k] = t
	}
}

func mergeTemplates(tc *stackassembly.StackTemplate, otherTpl stackassembly.StackTemplate) {
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

func InitStasService(cfg stackassembly.Config) stackassembly.Service {
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

func LoadConfig(cfgFiles []string) (stackassembly.Config, error) {
	mainConfig := stackassembly.Config{}

	if len(cfgFiles) == 0 {
		if _, err := os.Stat("Stack-assembly.toml"); err == nil {
			cfgFiles = []string{"Stack-assembly.toml"}
		}
	}

	for _, cf := range cfgFiles {
		extraCfg := stackassembly.Config{}
		if err := parseFile(cf, &extraCfg); err != nil {
			return mainConfig, fmt.Errorf("error occured while parsing config file %s: %v", cf, err)
		}
		merge(&mainConfig, extraCfg)
	}

	return mainConfig, nil
}

func parseFile(filename string, cfg *stackassembly.Config) error {
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
