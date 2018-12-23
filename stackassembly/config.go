package stackassembly

// StackTemplate encapsulates information about stack template
type StackTemplate struct {
	Name       string
	Path       string
	Body       string
	Parameters map[string]string
	DependsOn  []string
	Blocked    []string
}

// Config is a struct holding templates configurations
type Config struct {
	Parameters map[string]string
	Templates  map[string]StackTemplate
}
