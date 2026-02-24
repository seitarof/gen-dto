package cli

// Config stores CLI options for a single generation run.
type Config struct {
	SrcType      string
	SrcPath      string
	DstType      string
	DstPath      string
	Filename     string
	FuncName     string
	IgnoreFields []string
	ShowVersion  bool
}

// OutputFilename returns destination file path for generator layer.
func (c *Config) OutputFilename() string {
	return c.Filename
}
