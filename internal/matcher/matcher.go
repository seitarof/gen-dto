package matcher

import (
	"slices"
	"strings"

	"github.com/seitarof/gen-dto/internal/parser"
)

// StructPair is a source/destination struct pair.
type StructPair struct {
	Src *parser.StructInfo
	Dst *parser.StructInfo
}

// FieldPair is a source/destination field pair.
type FieldPair struct {
	SrcField parser.FieldInfo
	DstField parser.FieldInfo
}

// StructMatcher matches source/destination structs.
type StructMatcher interface {
	MatchStructs(srcInfos, dstInfos []*parser.StructInfo) []StructPair
}

// FieldMatcher matches fields in a struct pair.
type FieldMatcher interface {
	Match(src, dst *parser.StructInfo, ignoreFields []string) []FieldPair
}

type structMatcherImpl struct{}

type fieldMatcherImpl struct{}

// NewStructMatcher returns default struct matcher.
func NewStructMatcher() StructMatcher {
	return &structMatcherImpl{}
}

// NewFieldMatcher returns default field matcher.
func NewFieldMatcher() FieldMatcher {
	return &fieldMatcherImpl{}
}

func (m *structMatcherImpl) MatchStructs(srcInfos, dstInfos []*parser.StructInfo) []StructPair {
	dstMap := make(map[string]*parser.StructInfo, len(dstInfos))
	for _, d := range dstInfos {
		dstMap[d.Name] = d
	}

	pairs := make([]StructPair, 0, len(srcInfos))
	for _, s := range srcInfos {
		if d, ok := dstMap[s.Name]; ok {
			pairs = append(pairs, StructPair{Src: s, Dst: d})
		}
	}

	// ParseRecursive returns leaf-first; reverse so root comes first.
	slices.Reverse(pairs)
	return pairs
}

func (m *fieldMatcherImpl) Match(src, dst *parser.StructInfo, ignoreFields []string) []FieldPair {
	ignoreSet := toIgnoreSet(ignoreFields)
	dstMap := make(map[string]parser.FieldInfo, len(dst.Fields))
	for _, f := range dst.Fields {
		dstMap[strings.ToLower(f.Name)] = f
	}

	pairs := make([]FieldPair, 0, len(src.Fields))
	for _, sf := range src.Fields {
		lower := strings.ToLower(sf.Name)
		if ignoreSet[lower] {
			continue
		}
		df, ok := dstMap[lower]
		if !ok {
			continue
		}
		pairs = append(pairs, FieldPair{SrcField: sf, DstField: df})
	}
	return pairs
}

func toIgnoreSet(ignoreFields []string) map[string]bool {
	set := make(map[string]bool, len(ignoreFields))
	for _, f := range ignoreFields {
		f = strings.TrimSpace(strings.ToLower(f))
		if f == "" {
			continue
		}
		set[f] = true
	}
	return set
}
