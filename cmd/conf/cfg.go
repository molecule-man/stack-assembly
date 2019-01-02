package conf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/mitchellh/mapstructure"
	"github.com/molecule-man/stack-assembly/awsprov"
	"github.com/molecule-man/stack-assembly/stackassembly"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

func Aws(cfg stackassembly.Config) *awsprov.AwsProvider {
	opts := session.Options{}

	if cfg.Settings.Aws.Profile != "" {
		opts.Profile = cfg.Settings.Aws.Profile
	}

	awsCfg := aws.Config{}
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

	return awsprov.New(sess)
}

func LoadConfig(cfgFiles []string) (stackassembly.Config, error) {
	mainConfig := stackassembly.Config{}

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

	viper.SetEnvPrefix("STAS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetConfigType("json")
	buf := bytes.Buffer{}
	enc := json.NewEncoder(&buf)
	err = enc.Encode(mainConfig.Settings)
	if err != nil {
		return mainConfig, err
	}

	err = viper.ReadConfig(&buf)
	if err != nil {
		return mainConfig, err
	}

	if err := viper.Unmarshal(&mainConfig.Settings); err != nil {
		return mainConfig, err
	}

	return mainConfig, nil
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
		x2, ok := x2.(map[string]interface{})
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
	case []interface{}:
		x2, ok := x2.([]interface{})
		if !ok {
			return x1
		}
		// for i := range x2 {
		// 	x1 = append(x1, x2[i])
		// }
		// return x1
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
