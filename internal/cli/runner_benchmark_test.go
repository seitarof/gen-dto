package cli

import (
	"path/filepath"
	"testing"

	"github.com/seitarof/gen-dto/internal/generator"
	"github.com/seitarof/gen-dto/internal/matcher"
	"github.com/seitarof/gen-dto/internal/parser"
	"github.com/seitarof/gen-dto/internal/resolver"
)

func BenchmarkRunnerRun_EndToEnd(b *testing.B) {
	out := filepath.Join(b.TempDir(), "bidi_gen.go")

	runner := NewRunner(
		parser.New(),
		matcher.NewStructMatcher(),
		matcher.NewFieldMatcher(),
		resolver.New(resolver.DefaultRules()...),
		generator.New(generator.NewGoimportsFormatter(), generator.NewFileWriter()),
	)

	cfg := &Config{
		SrcType:  "User",
		SrcPath:  "github.com/seitarof/gen-dto/testdata/bidi/source",
		DstType:  "UserResponse",
		DstPath:  "github.com/seitarof/gen-dto/testdata/bidi/dest",
		Filename: out,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := runner.Run(cfg); err != nil {
			b.Fatal(err)
		}
	}
}
