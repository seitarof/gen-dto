package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seitarof/gen-dto/internal/parser"
	"github.com/seitarof/gen-dto/internal/resolver"
)

type testConfig struct {
	filename string
}

func (c testConfig) OutputFilename() string { return c.filename }

func TestGenerate_WritesFile(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "user_conv_gen.go")

	g := New(NewGoimportsFormatter(), NewFileWriter())
	plans := []resolver.StructConversionPlan{
		{
			Src:      &parser.StructInfo{Name: "User", PkgName: "model", PkgPath: "example.com/model"},
			Dst:      &parser.StructInfo{Name: "UserDTO", PkgName: "dto", PkgPath: "example.com/dto"},
			FuncName: "ConvertUserToUserDTO",
			Plans: []resolver.ConversionPlan{
				{
					Strategy:   resolver.StrategyDirectAssign,
					Expression: "dst.ID = src.ID",
				},
				{
					Strategy:   resolver.StrategyDirectAssign,
					Expression: "dst.Name = src.Name",
				},
			},
		},
	}

	if err := g.Generate(testConfig{filename: filename}, plans); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	b, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(b)
	if !strings.Contains(got, "package model") {
		t.Fatalf("generated package should be source package: %s", got)
	}
	if !strings.Contains(got, "func ConvertUserToUserDTO") {
		t.Fatalf("generated function not found: %s", got)
	}
	if !strings.Contains(got, "src *User") {
		t.Fatalf("source type should be unqualified in source package: %s", got)
	}
	if !strings.Contains(got, "*dto.UserDTO") {
		t.Fatalf("destination type should be qualified when external: %s", got)
	}
	if !strings.Contains(got, "dst.ID = src.ID") {
		t.Fatalf("generated assignment not found: %s", got)
	}
	if strings.Contains(got, "dst.ID = src.ID\n\n\tdst.Name = src.Name") {
		t.Fatalf("unexpected blank line between assignment lines: %s", got)
	}
}

func TestGenerate_WritesSkipCommentForUnsupportedConversion(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "skip_conv_gen.go")

	g := New(NewGoimportsFormatter(), NewFileWriter())
	plans := []resolver.StructConversionPlan{
		{
			Src:      &parser.StructInfo{Name: "User", PkgName: "model", PkgPath: "example.com/model"},
			Dst:      &parser.StructInfo{Name: "UserDTO", PkgName: "dto", PkgPath: "example.com/dto"},
			FuncName: "ConvertUserToUserDTO",
			Plans: []resolver.ConversionPlan{
				{
					SrcField: parser.FieldInfo{Name: "Metadata", TypeStr: "map[string]any"},
					DstField: parser.FieldInfo{Name: "Metadata", TypeStr: "string"},
					Strategy: resolver.StrategySkip,
				},
			},
		},
	}

	if err := g.Generate(testConfig{filename: filename}, plans); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	b, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(b)
	if !strings.Contains(got, "// Metadata: //TODO: couldn't auto-generate") {
		t.Fatalf("skip comment not found: %s", got)
	}
}
