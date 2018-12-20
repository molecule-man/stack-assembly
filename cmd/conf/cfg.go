package conf

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
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

	if tc.Parameters == nil {
		tc.Parameters = make(map[string]string)
	}
	for k, v := range otherTpl.Parameters {
		tc.Parameters[k] = v
	}
}

func LoadConfig(cfgFiles []string) Config {
	mainConfig := Config{}

	if len(cfgFiles) == 0 {
		if _, err := os.Stat("Stack-assembly.toml"); err == nil {
			cfgFiles = []string{"Stack-assembly.toml"}
		}
	}

	for _, cf := range cfgFiles {
		extraCfg := Config{}
		if err := fromFile(cf, &extraCfg); err != nil {
			log.Fatalf("error occured while parsing config file %s: %v", cf, err)
		}
		mainConfig.merge(extraCfg)
	}

	return mainConfig
}

func fromFile(filename string, cfg *Config) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".yaml", ".yml":
		return yaml.NewDecoder(f).Decode(cfg)
	case ".json":
		return json.NewDecoder(f).Decode(cfg)
	case ".toml":
		_, err := toml.DecodeReader(f, cfg)
		return err
	}

	return fmt.Errorf("extension %s is not supported", ext)
}
