package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

// ParseArgs parses command line arguments into Config.
func ParseArgs(args []string) (*Config, error) {
	cfg := &Config{}
	var ignoreFieldsRaw string

	fs := pflag.NewFlagSet("gen-dto", pflag.ContinueOnError)
	fs.StringVarP(&cfg.SrcType, "src-type", "s", "", "source struct type")
	fs.StringVar(&cfg.SrcPath, "src-path", "", "source package path")
	fs.StringVarP(&cfg.DstType, "dst-type", "d", "", "destination struct type")
	fs.StringVar(&cfg.DstPath, "dst-path", "", "destination package path")
	fs.StringVarP(&cfg.Filename, "filename", "o", "", "output file name")
	fs.StringVar(&ignoreFieldsRaw, "ignore-fields", "", "comma-separated field names to ignore")
	fs.StringVar(&cfg.FuncName, "func-name", "", "converter function name for root type")
	fs.BoolVarP(&cfg.ShowVersion, "version", "v", false, "show version")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if cfg.ShowVersion {
		return cfg, nil
	}

	if strings.TrimSpace(cfg.SrcType) == "" {
		return nil, fmt.Errorf("--src-type is required")
	}
	if strings.TrimSpace(cfg.SrcPath) == "" {
		return nil, fmt.Errorf("--src-path is required")
	}
	if strings.TrimSpace(cfg.DstType) == "" {
		return nil, fmt.Errorf("--dst-type is required")
	}
	if strings.TrimSpace(cfg.DstPath) == "" {
		return nil, fmt.Errorf("--dst-path is required")
	}
	if strings.TrimSpace(cfg.Filename) == "" {
		return nil, fmt.Errorf("--filename is required")
	}

	cfg.IgnoreFields = splitCommaList(ignoreFieldsRaw)
	return cfg, nil
}

func splitCommaList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
