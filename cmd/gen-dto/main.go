package main

import (
	"fmt"
	"log"
	"os"

	"github.com/seitarof/gen-dto/internal/cli"
	"github.com/seitarof/gen-dto/internal/generator"
	"github.com/seitarof/gen-dto/internal/matcher"
	"github.com/seitarof/gen-dto/internal/parser"
	"github.com/seitarof/gen-dto/internal/resolver"
)

var version = "dev"

func main() {
	cfg, err := cli.ParseArgs(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
	if cfg.ShowVersion {
		fmt.Println(version)
		return
	}

	p := parser.New()
	sm := matcher.NewStructMatcher()
	fm := matcher.NewFieldMatcher()
	r := resolver.New(resolver.DefaultRules()...)
	f := generator.NewGoimportsFormatter()
	w := generator.NewFileWriter()
	g := generator.New(f, w)

	runner := cli.NewRunner(p, sm, fm, r, g)
	if err := runner.Run(cfg); err != nil {
		log.Fatal(err)
	}
}
