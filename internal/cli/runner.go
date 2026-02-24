package cli

import (
	"fmt"
	"go/types"
	"log"

	"github.com/seitarof/gen-dto/internal/generator"
	"github.com/seitarof/gen-dto/internal/matcher"
	"github.com/seitarof/gen-dto/internal/parser"
	"github.com/seitarof/gen-dto/internal/resolver"
)

// Runner orchestrates parser/matcher/resolver/generator layers.
type Runner interface {
	Run(cfg *Config) error
}

type runnerImpl struct {
	parser      parser.Parser
	structMatch matcher.StructMatcher
	fieldMatch  matcher.FieldMatcher
	resolver    resolver.Resolver
	generator   generator.Generator
}

// NewRunner creates a default runner implementation.
func NewRunner(
	p parser.Parser,
	sm matcher.StructMatcher,
	fm matcher.FieldMatcher,
	r resolver.Resolver,
	g generator.Generator,
) Runner {
	return &runnerImpl{
		parser:      p,
		structMatch: sm,
		fieldMatch:  fm,
		resolver:    r,
		generator:   g,
	}
}

// Run executes a single generation cycle.
func (r *runnerImpl) Run(cfg *Config) error {
	srcInfos, err := r.parser.ParseRecursive(cfg.SrcPath, cfg.SrcType)
	if err != nil {
		return fmt.Errorf("parse src: %w", err)
	}
	dstInfos, err := r.parser.ParseRecursive(cfg.DstPath, cfg.DstType)
	if err != nil {
		return fmt.Errorf("parse dst: %w", err)
	}

	forwardPairs := r.structMatch.MatchStructs(srcInfos, dstInfos)
	forwardPairs = ensureRootPair(cfg, srcInfos, dstInfos, forwardPairs)
	if len(forwardPairs) == 0 {
		return fmt.Errorf("no matching structs found between %q and %q", cfg.SrcType, cfg.DstType)
	}

	outputPkgPath := srcInfos[len(srcInfos)-1].PkgPath
	if root := findStructByName(srcInfos, cfg.SrcType); root != nil {
		outputPkgPath = root.PkgPath
	}

	allPlans := make([]resolver.StructConversionPlan, 0, len(forwardPairs)*2)
	allPlans = r.appendPlans(allPlans, forwardPairs, cfg, cfg.SrcType, cfg.DstType, cfg.FuncName, outputPkgPath)

	reversePairs := reverseStructPairs(forwardPairs)
	if len(reversePairs) > 0 {
		allPlans = r.appendPlans(allPlans, reversePairs, cfg, cfg.DstType, cfg.SrcType, "", outputPkgPath)
	}

	return r.generator.Generate(cfg, allPlans)
}

func (r *runnerImpl) appendPlans(
	dst []resolver.StructConversionPlan,
	structPairs []matcher.StructPair,
	cfg *Config,
	rootSrcType string,
	rootDstType string,
	rootFuncName string,
	outputPkgPath string,
) []resolver.StructConversionPlan {
	for _, sp := range structPairs {
		pairs := r.fieldMatch.Match(sp.Src, sp.Dst, cfg.IgnoreFields)
		pairs = normalizePairTypeStrings(pairs, outputPkgPath)
		plans := r.resolver.Resolve(pairs, structPairs)
		logSkippedFields(plans)

		funcName := resolver.DefaultConverterName(sp.Src.PkgPath, sp.Src.Name, sp.Dst.PkgPath, sp.Dst.Name)
		if sp.Src.Name == rootSrcType && sp.Dst.Name == rootDstType && rootFuncName != "" {
			funcName = rootFuncName
		}

		dst = append(dst, resolver.StructConversionPlan{
			Src:      sp.Src,
			Dst:      sp.Dst,
			FuncName: funcName,
			Plans:    plans,
		})
	}
	return dst
}

func normalizePairTypeStrings(pairs []matcher.FieldPair, outputPkgPath string) []matcher.FieldPair {
	if len(pairs) == 0 {
		return pairs
	}
	out := make([]matcher.FieldPair, 0, len(pairs))
	for _, p := range pairs {
		p.SrcField.TypeStr = renderTypeForOutputPackage(p.SrcField.Type, outputPkgPath, p.SrcField.TypeStr)
		p.DstField.TypeStr = renderTypeForOutputPackage(p.DstField.Type, outputPkgPath, p.DstField.TypeStr)
		out = append(out, p)
	}
	return out
}

func renderTypeForOutputPackage(t types.Type, outputPkgPath string, fallback string) string {
	if t == nil {
		return fallback
	}
	qualifier := func(pkg *types.Package) string {
		if pkg == nil {
			return ""
		}
		if outputPkgPath != "" && pkg.Path() == outputPkgPath {
			return ""
		}
		return pkg.Name()
	}
	return types.TypeString(t, qualifier)
}

func reverseStructPairs(forwardPairs []matcher.StructPair) []matcher.StructPair {
	if len(forwardPairs) == 0 {
		return nil
	}

	reverse := make([]matcher.StructPair, 0, len(forwardPairs))
	for _, p := range forwardPairs {
		if p.Src == nil || p.Dst == nil {
			continue
		}
		// Same type conversion is direction-agnostic; avoid duplicate generation.
		if p.Src.PkgPath == p.Dst.PkgPath && p.Src.Name == p.Dst.Name {
			continue
		}

		reverse = append(reverse, matcher.StructPair{Src: p.Dst, Dst: p.Src})
	}
	return reverse
}

func ensureRootPair(
	cfg *Config,
	srcInfos []*parser.StructInfo,
	dstInfos []*parser.StructInfo,
	pairs []matcher.StructPair,
) []matcher.StructPair {
	srcRoot := findStructByName(srcInfos, cfg.SrcType)
	dstRoot := findStructByName(dstInfos, cfg.DstType)
	if srcRoot == nil || dstRoot == nil {
		return pairs
	}

	rootFound := false
	for _, p := range pairs {
		if p.Src.Name == srcRoot.Name && p.Dst.Name == dstRoot.Name {
			rootFound = true
			break
		}
	}

	if !rootFound {
		pairs = append([]matcher.StructPair{{Src: srcRoot, Dst: dstRoot}}, pairs...)
		return dedupePairs(pairs)
	}
	return pairs
}

func dedupePairs(pairs []matcher.StructPair) []matcher.StructPair {
	seen := map[string]bool{}
	out := make([]matcher.StructPair, 0, len(pairs))
	for _, p := range pairs {
		key := p.Src.PkgPath + "." + p.Src.Name + "->" + p.Dst.PkgPath + "." + p.Dst.Name
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, p)
	}
	return out
}

func findStructByName(infos []*parser.StructInfo, name string) *parser.StructInfo {
	for _, info := range infos {
		if info.Name == name {
			return info
		}
	}
	return nil
}

func logSkippedFields(plans []resolver.ConversionPlan) {
	for _, plan := range plans {
		if plan.Strategy != resolver.StrategySkip {
			continue
		}
		log.Printf(
			"gen-dto: warning: field %q (%s) -> %q (%s): conversion not supported, skipped",
			plan.SrcField.Name,
			plan.SrcField.TypeStr,
			plan.DstField.Name,
			plan.DstField.TypeStr,
		)
	}
}
