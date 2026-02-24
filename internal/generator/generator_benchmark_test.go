package generator

import (
	"fmt"
	"testing"

	"github.com/seitarof/gen-dto/internal/parser"
	"github.com/seitarof/gen-dto/internal/resolver"
)

type passthroughFormatter struct{}

type discardWriter struct{}

func (passthroughFormatter) Format(_ string, src []byte) ([]byte, error) { return src, nil }

func (discardWriter) Write(_ string, _ []byte) error { return nil }

func BenchmarkGeneratorGenerate_TemplateOnly(b *testing.B) {
	g := New(passthroughFormatter{}, discardWriter{})
	cfg := testConfig{filename: "bench_gen.go"}
	plans := benchmarkPlans(8, 32)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := g.Generate(cfg, plans); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkPlans(structCount, fieldCount int) []resolver.StructConversionPlan {
	out := make([]resolver.StructConversionPlan, 0, structCount)
	for i := 0; i < structCount; i++ {
		fields := make([]resolver.ConversionPlan, 0, fieldCount)
		for j := 0; j < fieldCount; j++ {
			fields = append(fields, resolver.ConversionPlan{
				SrcField: parser.FieldInfo{Name: fmt.Sprintf("SrcField%d", j), AccessPath: fmt.Sprintf("SrcField%d", j)},
				DstField: parser.FieldInfo{Name: fmt.Sprintf("DstField%d", j), AccessPath: fmt.Sprintf("DstField%d", j)},
				Strategy: resolver.StrategyDirectAssign,
				Expression: fmt.Sprintf(
					"dst.DstField%d = src.SrcField%d",
					j,
					j,
				),
			})
		}
		out = append(out, resolver.StructConversionPlan{
			Src:      &parser.StructInfo{Name: fmt.Sprintf("SrcType%d", i), PkgName: "src", PkgPath: "example.com/src"},
			Dst:      &parser.StructInfo{Name: fmt.Sprintf("DstType%d", i), PkgName: "dto", PkgPath: "example.com/dto"},
			FuncName: fmt.Sprintf("ConvertSrcType%dToDstType%d", i, i),
			Plans:    fields,
		})
	}
	return out
}
