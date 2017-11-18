package main

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
