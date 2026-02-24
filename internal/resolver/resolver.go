package resolver

import (
	"github.com/seitarof/gen-dto/internal/matcher"
	"github.com/seitarof/gen-dto/internal/parser"
)

// Resolver resolves field conversion strategies.
type Resolver interface {
	Resolve(pairs []matcher.FieldPair, structPairs []matcher.StructPair) []ConversionPlan
}

// Rule tries to generate a conversion plan for one field pair.
type Rule interface {
	Name() string
	Try(src, dst parser.FieldInfo) (ConversionPlan, bool)
}

type resolverImpl struct {
	rules     []Rule
	nestedSet NestedSet
}

// New builds resolver with rule chain.
func New(rules ...Rule) Resolver {
	return &resolverImpl{rules: rules}
}

func (r *resolverImpl) Resolve(
	pairs []matcher.FieldPair,
	structPairs []matcher.StructPair,
) []ConversionPlan {
	r.nestedSet = buildNestedSet(r.nestedSet, structPairs)
	for _, rule := range r.rules {
		if aware, ok := rule.(NestedAware); ok {
			aware.SetNestedSet(r.nestedSet)
		}
	}

	plans := make([]ConversionPlan, 0, len(pairs))
	for _, p := range pairs {
		plans = append(plans, r.resolveOne(p))
	}
	return plans
}

func (r *resolverImpl) resolveOne(pair matcher.FieldPair) ConversionPlan {
	for _, rule := range r.rules {
		if plan, ok := rule.Try(pair.SrcField, pair.DstField); ok {
			return plan
		}
	}
	return ConversionPlan{
		SrcField: pair.SrcField,
		DstField: pair.DstField,
		Strategy: StrategySkip,
	}
}

func buildNestedSet(reuse NestedSet, structPairs []matcher.StructPair) NestedSet {
	if reuse == nil {
		reuse = make(NestedSet, len(structPairs))
	} else {
		clear(reuse)
	}
	for _, pair := range structPairs {
		reuse[NestedPairKey{
			SrcPkgPath: pair.Src.PkgPath,
			SrcName:    pair.Src.Name,
			DstPkgPath: pair.Dst.PkgPath,
			DstName:    pair.Dst.Name,
		}] = struct{}{}
	}
	return reuse
}
