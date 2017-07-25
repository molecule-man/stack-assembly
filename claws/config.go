package claws

type (
	Config struct {
		Parameters map[string]string
		Templates  map[string]TemplateConfig
	}

	TemplateConfig struct {
		Path string
		Name *string
	}
)

func (c *Config) Merge(otherConfig Config) {
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
		c.Templates[k] = tc
	}
}
