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

	if c.Stacks == nil {
		c.Stacks = make(map[string]stackassembly.StackConfig)
	}
	for k, otherStack := range otherConfig.Stacks {
		// TODO what if k is not in c.Stacks
		thisStack := c.Stacks[k]
		mergeStacks(&thisStack, otherStack)
		c.Stacks[k] = thisStack
	}
}

func mergeStacks(s *stackassembly.StackConfig, otherStack stackassembly.StackConfig) {
	if otherStack.Path != "" {
		s.Path = otherStack.Path
	}

	if otherStack.Name != "" {
		s.Name = otherStack.Name
	}

	if otherStack.DependsOn != nil {
		s.DependsOn = otherStack.DependsOn
	}

	if otherStack.Blocked != nil {
		s.Blocked = otherStack.Blocked
	}

	if s.Parameters == nil {
		s.Parameters = make(map[string]string)
	}
	for k, v := range otherStack.Parameters {
		s.Parameters[k] = v
	}
}

func Aws(cfg stackassembly.Config) *awsprov.AwsProvider {
	awsConfig := awsprov.Config{}

	if profile, ok := cfg.Parameters["profile"]; ok && profile != "" {
		awsConfig.Profile = profile
	}

	if region, ok := cfg.Parameters["region"]; ok && region != "" {
		awsConfig.Region = region
	}

	return awsprov.New(awsConfig)
}

func InitStasService(cfg stackassembly.Config) stackassembly.Service {
	return stackassembly.Service{
		Approver:      &cli.Approval{},
		Log:           log.New(os.Stderr, "", log.LstdFlags),
		CloudProvider: Aws(cfg),
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
