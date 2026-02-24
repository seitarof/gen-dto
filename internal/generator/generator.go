package generator

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"

	"golang.org/x/tools/imports"

	"github.com/seitarof/gen-dto/internal/resolver"
)

//go:embed templates/*.go.tmpl
var templateFS embed.FS

// Generator generates converter code from conversion plans.
type Generator interface {
	Generate(cfg Config, plans []resolver.StructConversionPlan) error
}

// Config is the minimum config contract required by generator.
type Config interface {
	OutputFilename() string
}

// Formatter formats generated Go code and organizes imports.
type Formatter interface {
	Format(filename string, src []byte) ([]byte, error)
}

// FileWriter writes generated code to disk.
type FileWriter interface {
	Write(filename string, data []byte) error
}

type generatorImpl struct {
	formatter Formatter
	writer    FileWriter
	tmpl      *template.Template
}

type goimportsFormatter struct{}

type fileWriter struct{}

type templateData struct {
	Package     string
	Imports     []string
	Conversions []conversionTemplateData
}

type conversionTemplateData struct {
	FuncName string
	SrcType  string
	DstType  string
	Plans    []resolver.ConversionPlan
}

// New creates a code generator.
func New(f Formatter, w FileWriter) Generator {
	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"renderPlan": renderPlan,
	}).ParseFS(templateFS, "templates/*.go.tmpl"))
	return &generatorImpl{formatter: f, writer: w, tmpl: tmpl}
}

// NewGoimportsFormatter creates a formatter backed by goimports.
func NewGoimportsFormatter() Formatter {
	return &goimportsFormatter{}
}

// NewFileWriter creates a plain file writer.
func NewFileWriter() FileWriter {
	return &fileWriter{}
}

func (g *generatorImpl) Generate(cfg Config, plans []resolver.StructConversionPlan) error {
	if len(plans) == 0 {
		return fmt.Errorf("no conversion plans")
	}

	data := buildTemplateData(plans)
	var buf bytes.Buffer
	if err := g.tmpl.ExecuteTemplate(&buf, "convert.go.tmpl", data); err != nil {
		return fmt.Errorf("template: %w", err)
	}

	formatted, err := g.formatter.Format(cfg.OutputFilename(), buf.Bytes())
	if err != nil {
		return fmt.Errorf("format: %w", err)
	}
	if err := g.writer.Write(cfg.OutputFilename(), formatted); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

func (f *goimportsFormatter) Format(filename string, src []byte) ([]byte, error) {
	return imports.Process(filename, src, nil)
}

func (w *fileWriter) Write(filename string, data []byte) error {
	return os.WriteFile(filename, data, 0o644)
}

func buildTemplateData(plans []resolver.StructConversionPlan) templateData {
	pkgName := plans[0].Src.PkgName
	pkgPath := plans[0].Src.PkgPath
	importsSet := map[string]struct{}{}
	conversions := make([]conversionTemplateData, 0, len(plans))

	for _, p := range plans {
		srcType := p.Src.Name
		if p.Src.PkgPath != pkgPath {
			srcType = p.Src.PkgName + "." + p.Src.Name
			importsSet[p.Src.PkgPath] = struct{}{}
		}

		dstType := p.Dst.Name
		if p.Dst.PkgPath != pkgPath {
			importsSet[p.Dst.PkgPath] = struct{}{}
			dstType = p.Dst.PkgName + "." + p.Dst.Name
		}

		conversions = append(conversions, conversionTemplateData{
			FuncName: p.FuncName,
			SrcType:  srcType,
			DstType:  dstType,
			Plans:    p.Plans,
		})
	}

	importsList := make([]string, 0, len(importsSet))
	for path := range importsSet {
		importsList = append(importsList, path)
	}
	sort.Strings(importsList)

	return templateData{
		Package:     pkgName,
		Imports:     importsList,
		Conversions: conversions,
	}
}

func renderPlan(plan resolver.ConversionPlan) string {
	if plan.Strategy == resolver.StrategySkip {
		return renderSkipComment(plan)
	}

	snippet := strings.TrimSpace(plan.Expression)
	if snippet == "" {
		return ""
	}

	if !strings.Contains(snippet, "\n") {
		return "\t" + snippet + "\n"
	}

	var b strings.Builder
	remaining := snippet
	for {
		line, rest, found := strings.Cut(remaining, "\n")
		trimmed := strings.TrimRight(line, " ")
		if strings.TrimSpace(trimmed) != "" {
			b.WriteString("\t")
			b.WriteString(trimmed)
			b.WriteString("\n")
		}
		if !found {
			break
		}
		remaining = rest
	}
	return b.String()
}

func renderSkipComment(plan resolver.ConversionPlan) string {
	dstName := plan.DstField.Name
	if dstName == "" {
		dstName = plan.SrcField.Name
		if dstName == "" {
			dstName = "unknown"
		}
	}
	return "\t// " + dstName + ": //TODO: couldn't auto-generate\n"
}
